package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros dos casos de uso.
var (
	ErrOrdemNaoEncontrada     = errors.New("ordem de serviço não encontrada")
	ErrPecaNaoEncontrada      = errors.New("peça não encontrada no estoque")
	ErrAcessoNegado           = errors.New("acesso negado: a OS pertence a outro cliente")
	ErrEstoqueInsuficiente    = errors.New("estoque insuficiente para aprovar o orçamento")
	ErrSenhaDoGerenteInvalida = errors.New("senha do gerente inválida")
)

// OrdemDeServicoService reúne os casos de uso do fluxo da OS. Orquestra o domínio,
// os repositórios e a transação, sem conter regra de negócio própria.
type OrdemDeServicoService struct {
	ordens  OrdemRepositorio
	itens   EstoqueRepositorio
	tx      Transacao
	auth    AutorizadorGerente
	relogio Relogio
}

// NovoOrdemDeServicoService monta o serviço com suas dependências.
func NovoOrdemDeServicoService(ordens OrdemRepositorio, itens EstoqueRepositorio, tx Transacao, auth AutorizadorGerente, relogio Relogio) *OrdemDeServicoService {
	return &OrdemDeServicoService{ordens: ordens, itens: itens, tx: tx, auth: auth, relogio: relogio}
}

func (s *OrdemDeServicoService) carregar(ctx context.Context, id string) (*ordemservico.OrdemDeServico, error) {
	o, err := s.ordens.BuscarPorID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return nil, ErrOrdemNaoEncontrada
		}
		return nil, err
	}
	return o, nil
}

// executar carrega a OS, aplica a mudança e salva, tudo numa transação.
func (s *OrdemDeServicoService) executar(ctx context.Context, id string, mudar func(o *ordemservico.OrdemDeServico) error) error {
	return s.tx.Executar(ctx, func(ctx context.Context) error {
		o, err := s.carregar(ctx, id)
		if err != nil {
			return err
		}
		if err := mudar(o); err != nil {
			return err
		}
		return s.ordens.Salvar(ctx, o)
	})
}

// Abrir cria uma OS no status Recebida. O valor do orçamento é o que o cliente paga se reprovar.
func (s *OrdemDeServicoService) Abrir(ctx context.Context, id, clienteID, veiculoID string, valorDoOrcamentoCentavos int64) error {
	valor, err := valueobject.NovoDinheiro(valorDoOrcamentoCentavos)
	if err != nil {
		return err
	}
	o, err := ordemservico.Nova(id, clienteID, veiculoID, valor, s.relogio.Agora())
	if err != nil {
		return err
	}
	return s.ordens.Salvar(ctx, o)
}

// IniciarDiagnostico leva a OS para Em diagnóstico.
func (s *OrdemDeServicoService) IniciarDiagnostico(ctx context.Context, id string) error {
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.IniciarDiagnostico(s.relogio.Agora())
	})
}

// AdicionarServico inclui um serviço de mão de obra na OS.
func (s *OrdemDeServicoService) AdicionarServico(ctx context.Context, id, descricao string, precoCentavos int64) error {
	preco, err := valueobject.NovoDinheiro(precoCentavos)
	if err != nil {
		return err
	}
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.AdicionarServico(descricao, preco)
	})
}

// AdicionarPeca inclui uma peça da lista do estoque na OS.
func (s *OrdemDeServicoService) AdicionarPeca(ctx context.Context, id, pecaID, descricao string, precoUnitarioCentavos int64, quantidade int) error {
	preco, err := valueobject.NovoDinheiro(precoUnitarioCentavos)
	if err != nil {
		return err
	}
	q, err := valueobject.NovaQuantidade(quantidade)
	if err != nil {
		return err
	}
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		if _, err := s.itens.BuscarPorID(ctx, pecaID); err != nil {
			if errors.Is(err, ErrNaoEncontrado) {
				return ErrPecaNaoEncontrada
			}
			return err
		}
		return o.AdicionarPeca(pecaID, descricao, preco, q)
	})
}

// EnviarOrcamento manda o orçamento ao cliente (Aguardando aprovação).
func (s *OrdemDeServicoService) EnviarOrcamento(ctx context.Context, id string) error {
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.EnviarOrcamento(s.relogio.Agora())
	})
}

// AprovarOrcamento registra a aprovação do cliente dono e baixa as peças do estoque,
// numa única transação (RN09). Sem saldo suficiente a aprovação é recusada e nada muda.
func (s *OrdemDeServicoService) AprovarOrcamento(ctx context.Context, id, clienteID string) error {
	return s.tx.Executar(ctx, func(ctx context.Context) error {
		o, err := s.carregar(ctx, id)
		if err != nil {
			return err
		}
		itens, err := s.itensParaBaixa(ctx, o)
		if err != nil {
			return err
		}
		// Confere o saldo de tudo antes de mudar qualquer coisa.
		for _, b := range itens {
			if b.item.Saldo().Valor() < b.quantidade.Valor() {
				return fmt.Errorf("%w: %s", ErrEstoqueInsuficiente, b.item.Nome())
			}
		}
		if err := o.Aprovar(clienteID, s.relogio.Agora()); err != nil {
			return err
		}
		for _, b := range itens {
			if err := b.item.Baixar(b.quantidade); err != nil {
				return err
			}
			if err := s.itens.Salvar(ctx, b.item); err != nil {
				return err
			}
		}
		return s.ordens.Salvar(ctx, o)
	})
}

