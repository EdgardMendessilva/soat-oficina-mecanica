package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// --- dublês de teste (repositórios em memória) ---

type ordensFake struct {
	dados  map[string]*ordemservico.OrdemDeServico
	salvou int
}

func (f *ordensFake) BuscarPorID(_ context.Context, id string) (*ordemservico.OrdemDeServico, error) {
	if o, ok := f.dados[id]; ok {
		return o, nil
	}
	return nil, ErrNaoEncontrado
}
func (f *ordensFake) Salvar(_ context.Context, o *ordemservico.OrdemDeServico) error {
	f.dados[o.ID()] = o
	f.salvou++
	return nil
}
func (f *ordensFake) ListarTodas(context.Context) ([]*ordemservico.OrdemDeServico, error) {
	var l []*ordemservico.OrdemDeServico
	for _, o := range f.dados {
		l = append(l, o)
	}
	return l, nil
}

type estoqueFake struct {
	dados  map[string]*estoque.Item
	salvou int
}

func (f *estoqueFake) BuscarPorID(_ context.Context, id string) (*estoque.Item, error) {
	if i, ok := f.dados[id]; ok {
		return i, nil
	}
	return nil, ErrNaoEncontrado
}
func (f *estoqueFake) Salvar(_ context.Context, i *estoque.Item) error {
	f.dados[i.ID()] = i
	f.salvou++
	return nil
}

type txFake struct{ chamadas int }

func (f *txFake) Executar(ctx context.Context, fn func(context.Context) error) error {
	f.chamadas++
	return fn(ctx)
}

type authFake struct {
	senhas map[string]string
	err    error
}

func (f authFake) SenhaConfere(_ context.Context, gerenteID, senha string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.senhas[gerenteID] == senha, nil
}

type relogioFake struct{ t time.Time }

func (r *relogioFake) Agora() time.Time        { return r.t }
func (r *relogioFake) Avancar(d time.Duration) { r.t = r.t.Add(d) }

// --- montagem do cenário ---

type cenario struct {
	svc     *OrdemDeServicoService
	ordens  *ordensFake
	itens   *estoqueFake
	tx      *txFake
	relogio *relogioFake
}

func novoCenario(t *testing.T, saldoFiltro int) *cenario {
	t.Helper()
	q, _ := valueobject.NovaQuantidade(saldoFiltro)
	filtro, err := estoque.NovoItem("filtro", "Filtro de óleo", estoque.Peca, q)
	if err != nil {
		t.Fatal(err)
	}
	c := &cenario{
		ordens:  &ordensFake{dados: map[string]*ordemservico.OrdemDeServico{}},
		itens:   &estoqueFake{dados: map[string]*estoque.Item{"filtro": filtro}},
		tx:      &txFake{},
		relogio: &relogioFake{t: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)},
	}
	auth := authFake{senhas: map[string]string{"ger-1": "Teste@1234"}}
	c.svc = NovoOrdemDeServicoService(c.ordens, c.itens, c.tx, auth, c.relogio)
	return c
}

// osAguardando leva uma OS até Aguardando aprovação: serviço de R$ 200,00 e peças
// pedindo 2 unidades de filtro a R$ 150,00 (total R$ 500,00).
func (c *cenario) osAguardando(t *testing.T, id string, pecas ...int) {
	t.Helper()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(c.svc.Abrir(ctx, id, "cli-1", "vei-1", 8000))
	must(c.svc.IniciarDiagnostico(ctx, id))
	must(c.svc.AdicionarServico(ctx, id, "Troca de óleo", 20000))
	if len(pecas) == 0 {
		pecas = []int{2}
	}
	for _, n := range pecas {
		must(c.svc.AdicionarPeca(ctx, id, "filtro", "Filtro de óleo", 15000, n))
	}
	must(c.svc.EnviarOrcamento(ctx, id))
}

// --- testes ---

