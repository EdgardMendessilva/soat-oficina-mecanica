package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Estoque implementa application.EstoqueRepositorio.
type Estoque struct{ db *sql.DB }

// NovoEstoque cria o repositório de itens de estoque.
func NovoEstoque(db *sql.DB) *Estoque { return &Estoque{db: db} }

// BuscarPorID devolve o item ou ErrNaoEncontrado. Dentro de uma transação trava a linha
// com FOR UPDATE, para que a baixa do saldo não corra junto com outra (RN09).
func (r *Estoque) BuscarPorID(ctx context.Context, id string) (*estoque.Item, error) {
	consulta := `SELECT id, nome, tipo, saldo FROM itens_estoque WHERE id = $1`
	if _, ok := emTransacao(ctx); ok {
		consulta += ` FOR UPDATE`
	}
	var (
		iid, nome, tipo string
		saldo           int
	)
	err := executorDe(ctx, r.db).QueryRowContext(ctx, consulta, id).Scan(&iid, &nome, &tipo, &saldo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	q, err := valueobject.NovaQuantidade(saldo)
	if err != nil {
		return nil, err
	}
	return estoque.NovoItem(iid, nome, estoque.Tipo(tipo), q)
}

// Salvar grava ou atualiza o item.
func (r *Estoque) Salvar(ctx context.Context, i *estoque.Item) error {
	_, err := executorDe(ctx, r.db).ExecContext(ctx,
		`INSERT INTO itens_estoque (id, nome, tipo, saldo) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE SET nome = EXCLUDED.nome, tipo = EXCLUDED.tipo, saldo = EXCLUDED.saldo`,
		i.ID(), i.Nome(), string(i.Tipo()), i.Saldo().Valor())
	return err
}