type baixa struct {
	item       *estoque.Item
	quantidade valueobject.Quantidade
}

// itensParaBaixa junta as peças repetidas da OS e carrega o item de estoque de cada uma.
func (s *OrdemDeServicoService) itensParaBaixa(ctx context.Context, o *ordemservico.OrdemDeServico) ([]baixa, error) {
	total := map[string]int{}
	var ordem []string
	for _, p := range o.Pecas() {
		if _, ok := total[p.PecaID]; !ok {
			ordem = append(ordem, p.PecaID)
		}
		total[p.PecaID] += p.Quantidade.Valor()
	}
	var baixas []baixa
	for _, pecaID := range ordem {
		item, err := s.itens.BuscarPorID(ctx, pecaID)
		if err != nil {
			if errors.Is(err, ErrNaoEncontrado) {
				return nil, ErrPecaNaoEncontrada
			}
			return nil, err
		}
		q, err := valueobject.NovaQuantidade(total[pecaID])
		if err != nil {
			return nil, err
		}
		baixas = append(baixas, baixa{item: item, quantidade: q})
	}
	return baixas, nil
}

// ReprovarOrcamento registra a reprovação do cliente dono. Nenhuma peça é baixada (RN05).
func (s *OrdemDeServicoService) ReprovarOrcamento(ctx context.Context, id, clienteID string) error {
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.Reprovar(clienteID, s.relogio.Agora())
	})
}

// AplicarDesconto concede o desconto depois de conferir a senha de login do gerente (RN07).
func (s *OrdemDeServicoService) AplicarDesconto(ctx context.Context, id, gerenteID, senha string, pontosBase int64, base ordemservico.BaseDesconto) error {
	ok, err := s.auth.SenhaConfere(ctx, gerenteID, senha)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSenhaDoGerenteInvalida
	}
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.AplicarDesconto(pontosBase, base, gerenteID, s.relogio.Agora())
	})
}

// Finalizar encerra a execução da OS.
func (s *OrdemDeServicoService) Finalizar(ctx context.Context, id string) error {
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.Finalizar(s.relogio.Agora())
	})
}

// Entregar registra a devolução do veículo.
func (s *OrdemDeServicoService) Entregar(ctx context.Context, id string) error {
	return s.executar(ctx, id, func(o *ordemservico.OrdemDeServico) error {
		return o.Entregar(s.relogio.Agora())
	})
}

// ConsultarDoCliente devolve a OS só se ela pertence ao cliente autenticado (RN10).
func (s *OrdemDeServicoService) ConsultarDoCliente(ctx context.Context, id, clienteID string) (*ordemservico.OrdemDeServico, error) {
	o, err := s.carregar(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.ClienteID() != clienteID {
		return nil, ErrAcessoNegado
	}
	return o, nil
}

// TempoMedioDeExecucao devolve a média de tempo em execução das OS que já terminaram a
// execução, e quantas OS entraram na conta.
func (s *OrdemDeServicoService) TempoMedioDeExecucao(ctx context.Context) (time.Duration, int, error) {
	todas, err := s.ordens.ListarTodas(ctx)
	if err != nil {
		return 0, 0, err
	}
	var soma time.Duration
	var n int
	for _, o := range todas {
		if d, ok := o.TempoDeExecucao(); ok {
			soma += d
			n++
		}
	}
	if n == 0 {
		return 0, 0, nil
	}
	return soma / time.Duration(n), n, nil
}

// Consultar devolve qualquer OS; é para funcionários. O cliente usa ConsultarDoCliente.
func (s *OrdemDeServicoService) Consultar(ctx context.Context, id string) (*ordemservico.OrdemDeServico, error) {
	return s.carregar(ctx, id)
}

// Listar devolve todas as OS, para funcionários.
func (s *OrdemDeServicoService) Listar(ctx context.Context) ([]*ordemservico.OrdemDeServico, error) {
	return s.ordens.ListarTodas(ctx)
}

// ListarDoCliente devolve só as OS do cliente (RN10).
func (s *OrdemDeServicoService) ListarDoCliente(ctx context.Context, clienteID string) ([]*ordemservico.OrdemDeServico, error) {
	todas, err := s.ordens.ListarTodas(ctx)
	if err != nil {
		return nil, err
	}
	var minhas []*ordemservico.OrdemDeServico
	for _, o := range todas {
		if o.ClienteID() == clienteID {
			minhas = append(minhas, o)
		}
	}
	return minhas, nil
}