func TestFluxoCompletoComBaixaDeEstoque(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")

	if err := c.svc.AprovarOrcamento(ctx, "OS-1", "cli-1"); err != nil {
		t.Fatal(err)
	}
	o := c.ordens.dados["OS-1"]
	if o.Status() != ordemservico.EmExecucao || o.Decisao() != ordemservico.Aprovado {
		t.Fatalf("estado inesperado: %s / %s", o.Status(), o.Decisao())
	}
	if saldo := c.itens.dados["filtro"].Saldo().Valor(); saldo != 8 {
		t.Fatalf("saldo esperado 8 após baixar 2, veio %d", saldo)
	}

	if err := c.svc.AplicarDesconto(ctx, "OS-1", "ger-1", "Teste@1234", 1000, ordemservico.BaseTotal); err != nil {
		t.Fatal(err)
	}
	c.relogio.Avancar(2 * time.Hour)
	if err := c.svc.Finalizar(ctx, "OS-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.svc.Entregar(ctx, "OS-1"); err != nil {
		t.Fatal(err)
	}
	if v, err := o.ValorAPagar(); err != nil || v.Centavos() != 45000 {
		t.Fatalf("valor a pagar esperado 45000 (500,00 menos 10%%), veio %d (erro: %v)", v.Centavos(), err)
	}
	media, n, err := c.svc.TempoMedioDeExecucao(ctx)
	if err != nil || n != 1 || media != 2*time.Hour {
		t.Fatalf("tempo médio esperado 2h com 1 OS, veio %v com %d (erro: %v)", media, n, err)
	}
}

func TestAprovarSemSaldoNaoMudaNada(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 1) // pede 2, só tem 1
	c.osAguardando(t, "OS-1")
	salvouOS, salvouItens := c.ordens.salvou, c.itens.salvou

	err := c.svc.AprovarOrcamento(ctx, "OS-1", "cli-1")
	if !errors.Is(err, ErrEstoqueInsuficiente) {
		t.Fatalf("esperava ErrEstoqueInsuficiente, veio %v", err)
	}
	o := c.ordens.dados["OS-1"]
	if o.Status() != ordemservico.AguardandoAprovacao || o.Decisao() != ordemservico.Pendente {
		t.Fatalf("a OS não deveria ter mudado: %s / %s", o.Status(), o.Decisao())
	}
	if saldo := c.itens.dados["filtro"].Saldo().Valor(); saldo != 1 {
		t.Fatalf("o saldo não deveria mudar, veio %d", saldo)
	}
	if c.ordens.salvou != salvouOS || c.itens.salvou != salvouItens {
		t.Fatal("nada deveria ter sido gravado")
	}
}

func TestPecasRepetidasSomamNaBaixa(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 5)
	c.osAguardando(t, "OS-1", 3, 3) // duas linhas de 3 = 6, mas só há 5
	if err := c.svc.AprovarOrcamento(ctx, "OS-1", "cli-1"); !errors.Is(err, ErrEstoqueInsuficiente) {
		t.Fatalf("esperava ErrEstoqueInsuficiente pela soma das linhas, veio %v", err)
	}
	c2 := novoCenario(t, 6)
	c2.osAguardando(t, "OS-1", 3, 3)
	if err := c2.svc.AprovarOrcamento(ctx, "OS-1", "cli-1"); err != nil {
		t.Fatal(err)
	}
	if saldo := c2.itens.dados["filtro"].Saldo().Valor(); saldo != 0 {
		t.Fatalf("saldo esperado 0, veio %d", saldo)
	}
}

func TestReprovarNaoBaixaEstoque(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")
	if err := c.svc.ReprovarOrcamento(ctx, "OS-1", "cli-1"); err != nil {
		t.Fatal(err)
	}
	o := c.ordens.dados["OS-1"]
	if o.Status() != ordemservico.Finalizada || o.Decisao() != ordemservico.Reprovado {
		t.Fatalf("estado inesperado: %s / %s", o.Status(), o.Decisao())
	}
	if saldo := c.itens.dados["filtro"].Saldo().Valor(); saldo != 10 {
		t.Fatalf("saldo não deveria mudar na reprovação, veio %d", saldo)
	}
	if v, _ := o.ValorAPagar(); v.Centavos() != 8000 {
		t.Fatalf("reprovado paga só o valor do orçamento (8000), veio %d", v.Centavos())
	}
}

func TestSoOClienteDonoDecide(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")
	if err := c.svc.AprovarOrcamento(ctx, "OS-1", "outro"); !errors.Is(err, ordemservico.ErrSomenteDonoDecide) {
		t.Fatalf("esperava ErrSomenteDonoDecide, veio %v", err)
	}
	if err := c.svc.ReprovarOrcamento(ctx, "OS-1", "admin"); !errors.Is(err, ordemservico.ErrSomenteDonoDecide) {
		t.Fatalf("esperava ErrSomenteDonoDecide, veio %v", err)
	}
	if saldo := c.itens.dados["filtro"].Saldo().Valor(); saldo != 10 {
		t.Fatalf("saldo não deveria mudar, veio %d", saldo)
	}
}

