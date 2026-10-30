package valueobject

import (
	"errors"
	"fmt"
)

// Erros de Dinheiro.
var (
	ErrValorNegativo      = errors.New("valor em dinheiro não pode ser negativo")
	ErrPercentualInvalido = errors.New("percentual inválido: use de 0 a 10000 pontos-base (0% a 100%)")
)

// Dinheiro guarda um valor em reais como inteiro de centavos, para evitar erro de ponto flutuante.
type Dinheiro struct{ centavos int64 }

// NovoDinheiro cria um valor a partir de centavos; valores negativos são recusados.
func NovoDinheiro(centavos int64) (Dinheiro, error) {
	if centavos < 0 {
		return Dinheiro{}, ErrValorNegativo
	}
	return Dinheiro{centavos: centavos}, nil
}

// Centavos devolve o valor em centavos.
func (d Dinheiro) Centavos() int64 { return d.centavos }

// Somar devolve a soma de dois valores.
func (d Dinheiro) Somar(o Dinheiro) Dinheiro { return Dinheiro{centavos: d.centavos + o.centavos} }

// Multiplicar devolve o valor multiplicado por uma quantidade.
func (d Dinheiro) Multiplicar(q Quantidade) Dinheiro {
	return Dinheiro{centavos: d.centavos * int64(q.Valor())}
}

// Subtrair devolve a diferença; falha se o resultado ficaria negativo.
func (d Dinheiro) Subtrair(o Dinheiro) (Dinheiro, error) {
	return NovoDinheiro(d.centavos - o.centavos)
}

// Percentual devolve a parte do valor correspondente a pontos-base (1000 = 10%),
// arredondando meio centavo para cima.
func (d Dinheiro) Percentual(pontosBase int64) (Dinheiro, error) {
	if pontosBase < 0 || pontosBase > 10000 {
		return Dinheiro{}, ErrPercentualInvalido
	}
	return Dinheiro{centavos: (d.centavos*pontosBase + 5000) / 10000}, nil
}

// String formata o valor como R$ 1234,56.
func (d Dinheiro) String() string {
	return fmt.Sprintf("R$ %d,%02d", d.centavos/100, d.centavos%100)
}
