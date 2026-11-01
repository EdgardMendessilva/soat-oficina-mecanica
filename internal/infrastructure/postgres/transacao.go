package postgres

import (
	"context"
	"database/sql"
)

type chaveTx struct{}

// executor é o que *sql.DB e *sql.Tx têm em comum.
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Transacao implementa application.Transacao: BEGIN/COMMIT, com o *sql.Tx no contexto.
type Transacao struct{ db *sql.DB }

// NovaTransacao cria o executor de transações sobre o pool.
func NovaTransacao(db *sql.DB) *Transacao { return &Transacao{db: db} }

// Executar roda fn numa transação; erro (ou pânico) em fn desfaz tudo. Chamada aninhada
// reaproveita a transação já aberta.
func (t *Transacao) Executar(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := emTransacao(ctx); ok {
		return fn(ctx)
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback() //nolint:errcheck
			panic(p)
		}
	}()
	if err := fn(context.WithValue(ctx, chaveTx{}, tx)); err != nil {
		tx.Rollback() //nolint:errcheck
		return err
	}
	return tx.Commit()
}

// emTransacao devolve a transação do contexto, se houver.
func emTransacao(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(chaveTx{}).(*sql.Tx)
	return tx, ok
}

// executorDe devolve a transação do contexto ou, fora de uma, o pool.
func executorDe(ctx context.Context, db *sql.DB) executor {
	if tx, ok := emTransacao(ctx); ok {
		return tx
	}
	return db
}
