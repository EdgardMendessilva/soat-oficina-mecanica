package application

import (
	"context"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// ErrCredenciaisInvalidas não diz se o erro foi no CPF ou na senha, de propósito.
var ErrCredenciaisInvalidas = errors.New("CPF ou senha inválidos")

// AutenticacaoService faz o login por CPF + senha.
type AutenticacaoService struct {
	usuarios    UsuarioRepositorio
	verificador VerificadorSenha
}

// NovoAutenticacaoService monta o serviço.
func NovoAutenticacaoService(u UsuarioRepositorio, v VerificadorSenha) *AutenticacaoService {
	return &AutenticacaoService{usuarios: u, verificador: v}
}

// Login confere CPF e senha e devolve o usuário.
func (s *AutenticacaoService) Login(ctx context.Context, documento, senha string) (identidade.Usuario, error) {
	doc, err := valueobject.NovoDocumento(documento)
	if err != nil {
		return identidade.Usuario{}, ErrCredenciaisInvalidas
	}
	u, err := s.usuarios.BuscarPorDocumento(ctx, doc.Numero())
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return identidade.Usuario{}, ErrCredenciaisInvalidas
		}
		return identidade.Usuario{}, err
	}
	if !s.verificador.Conferir(senha, u.SenhaHash) {
		return identidade.Usuario{}, ErrCredenciaisInvalidas
	}
	return u, nil
}
