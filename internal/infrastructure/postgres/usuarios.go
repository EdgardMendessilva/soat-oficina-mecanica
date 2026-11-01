package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

// Usuarios implementa application.UsuarioRepositorio e application.AutorizadorGerente.
type Usuarios struct{ db *sql.DB }

// NovosUsuarios cria o repositório de usuários.
func NovosUsuarios(db *sql.DB) *Usuarios { return &Usuarios{db: db} }

// Adicionar cadastra o usuário; se o ID ou o documento já existir, não altera nada (idempotente).
func (r *Usuarios) Adicionar(ctx context.Context, u identidade.Usuario) error {
	_, err := executorDe(ctx, r.db).ExecContext(ctx,
		`INSERT INTO usuarios (id, nome, documento, perfil, cliente_id, senha_hash)
		 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
		u.ID, u.Nome, u.Documento, string(u.Perfil), u.ClienteID, u.SenhaHash)
	return err
}

// BuscarPorDocumento devolve o usuário do CPF informado.
func (r *Usuarios) BuscarPorDocumento(ctx context.Context, documento string) (identidade.Usuario, error) {
	return r.buscar(ctx, `WHERE documento = $1`, documento)
}

func (r *Usuarios) buscar(ctx context.Context, filtro, arg string) (identidade.Usuario, error) {
	var u identidade.Usuario
	var perfil string
	err := executorDe(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, nome, documento, perfil, cliente_id, senha_hash FROM usuarios `+filtro, arg).
		Scan(&u.ID, &u.Nome, &u.Documento, &perfil, &u.ClienteID, &u.SenhaHash)
	if errors.Is(err, sql.ErrNoRows) {
		return identidade.Usuario{}, application.ErrNaoEncontrado
	}
	u.Perfil = identidade.Perfil(perfil)
	return u, err
}

// SenhaConfere implementa application.AutorizadorGerente: só vale para perfil Gerente (RN07).
func (r *Usuarios) SenhaConfere(ctx context.Context, gerenteID, senha string) (bool, error) {
	u, err := r.buscar(ctx, `WHERE id = $1`, gerenteID)
	if errors.Is(err, application.ErrNaoEncontrado) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if u.Perfil != identidade.Gerente {
		return false, nil
	}
	return seguranca.ConferirSenha(senha, u.SenhaHash), nil
}

// Semear grava o Administrador (RN14) e, se demo for true, um usuário de cada perfil.
// É idempotente: reiniciar a API não duplica nem sobrescreve usuários.
func Semear(ctx context.Context, u *Usuarios, adminDocumento, adminSenha string, demo bool) error {
	iniciais, err := memoria.UsuariosIniciais(adminDocumento, adminSenha, demo)
	if err != nil {
		return err
	}
	for _, x := range iniciais {
		if err := u.Adicionar(ctx, x); err != nil {
			return err
		}
	}
	return nil
}
