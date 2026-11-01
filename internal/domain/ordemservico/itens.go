package ordemservico

import (
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros de itens da OS.
var (
	ErrDescricaoVazia       = errors.New("a descrição é obrigatória")
	ErrQuantidadeInvalida   = errors.New("a quantidade da peça deve ser maior que zero")
	ErrItensSoNoDiagnostico = errors.New("serviços e peças só podem mudar durante o diagnóstico")
)

// Servico é um serviço de mão de obra incluído na OS.
type Servico struct {
	Descricao string
	Preco     valueobject.Dinheiro
}

// Peca é uma peça ou insumo incluído na OS, com o preço unitário e a quantidade.
type Peca struct {
	PecaID        string
	Descricao     string
	PrecoUnitario valueobject.Dinheiro
	Quantidade    valueobject.Quantidade
}

// Subtotal devolve o preço unitário multiplicado pela quantidade.
func (p Peca) Subtotal() valueobject.Dinheiro {
	return p.PrecoUnitario.Multiplicar(p.Quantidade)
}
