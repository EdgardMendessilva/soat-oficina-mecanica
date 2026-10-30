package valueobject

import "testing"

func dinheiro(t *testing.T, centavos int64) Dinheiro {
	t.Helper()
	d, err := NovoDinheiro(centavos)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	return d
}

func TestNovoDinheiro(t *testing.T) {
	if d := dinheiro(t, 12345); d.Centavos() != 12345 {
		t.Fatalf("centavos inesperados: %d", d.Centavos())
	}
	if _, err := NovoDinheiro(-1); err != ErrValorNegativo {
		t.Fatalf("esperava ErrValorNegativo, veio %v", err)
	}
}

func TestSomarEMultiplicar(t *testing.T) {
	soma := dinheiro(t, 10000).Somar(dinheiro(t, 2550))
	if soma.Centavos() != 12550 {
		t.Fatalf("soma errada: %d", soma.Centavos())
	}
	q, _ := NovaQuantidade(3)
	if total := dinheiro(t, 4000).Multiplicar(q); total.Centavos() != 12000 {
		t.Fatalf("multiplicação errada: %d", total.Centavos())
	}
}

func TestSubtrair(t *testing.T) {
	r, err := dinheiro(t, 10000).Subtrair(dinheiro(t, 2500))
	if err != nil || r.Centavos() != 7500 {
		t.Fatalf("subtração errada: %d (erro: %v)", r.Centavos(), err)
	}
	if _, err := dinheiro(t, 1000).Subtrair(dinheiro(t, 2500)); err != ErrValorNegativo {
		t.Fatalf("esperava ErrValorNegativo, veio %v", err)
	}
}

func TestPercentual(t *testing.T) {
	casos := []struct {
		nome     string
		base     int64
		pontos   int64
		esperado int64
		valido   bool
	}{
		{"10% de R$ 1.000,00", 100000, 1000, 10000, true},
		{"2,5% de R$ 1.000,00", 100000, 250, 2500, true},
		{"5% de R$ 100,00", 10000, 500, 500, true},
		{"arredonda meio centavo para cima", 1, 5000, 1, true},
		{"0%", 100000, 0, 0, true},
		{"acima de 100%", 100000, 10001, 0, false},
		{"negativo", 100000, -1, 0, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r, err := dinheiro(t, c.base).Percentual(c.pontos)
			if c.valido {
				if err != nil || r.Centavos() != c.esperado {
					t.Fatalf("esperava %d, veio %d (erro: %v)", c.esperado, r.Centavos(), err)
				}
				return
			}
			if err != ErrPercentualInvalido {
				t.Fatalf("esperava ErrPercentualInvalido, veio %v", err)
			}
		})
	}
}

func TestStringFormataReais(t *testing.T) {
	if s := dinheiro(t, 123456).String(); s != "R$ 1234,56" {
		t.Fatalf("formato inesperado: %s", s)
	}
	if s := dinheiro(t, 5).String(); s != "R$ 0,05" {
		t.Fatalf("formato inesperado: %s", s)
	}
}
