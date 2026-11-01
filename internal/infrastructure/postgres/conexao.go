// Package postgres implementa as portas da aplicação sobre o PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" para database/sql
)

//go:embed migrations/*.sql
var migrations embed.FS

// Abrir conecta ao banco, confere a conexão e aplica as migrations pendentes.
func Abrir(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectar ao banco: %w", err)
	}
	if err := Migrar(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// idTravaMigracao é a chave do advisory lock que impede duas instâncias de migrarem juntas.
const idTravaMigracao = 726130001

// Migrar aplica, em ordem de nome, as migrations ainda não registradas em schema_migrations.
func Migrar(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, idTravaMigracao); err != nil {
		return err
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, idTravaMigracao) //nolint:errcheck

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		versao TEXT PRIMARY KEY,
		aplicada_em TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}

	arquivos, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	nomes := make([]string, 0, len(arquivos))
	for _, a := range arquivos {
		nomes = append(nomes, a.Name())
	}
	sort.Strings(nomes)

	for _, nome := range nomes {
		var existe bool
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE versao = $1)`, nome).Scan(&existe); err != nil {
			return err
		}
		if existe {
			continue
		}
		script, err := migrations.ReadFile("migrations/" + nome)
		if err != nil {
			return err
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("migration %s: %w", nome, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (versao) VALUES ($1)`, nome); err != nil {
			tx.Rollback() //nolint:errcheck
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
