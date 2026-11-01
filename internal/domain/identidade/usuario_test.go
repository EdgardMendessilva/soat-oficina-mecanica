package identidade

import (
	"errors"
	"strings"
	"testing"
)

func TestValidarSenha(t *testing.T) {
	casos := []struct {
		senha string
		falta string
	}{
		{"Teste@1234", ""},
		{"Ab@1", "8 caracteres"},
		{"teste@1234", "maiúscula"},
		{"TESTE@1234", "minúscula"},
		{"Teste@abcd", "número"},
		{"Teste12345", "especial"},
	}
	for _, c := range casos {
		err := ValidarSenha(c.senha)
		if c.falta == "" {
			if err != nil {
				t.Errorf("%q deveria ser aceita: %v", c.senha, err)
			}
			continue
		}
		if !errors.Is(err, ErrSenhaFraca) || !strings.Contains(err.Error(), c.falta) {
			t.Errorf("%q: esperava faltar %q, veio %v", c.senha, c.falta, err)
		}
	}
}

func TestPerfis(t *testing.T) {
	if Cliente.EhFuncionario() || !Mecanico.EhFuncionario() {
		t.Error("EhFuncionario errado")
	}
	if !Gerente.EstaEntre(Gerente, Administrador) || Atendente.EstaEntre(Gerente) {
		t.Error("EstaEntre errado")
	}
}
