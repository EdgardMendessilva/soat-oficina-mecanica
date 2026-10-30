package valueobject

import "testing"

func TestNovaQuantidade(t *testing.T) {
	for _, n := range []int{0, 1, 250} {
		q, err := NovaQuantidade(n)
		if err != nil || q.Valor() != n {
			t.Fatalf("quantidade %d deveria ser válida (erro: %v)", n, err)
		}
	}
	if _, err := NovaQuantidade(-1); err != ErrQuantidadeNegativa {
		t.Fatalf("esperava ErrQuantidadeNegativa, veio %v", err)
	}
}
