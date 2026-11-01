package application

import (
	"context"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/catalogo"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros do catálogo.
var (
	ErrServicoJaExiste      = errors.New("já existe um serviço com esse identificador")
	ErrServicoNaoEncontrado = errors.New("serviço não encontrado no catálogo")
)

// CatalogoService mantém o catálogo de serviços e preços (CRUD).
type CatalogoService struct {
	servicos ServicoRepositorio
	tx       Transacao
}

// NovoCatalogoService monta o serviço.
func NovoCatalogoService(s ServicoRepositorio, tx Transacao) *CatalogoService {
	return &CatalogoService{servicos: s, tx: tx}
}

// Criar cadastra um serviço no catálogo.
func (s *CatalogoService) Criar(ctx context.Context, id, nome string, precoCentavos int64) (*catalogo.Servico, error) {
	preco, err := valueobject.NovoDinheiro(precoCentavos)
	if err != nil {
		return nil, err
	}
	sv, err := catalogo.NovoServico(id, nome, preco)
	if err != nil {
		return nil, err
	}
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		if _, err := s.servicos.BuscarPorID(ctx, id); err == nil {
			return ErrServicoJaExiste
		} else if !errors.Is(err, ErrNaoEncontrado) {
			return err
		}
		return s.servicos.Salvar(ctx, sv)
	})
	return sv, err
}

// Consultar devolve o serviço do catálogo.
func (s *CatalogoService) Consultar(ctx context.Context, id string) (*catalogo.Servico, error) {
	sv, err := s.servicos.BuscarPorID(ctx, id)
	if errors.Is(err, ErrNaoEncontrado) {
		return nil, ErrServicoNaoEncontrado
	}
	return sv, err
}

// Listar devolve todos os serviços.
func (s *CatalogoService) Listar(ctx context.Context) ([]*catalogo.Servico, error) {
	return s.servicos.Listar(ctx)
}

// Alterar troca nome e preço.
func (s *CatalogoService) Alterar(ctx context.Context, id, nome string, precoCentavos int64) (*catalogo.Servico, error) {
	preco, err := valueobject.NovoDinheiro(precoCentavos)
	if err != nil {
		return nil, err
	}
	var sv *catalogo.Servico
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		var err error
		if sv, err = s.Consultar(ctx, id); err != nil {
			return err
		}
		if err := sv.Alterar(nome, preco); err != nil {
			return err
		}
		return s.servicos.Salvar(ctx, sv)
	})
	return sv, err
}

// Remover tira o serviço do catálogo. As OS já abertas guardam o próprio texto e preço e não mudam.
func (s *CatalogoService) Remover(ctx context.Context, id string) error {
	if _, err := s.Consultar(ctx, id); err != nil {
		return err
	}
	return s.servicos.Remover(ctx, id)
}