func TestDescontoExigeSenhaDoGerente(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")
	_ = c.svc.AprovarOrcamento(ctx, "OS-1", "cli-1")

	if err := c.svc.AplicarDesconto(ctx, "OS-1", "ger-1", "errada", 500, ordemservico.BaseTotal); err != ErrSenhaDoGerenteInvalida {
		t.Fatalf("esperava ErrSenhaDoGerenteInvalida, veio %v", err)
	}
	if c.ordens.dados["OS-1"].Desconto() != nil {
		t.Fatal("sem a senha certa o desconto não pode ser aplicado")
	}
	if err := c.svc.AplicarDesconto(ctx, "OS-1", "ger-1", "Teste@1234", 1050, ordemservico.BaseTotal); !errors.Is(err, ordemservico.ErrDescontoAcimaDoTeto) {
		t.Fatalf("esperava ErrDescontoAcimaDoTeto, veio %v", err)
	}
	// Falha do autorizador também impede o desconto.
	c.svc.auth = authFake{err: errors.New("banco fora do ar")}
	if err := c.svc.AplicarDesconto(ctx, "OS-1", "ger-1", "Teste@1234", 500, ordemservico.BaseTotal); err == nil {
		t.Fatal("erro do autorizador deveria ser propagado")
	}
}

func TestIsolamentoDoCliente(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")
	if o, err := c.svc.ConsultarDoCliente(ctx, "OS-1", "cli-1"); err != nil || o.ID() != "OS-1" {
		t.Fatalf("o dono deveria ver a própria OS (erro: %v)", err)
	}
	if _, err := c.svc.ConsultarDoCliente(ctx, "OS-1", "cli-2"); err != ErrAcessoNegado {
		t.Fatalf("esperava ErrAcessoNegado, veio %v", err)
	}
	if _, err := c.svc.ConsultarDoCliente(ctx, "nao-existe", "cli-1"); err != ErrOrdemNaoEncontrada {
		t.Fatalf("esperava ErrOrdemNaoEncontrada, veio %v", err)
	}
}

func TestErrosDeEntrada(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t, 10)
	if err := c.svc.Abrir(ctx, "OS-1", "cli-1", "vei-1", -1); err != valueobject.ErrValorNegativo {
		t.Fatalf("esperava ErrValorNegativo, veio %v", err)
	}
	if err := c.svc.Abrir(ctx, "", "cli-1", "vei-1", 100); err != ordemservico.ErrIDObrigatorio {
		t.Fatalf("esperava ErrIDObrigatorio, veio %v", err)
	}
	_ = c.svc.Abrir(ctx, "OS-1", "cli-1", "vei-1", 100)
	_ = c.svc.IniciarDiagnostico(ctx, "OS-1")
	if err := c.svc.AdicionarServico(ctx, "OS-1", "x", -5); err != valueobject.ErrValorNegativo {
		t.Fatalf("esperava ErrValorNegativo, veio %v", err)
	}
	if err := c.svc.AdicionarPeca(ctx, "OS-1", "filtro", "x", -5, 1); err != valueobject.ErrValorNegativo {
		t.Fatalf("esperava ErrValorNegativo, veio %v", err)
	}
	if err := c.svc.AdicionarPeca(ctx, "OS-1", "filtro", "x", 5, -1); err != valueobject.ErrQuantidadeNegativa {
		t.Fatalf("esperava ErrQuantidadeNegativa, veio %v", err)
	}
	if err := c.svc.AdicionarPeca(ctx, "OS-1", "inexistente", "x", 5, 1); err != ErrPecaNaoEncontrada {
		t.Fatalf("esperava ErrPecaNaoEncontrada, veio %v", err)
	}
	if err := c.svc.IniciarDiagnostico(ctx, "nao-existe"); err != ErrOrdemNaoEncontrada {
		t.Fatalf("esperava ErrOrdemNaoEncontrada, veio %v", err)
	}
}

func TestTempoMedioSemExecucoes(t *testing.T) {
	c := novoCenario(t, 10)
	media, n, err := c.svc.TempoMedioDeExecucao(context.Background())
	if err != nil || media != 0 || n != 0 {
		t.Fatalf("esperava 0 OS na conta, veio %v / %d (erro: %v)", media, n, err)
	}
}

func TestTodasAsMudancasPassamPorTransacao(t *testing.T) {
	c := novoCenario(t, 10)
	c.osAguardando(t, "OS-1")
	if c.tx.chamadas == 0 {
		t.Fatal("as mudanças da OS devem rodar dentro de uma transação")
	}
}
