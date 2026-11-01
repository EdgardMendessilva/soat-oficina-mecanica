package ordemservico

import (
	"errors"
	"strings"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros da OS.
var (
	ErrIDObrigatorio        = errors.New("OS, cliente e veículo são obrigatórios")
	ErrOrcamentoVazio       = errors.New("o orçamento precisa de ao menos um serviço ou uma peça")
	ErrSomenteDonoDecide    = errors.New("somente o cliente dono do veículo pode aprovar ou reprovar o orçamento")
	ErrDecisaoNaoRegistrada = errors.New("a decisão do cliente ainda não foi registrada")
)

// Decisao é a resposta do cliente ao orçamento.
type Decisao string

// Decisões possíveis.
const (
	Pendente  Decisao = "Pendente"
	Aprovado  Decisao = "Aprovado"
	Reprovado Decisao = "Reprovado"
)

// MudancaDeStatus registra uma passagem de status (histórico, base do tempo médio de execução).
type MudancaDeStatus struct {
	De   Status
	Para Status
	Em   time.Time
}

// OrdemDeServico é o agregado raiz do contexto. Toda mudança passa pelos métodos
// abaixo, que garantem as regras RN03 a RN07.
type OrdemDeServico struct {
	id               string
	clienteID        string
	veiculoID        string
	valorDoOrcamento valueobject.Dinheiro
	status           Status
	servicos         []Servico
	pecas            []Peca
	decisao          Decisao
	decididoPor      string
	decididoEm       time.Time
	desconto         *Desconto
	historico        []MudancaDeStatus
}

// Nova abre uma OS no status Recebida. O valor do orçamento é o que o cliente paga
// quando reprova o orçamento (RN05).
func Nova(id, clienteID, veiculoID string, valorDoOrcamento valueobject.Dinheiro, agora time.Time) (*OrdemDeServico, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(clienteID) == "" || strings.TrimSpace(veiculoID) == "" {
		return nil, ErrIDObrigatorio
	}
	return &OrdemDeServico{
		id: id, clienteID: clienteID, veiculoID: veiculoID,
		valorDoOrcamento: valorDoOrcamento,
		status:           Recebida,
		decisao:          Pendente,
		historico:        []MudancaDeStatus{{Para: Recebida, Em: agora}},
	}, nil
}

// Getters.
func (o *OrdemDeServico) ID() string            { return o.id }
func (o *OrdemDeServico) ClienteID() string     { return o.clienteID }
func (o *OrdemDeServico) VeiculoID() string     { return o.veiculoID }
func (o *OrdemDeServico) Status() Status        { return o.status }
func (o *OrdemDeServico) Decisao() Decisao      { return o.decisao }
func (o *OrdemDeServico) DecididoPor() string   { return o.decididoPor }
func (o *OrdemDeServico) DecididoEm() time.Time { return o.decididoEm }
func (o *OrdemDeServico) Desconto() *Desconto   { return o.desconto }
func (o *OrdemDeServico) Servicos() []Servico   { return append([]Servico(nil), o.servicos...) }
func (o *OrdemDeServico) Pecas() []Peca         { return append([]Peca(nil), o.pecas...) }
func (o *OrdemDeServico) Historico() []MudancaDeStatus {
	return append([]MudancaDeStatus(nil), o.historico...)
}

func (o *OrdemDeServico) mover(para Status, agora time.Time) error {
	if !o.status.podeIrPara(para) {
		return ErrTransicaoInvalida
	}
	o.historico = append(o.historico, MudancaDeStatus{De: o.status, Para: para, Em: agora})
	o.status = para
	return nil
}

// IniciarDiagnostico leva a OS de Recebida para Em diagnóstico.
func (o *OrdemDeServico) IniciarDiagnostico(agora time.Time) error {
	return o.mover(EmDiagnostico, agora)
}

// AdicionarServico inclui um serviço; só é permitido durante o diagnóstico (RN04).
func (o *OrdemDeServico) AdicionarServico(descricao string, preco valueobject.Dinheiro) error {
	if o.status != EmDiagnostico {
		return ErrItensSoNoDiagnostico
	}
	if strings.TrimSpace(descricao) == "" {
		return ErrDescricaoVazia
	}
	o.servicos = append(o.servicos, Servico{Descricao: descricao, Preco: preco})
	return nil
}

// AdicionarPeca inclui uma peça ou insumo; só é permitido durante o diagnóstico (RN04).
func (o *OrdemDeServico) AdicionarPeca(pecaID, descricao string, precoUnitario valueobject.Dinheiro, quantidade valueobject.Quantidade) error {
	if o.status != EmDiagnostico {
		return ErrItensSoNoDiagnostico
	}
	if strings.TrimSpace(pecaID) == "" || strings.TrimSpace(descricao) == "" {
		return ErrDescricaoVazia
	}
	if quantidade.Valor() == 0 {
		return ErrQuantidadeInvalida
	}
	o.pecas = append(o.pecas, Peca{PecaID: pecaID, Descricao: descricao, PrecoUnitario: precoUnitario, Quantidade: quantidade})
	return nil
}

// MaoDeObra devolve a soma dos serviços.
func (o *OrdemDeServico) MaoDeObra() valueobject.Dinheiro {
	total, _ := valueobject.NovoDinheiro(0)
	for _, s := range o.servicos {
		total = total.Somar(s.Preco)
	}
	return total
}

// TotalDoOrcamento devolve serviços + peças × quantidade, recalculado a cada chamada (RN04).
func (o *OrdemDeServico) TotalDoOrcamento() valueobject.Dinheiro {
	total := o.MaoDeObra()
	for _, p := range o.pecas {
		total = total.Somar(p.Subtotal())
	}
	return total
}

// EnviarOrcamento leva a OS para Aguardando aprovação. Daqui em diante os itens ficam travados.
func (o *OrdemDeServico) EnviarOrcamento(agora time.Time) error {
	if len(o.servicos) == 0 && len(o.pecas) == 0 {
		return ErrOrcamentoVazio
	}
	return o.mover(AguardandoAprovacao, agora)
}

func (o *OrdemDeServico) validarDecisor(quemDecide string) error {
	if o.status != AguardandoAprovacao {
		return ErrTransicaoInvalida
	}
	if quemDecide != o.clienteID {
		return ErrSomenteDonoDecide
	}
	return nil
}

// Aprovar registra a aprovação do orçamento pelo cliente dono e inicia a execução (RN06).
// O sistema registra quem decidiu e quando.
func (o *OrdemDeServico) Aprovar(quemDecide string, agora time.Time) error {
	if err := o.validarDecisor(quemDecide); err != nil {
		return err
	}
	if err := o.mover(EmExecucao, agora); err != nil {
		return err
	}
	o.decisao, o.decididoPor, o.decididoEm = Aprovado, quemDecide, agora
	return nil
}

// Reprovar registra a reprovação do orçamento: a OS vai para Finalizada sem execução (RN05, RN06).
func (o *OrdemDeServico) Reprovar(quemDecide string, agora time.Time) error {
	if err := o.validarDecisor(quemDecide); err != nil {
		return err
	}
	if err := o.mover(Finalizada, agora); err != nil {
		return err
	}
	o.decisao, o.decididoPor, o.decididoEm = Reprovado, quemDecide, agora
	return nil
}

// Finalizar encerra a execução (Em execução para Finalizada).
func (o *OrdemDeServico) Finalizar(agora time.Time) error { return o.mover(Finalizada, agora) }

// Entregar registra a devolução do veículo ao cliente (Finalizada para Entregue).
func (o *OrdemDeServico) Entregar(agora time.Time) error { return o.mover(Entregue, agora) }

// ValorAPagar aplica a RN05: reprovado paga apenas o valor do orçamento; aprovado paga
// serviços + peças menos o desconto, e não paga o valor do orçamento.
func (o *OrdemDeServico) ValorAPagar() (valueobject.Dinheiro, error) {
	switch o.decisao {
	case Reprovado:
		return o.valorDoOrcamento, nil
	case Aprovado:
		total := o.TotalDoOrcamento()
		if o.desconto == nil {
			return total, nil
		}
		return total.Subtrair(o.desconto.Valor)
	default:
		return valueobject.Dinheiro{}, ErrDecisaoNaoRegistrada
	}
}

// TempoDeExecucao devolve quanto tempo a OS ficou Em execução. O segundo retorno é
// falso se a execução ainda não começou ou não terminou. Base do tempo médio (TT-41).
func (o *OrdemDeServico) TempoDeExecucao() (time.Duration, bool) {
	var inicio, fim time.Time
	for _, m := range o.historico {
		if m.Para == EmExecucao {
			inicio = m.Em
		}
		if m.De == EmExecucao && m.Para == Finalizada {
			fim = m.Em
		}
	}
	if inicio.IsZero() || fim.IsZero() {
		return 0, false
	}
	return fim.Sub(inicio), true
}
