package ordemservico

import (
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

var t0 = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func din(t *testing.T, c int64) valueobject.Dinheiro {
	t.Helper()
	d, err := valueobject.NovoDinheiro(c)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func qtd(t *testing.T, n int) valueobject.Quantidade {
	t.Helper()
	q, err := valueobject.NovaQuantidade(n)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// osEmDiagnostico cria uma OS em diagnóstico com R$ 200,00 de serviço e 2 peças de R$ 150,00
// (total do orçamento: R$ 500,00; mão de obra: R$ 200,00; valor do orçamento: R$ 80,00).
func osEmDiagnostico(t *testing.T) *OrdemDeServico {
	t.Helper()
	o, err := Nova("OS-1", "cli-1", "vei-1", din(t, 8000), t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := o.IniciarDiagnostico(t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := o.AdicionarServico("Troca de óleo", din(t, 20000)); err != nil {
		t.Fatal(err)
	}
	if err := o.AdicionarPeca("peca-1", "Filtro", din(t, 15000), qtd(t, 2)); err != nil {
		t.Fatal(err)
	}
	return o
}

func osAprovada(t *testing.T) *OrdemDeServico {
	t.Helper()
	o := osEmDiagnostico(t)
	if err := o.EnviarOrcamento(t0.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := o.Aprovar("cli-1", t0.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestNova(t *testing.T) {
	if _, err := Nova("", "c", "v", din(t, 0), t0); err != ErrIDObrigatorio {
		t.Fatalf("esperava ErrIDObrigatorio, veio %v", err)
	}
	o, err := Nova("OS-1", "c", "v", din(t, 0), t0)
	if err != nil || o.Status() != Recebida || o.Decisao() != Pendente || o.ID() != "OS-1" || o.ClienteID() != "c" || o.VeiculoID() != "v" {
		t.Fatalf("OS inicial inesperada: %+v (erro: %v)", o, err)
	}
}

func TestOrcamentoAutomatico(t *testing.T) {
	o := osEmDiagnostico(t)
	if got := o.TotalDoOrcamento().Centavos(); got != 50000 {
		t.Fatalf("total esperado 50000, veio %d", got)
	}
	if got := o.MaoDeObra().Centavos(); got != 20000 {
		t.Fatalf("mão de obra esperada 20000, veio %d", got)
	}
	// Recalcula quando um item muda (RN04).
	if err := o.AdicionarServico("Alinhamento", din(t, 10000)); err != nil {
		t.Fatal(err)
	}
	if got := o.TotalDoOrcamento().Centavos(); got != 60000 {
		t.Fatalf("total esperado 60000, veio %d", got)
	}
	if len(o.Servicos()) != 2 || len(o.Pecas()) != 1 {
		t.Fatal("itens não foram registrados")
	}
}

func TestItensValidacoes(t *testing.T) {
	o, _ := Nova("OS-1", "c", "v", din(t, 0), t0)
	if err := o.AdicionarServico("x", din(t, 100)); err != ErrItensSoNoDiagnostico {
		t.Fatalf("esperava ErrItensSoNoDiagnostico, veio %v", err)
	}
	if err := o.AdicionarPeca("p", "x", din(t, 100), qtd(t, 1)); err != ErrItensSoNoDiagnostico {
		t.Fatalf("esperava ErrItensSoNoDiagnostico, veio %v", err)
	}
	_ = o.IniciarDiagnostico(t0)
	if err := o.AdicionarServico("  ", din(t, 100)); err != ErrDescricaoVazia {
		t.Fatalf("esperava ErrDescricaoVazia, veio %v", err)
	}
	if err := o.AdicionarPeca("", "x", din(t, 100), qtd(t, 1)); err != ErrDescricaoVazia {
		t.Fatalf("esperava ErrDescricaoVazia, veio %v", err)
	}
	if err := o.AdicionarPeca("p", "x", din(t, 100), qtd(t, 0)); err != ErrQuantidadeInvalida {
		t.Fatalf("esperava ErrQuantidadeInvalida, veio %v", err)
	}
}

func TestOrcamentoVazioNaoEnvia(t *testing.T) {
	o, _ := Nova("OS-1", "c", "v", din(t, 0), t0)
	_ = o.IniciarDiagnostico(t0)
	if err := o.EnviarOrcamento(t0); err != ErrOrcamentoVazio {
		t.Fatalf("esperava ErrOrcamentoVazio, veio %v", err)
	}
}

func TestFluxoAprovadoCompleto(t *testing.T) {
	o := osAprovada(t)
	if o.Status() != EmExecucao || o.Decisao() != Aprovado || o.DecididoPor() != "cli-1" || o.DecididoEm().IsZero() {
		t.Fatalf("estado após aprovação inesperado: %s %s", o.Status(), o.Decisao())
	}
	// Itens ficam travados depois do envio (RN04).
	if err := o.AdicionarServico("x", din(t, 1)); err != ErrItensSoNoDiagnostico {
		t.Fatalf("itens deveriam estar travados, veio %v", err)
	}
	if err := o.Finalizar(t0.Add(63 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := o.Entregar(t0.Add(70 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if o.Status() != Entregue {
		t.Fatalf("esperava Entregue, veio %s", o.Status())
	}
	// 6 mudanças no histórico: Recebida + 5 transições.
	if len(o.Historico()) != 6 {
		t.Fatalf("histórico esperado com 6 registros, veio %d", len(o.Historico()))
	}
	if d, ok := o.TempoDeExecucao(); !ok || d != 60*time.Minute {
		t.Fatalf("tempo de execução esperado 60min, veio %v (ok=%v)", d, ok)
	}
}

func TestFluxoReprovado(t *testing.T) {
	o := osEmDiagnostico(t)
	_ = o.EnviarOrcamento(t0.Add(2 * time.Minute))
	if err := o.Reprovar("cli-1", t0.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if o.Status() != Finalizada || o.Decisao() != Reprovado {
		t.Fatalf("estado após reprovação inesperado: %s %s", o.Status(), o.Decisao())
	}
	if _, ok := o.TempoDeExecucao(); ok {
		t.Fatal("OS reprovada não tem tempo de execução")
	}
	if err := o.Entregar(t0.Add(10 * time.Minute)); err != nil || o.Status() != Entregue {
		t.Fatalf("reprovada deve poder ser entregue, veio %v / %s", err, o.Status())
	}
}

func TestSomenteODonoDecide(t *testing.T) {
	for _, nome := range []string{"aprovar", "reprovar"} {
		o := osEmDiagnostico(t)
		_ = o.EnviarOrcamento(t0)
		var err error
		if nome == "aprovar" {
			err = o.Aprovar("outro-cliente", t0)
		} else {
			err = o.Reprovar("administrador", t0)
		}
		if err != ErrSomenteDonoDecide {
			t.Fatalf("%s: esperava ErrSomenteDonoDecide, veio %v", nome, err)
		}
		if o.Status() != AguardandoAprovacao || o.Decisao() != Pendente {
			t.Fatalf("%s: a OS não deveria ter mudado", nome)
		}
	}
}

func TestTransicoesInvalidas(t *testing.T) {
	o, _ := Nova("OS-1", "c", "v", din(t, 0), t0)
	if err := o.Aprovar("c", t0); err != ErrTransicaoInvalida {
		t.Fatalf("aprovar sem orçamento enviado: esperava ErrTransicaoInvalida, veio %v", err)
	}
	if err := o.Finalizar(t0); err != ErrTransicaoInvalida {
		t.Fatalf("pular etapa: esperava ErrTransicaoInvalida, veio %v", err)
	}
	if err := o.Entregar(t0); err != ErrTransicaoInvalida {
		t.Fatalf("pular etapa: esperava ErrTransicaoInvalida, veio %v", err)
	}
	if err := o.EnviarOrcamento(t0); err != ErrOrcamentoVazio && err != ErrTransicaoInvalida {
		t.Fatalf("enviar fora do diagnóstico deve falhar, veio %v", err)
	}
	if err := o.Reprovar("c", t0); err != ErrTransicaoInvalida {
		t.Fatalf("reprovar sem orçamento enviado: esperava ErrTransicaoInvalida, veio %v", err)
	}
}

func TestValorAPagar(t *testing.T) {
	// Sem decisão: erro.
	o := osEmDiagnostico(t)
	if _, err := o.ValorAPagar(); err != ErrDecisaoNaoRegistrada {
		t.Fatalf("esperava ErrDecisaoNaoRegistrada, veio %v", err)
	}
	// Reprovado paga só o valor do orçamento (R$ 80,00).
	_ = o.EnviarOrcamento(t0)
	_ = o.Reprovar("cli-1", t0)
	if v, err := o.ValorAPagar(); err != nil || v.Centavos() != 8000 {
		t.Fatalf("reprovado: esperava 8000, veio %d (erro: %v)", v.Centavos(), err)
	}
	// Aprovado paga serviços + peças (R$ 500,00), sem o valor do orçamento.
	a := osAprovada(t)
	if v, err := a.ValorAPagar(); err != nil || v.Centavos() != 50000 {
		t.Fatalf("aprovado: esperava 50000, veio %d (erro: %v)", v.Centavos(), err)
	}
}

func TestDesconto(t *testing.T) {
	casos := []struct {
		nome      string
		pontos    int64
		base      BaseDesconto
		gerente   string
		erro      error
		valorDesc int64 // centavos de desconto esperado
		aPagar    int64
	}{
		{"10% sobre serviços + peças", 1000, BaseTotal, "ger-1", nil, 5000, 45000},
		{"2,5% sobre serviços + peças", 250, BaseTotal, "ger-1", nil, 1250, 48750},
		{"10% só sobre a mão de obra", 1000, BaseMaoDeObra, "ger-1", nil, 2000, 48000},
		{"10,5% é recusado", 1050, BaseTotal, "ger-1", ErrDescontoAcimaDoTeto, 0, 0},
		{"15% é recusado", 1500, BaseTotal, "ger-1", ErrDescontoAcimaDoTeto, 0, 0},
		{"sem gerente não aplica", 500, BaseTotal, "  ", ErrDescontoSemAutorizacao, 0, 0},
		{"base inválida", 500, "outra", "ger-1", ErrBaseDeDescontoInvalida, 0, 0},
		{"percentual negativo", -1, BaseTotal, "ger-1", valueobject.ErrPercentualInvalido, 0, 0},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			o := osAprovada(t)
			err := o.AplicarDesconto(c.pontos, c.base, c.gerente, t0.Add(5*time.Minute))
			if err != c.erro {
				t.Fatalf("esperava erro %v, veio %v", c.erro, err)
			}
			if c.erro != nil {
				if o.Desconto() != nil {
					t.Fatal("desconto não deveria ter sido registrado")
				}
				return
			}
			d := o.Desconto()
			if d == nil || d.Valor.Centavos() != c.valorDesc || d.AutorizadoPor != c.gerente || d.PontosBase != c.pontos || d.Base != c.base || d.Em.IsZero() {
				t.Fatalf("desconto registrado incorretamente: %+v", d)
			}
			if v, _ := o.ValorAPagar(); v.Centavos() != c.aPagar {
				t.Fatalf("valor a pagar esperado %d, veio %d", c.aPagar, v.Centavos())
			}
		})
	}
}

func TestDescontoSoComOrcamentoFechado(t *testing.T) {
	semDecisao := osEmDiagnostico(t)
	if err := semDecisao.AplicarDesconto(500, BaseTotal, "ger-1", t0); err != ErrDescontoSemOrcamentoFechado {
		t.Fatalf("sem decisão: esperava ErrDescontoSemOrcamentoFechado, veio %v", err)
	}
	reprovada := osEmDiagnostico(t)
	_ = reprovada.EnviarOrcamento(t0)
	_ = reprovada.Reprovar("cli-1", t0)
	if err := reprovada.AplicarDesconto(500, BaseTotal, "ger-1", t0); err != ErrDescontoSemOrcamentoFechado {
		t.Fatalf("reprovada: esperava ErrDescontoSemOrcamentoFechado, veio %v", err)
	}
}

func TestDescontoSoUmaVez(t *testing.T) {
	o := osAprovada(t)
	if err := o.AplicarDesconto(500, BaseTotal, "ger-1", t0); err != nil {
		t.Fatal(err)
	}
	if err := o.AplicarDesconto(500, BaseTotal, "ger-1", t0); err != ErrDescontoJaAplicado {
		t.Fatalf("esperava ErrDescontoJaAplicado, veio %v", err)
	}
}

func TestStatusPodeIrPara(t *testing.T) {
	if !Recebida.podeIrPara(EmDiagnostico) || Recebida.podeIrPara(Entregue) || Entregue.podeIrPara(Recebida) {
		t.Fatal("tabela de transições inconsistente")
	}
}
