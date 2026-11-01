package valueobject

import "testing"

func TestNovaPlaca(t *testing.T) {
	casos := []struct {
		nome, entrada, esperado string
		valida                  bool
	}{
		{"padrão antigo", "ABC1234", "ABC1234", true},
		{"padrão antigo com hífen e minúsculas", "abc-1234", "ABC1234", true},
		{"mercosul", "ABC1D23", "ABC1D23", true},
		{"mercosul minúscula", "abc1d23", "ABC1D23", true},
		{"poucos caracteres", "ABC123", "", false},
		{"muitos caracteres", "ABC12345", "", false},
		{"letras no lugar de números", "ABCDEFG", "", false},
		{"formato misto inválido", "AB12C34", "", false},
		{"vazia", "", "", false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			p, err := NovaPlaca(c.entrada)
			if c.valida {
				if err != nil || p.String() != c.esperado {
					t.Fatalf("esperava %q, veio %q (erro: %v)", c.esperado, p.String(), err)
				}
				return
			}
			if err != ErrPlacaInvalida {
				t.Fatalf("esperava ErrPlacaInvalida, veio %v", err)
			}
		})
	}
}
