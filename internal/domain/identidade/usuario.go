package identidade

import (
	"errors"
	"strings"
	"unicode"
)

// Perfil é o papel do usuário no sistema (seção 3 do escopo).
type Perfil string

// Perfis existentes.
const (
	Cliente       Perfil = "Cliente"
	Atendente     Perfil = "Atendente"
	Mecanico      Perfil = "Mecânico"
	Gerente       Perfil = "Gerente"
	Administrador Perfil = "Administrador"
)

// EhFuncionario diz se o perfil é de funcionário da oficina.
func (p Perfil) EhFuncionario() bool {
	return p == Atendente || p == Mecanico || p == Gerente || p == Administrador
}

// EstaEntre diz se o perfil está na lista.
func (p Perfil) EstaEntre(perfis ...Perfil) bool {
	for _, x := range perfis {
		if p == x {
			return true
		}
	}
	return false
}

// Usuario é quem entra no sistema com documento (CPF) e senha. Guarda só o hash da senha (RN12).
type Usuario struct {
	ID        string
	Nome      string
	Documento string
	Perfil    Perfil
	ClienteID string // preenchido quando o perfil é Cliente (RN10)
	SenhaHash string
}

// ErrSenhaFraca indica senha fora da política da RN12.
var ErrSenhaFraca = errors.New("senha fora do padrão")

// ValidarSenha aplica a RN12 e diz o que falta na mensagem de erro.
func ValidarSenha(s string) error {
	var faltas []string
	if len([]rune(s)) < 8 {
		faltas = append(faltas, "mínimo de 8 caracteres")
	}
	var mai, min, num, esp bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			mai = true
		case unicode.IsLower(r):
			min = true
		case unicode.IsDigit(r):
			num = true
		default:
			esp = true
		}
	}
	if !mai {
		faltas = append(faltas, "uma letra maiúscula")
	}
	if !min {
		faltas = append(faltas, "uma letra minúscula")
	}
	if !num {
		faltas = append(faltas, "um número")
	}
	if !esp {
		faltas = append(faltas, "um caractere especial")
	}
	if len(faltas) > 0 {
		return errors.Join(ErrSenhaFraca, errors.New("falta: "+strings.Join(faltas, ", ")))
	}
	return nil
}
