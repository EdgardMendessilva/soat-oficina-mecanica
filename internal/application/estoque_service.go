package application

import (
	"context"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// ErrItemJaExiste evita cadastrar duas vezes o mesmo identificador de estoque.
var ErrItemJaExiste = errors.New("já existe um item de estoque com esse identificador")

// ErrItemNaoEncontrado é devolvido quando o item de estoque não existe.
var ErrItemNaoEncontrado = errors.New("item de estoque não encontrado")

// EstoqueService cadastra e repõe peças e insumos.
type EstoqueService struct {
	itens EstoqueRepositorio
	tx    Transacao
}

// NovoEstoqueService monta o serviço.
func NovoEstoqueService(itens EstoqueRepositorio, tx Transacao) *EstoqueService {
	return &EstoqueService{itens: itens, tx: tx}
}

// Cadastrar cria o item com o saldo inicial.
func (s *EstoqueService) Cadastrar(ctx context.Context, id, nome string, tipo estoque.Tipo, saldo int) (*estoque.Item, error) {
	q, err := valueobject.NovaQuantidade(saldo)
	if err != nil {
		return nil, err
	}
	item, err := estoque.NovoItem(id, nome, tipo, q)
	if err != nil {
		return nil, err
	}
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		if _, err := s.itens.BuscarPorID(ctx, id); err == nil {
			return ErrItemJaExiste
		} else if !errors.Is(err, ErrNaoEncontrado) {
			return err
		}
		return s.itens.Salvar(ctx, item)
	})
	return item, err
}

// Repor soma a quantidade ao saldo.
func (s *EstoqueService) Repor(ctx context.Context, id string, quantidade int) (*estoque.Item, error) {
	q, err := valueobject.NovaQuantidade(quantidade)
	if err != nil {
		return nil, err
	}
	var item *estoque.Item
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		var err error
		if item, err = s.Consultar(ctx, id); err != nil {
			return err
		}
		if err := item.Repor(q); err != nil {
			return err
		}
		return s.itens.Salvar(ctx, item)
	})
	return item, err
}

// Consultar devolve o item.
func (s *EstoqueService) Consultar(ctx context.Context, id string) (*estoque.Item, error) {
	item, err := s.itens.BuscarPorID(ctx, id)
	if errors.Is(err, ErrNaoEncontrado) {
		return nil, ErrItemNaoEncontrado
	}
	return item, err
}
