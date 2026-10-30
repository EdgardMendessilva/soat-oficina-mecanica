package valueobject

import "errors"

// ErrQuantidadeNegativa indica uma quantidade menor que zero.
var ErrQuantidadeNegativa = errors.New("quantidade não pode ser negativa")

// Quantidade é um número inteiro maior ou igual a zero (peças, insumos, saldo).
type Quantidade struct{ valor int }

// NovaQuantidade cria uma quantidade; valores negativos são recusados.
func NovaQuantidade(n int) (Quantidade, error) {
	if n < 0 {
		return Quantidade{}, ErrQuantidadeNegativa
	}
	return Quantidade{valor: n}, nil
}

// Valor devolve o número inteiro da quantidade.
func (q Quantidade) Valor() int { return q.valor }
