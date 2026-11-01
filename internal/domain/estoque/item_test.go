package estoque

import (
	"testing"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

func qtd(t *testing.T, n int) valueobject.Quantidade {
	t.Helper()
	q, err := valueobject.NovaQuantidade(n)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func item(t *testing.T, saldo int) *Item {
	t.Helper()
	i, err := NovoItem("peca-1", "Filtro de óleo", Peca, qtd(t, saldo))
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestNovoItem(t *testing.T) {
	i := item(t, 10)
	if i.ID() != "peca-1" || i.Nome() != "Filtro de óleo" || i.Tipo() != Peca || i.Saldo().Valor() != 10 {
		t.Fatalf("item inesperado: %+v", i)
	}
	invalidos := []struct {
		nome, id, n string
		tipo        Tipo
	}{
		{"sem id", "", "x", Peca},
		{"sem nome", "id", " ", Peca},
		{"tipo desconhecido", "id", "x", "Outro"},
	}
	for _, c := range invalidos {
		if _, err := NovoItem(c.id, c.n, c.tipo, qtd(t, 1)); err != ErrItemInvalido {
			t.Fatalf("%s: esperava ErrItemInvalido, veio %v", c.nome, err)
		}
	}
	if _, err := NovoItem("id", "Graxa", Insumo, qtd(t, 0)); err != nil {
		t.Fatalf("insumo com saldo zero deveria ser válido: %v", err)
	}
}

func TestBaixar(t *testing.T) {
	i := item(t, 10)
	if err := i.Baixar(qtd(t, 4)); err != nil || i.Saldo().Valor() != 6 {
		t.Fatalf("baixa de 4: saldo esperado 6, veio %d (erro: %v)", i.Saldo().Valor(), err)
	}
	if err := i.Baixar(qtd(t, 6)); err != nil || i.Saldo().Valor() != 0 {
		t.Fatalf("baixa do saldo inteiro: saldo esperado 0, veio %d (erro: %v)", i.Saldo().Valor(), err)
	}
}

func TestBaixarSemSaldoNaoAlteraNada(t *testing.T) {
	i := item(t, 3)
	if err := i.Baixar(qtd(t, 4)); err != ErrSaldoInsuficiente {
		t.Fatalf("esperava ErrSaldoInsuficiente, veio %v", err)
	}
	if i.Saldo().Valor() != 3 {
		t.Fatalf("o saldo não pode mudar quando a baixa é recusada, veio %d", i.Saldo().Valor())
	}
}

func TestMovimentacaoZeroEhRecusada(t *testing.T) {
	i := item(t, 3)
	if err := i.Baixar(qtd(t, 0)); err != ErrQuantidadeZero {
		t.Fatalf("baixa zero: esperava ErrQuantidadeZero, veio %v", err)
	}
	if err := i.Repor(qtd(t, 0)); err != ErrQuantidadeZero {
		t.Fatalf("reposição zero: esperava ErrQuantidadeZero, veio %v", err)
	}
}

func TestRepor(t *testing.T) {
	i := item(t, 3)
	if err := i.Repor(qtd(t, 7)); err != nil || i.Saldo().Valor() != 10 {
		t.Fatalf("reposição de 7: saldo esperado 10, veio %d (erro: %v)", i.Saldo().Valor(), err)
	}
}
