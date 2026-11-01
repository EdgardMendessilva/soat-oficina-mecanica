package estoque

import (
	"errors"
	"strings"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros do estoque.
var (
	ErrItemInvalido      = errors.New("o item precisa de identificador e nome")
	ErrSaldoInsuficiente = errors.New("saldo insuficiente em estoque")
	ErrQuantidadeZero    = errors.New("a quantidade da movimentação deve ser maior que zero")
)

// Tipo diferencia peça de insumo no estoque.
type Tipo string

// Tipos de item.
const (
	Peca   Tipo = "Peça"
	Insumo Tipo = "Insumo"
)

// Item é uma peça ou insumo com saldo em estoque. O saldo nunca fica negativo (RN09):
// o tipo Quantidade já impede valores abaixo de zero e a baixa recusa saldo insuficiente.
type Item struct {
	id    string
	nome  string
	tipo  Tipo
	saldo valueobject.Quantidade
}

// NovoItem cria um item de estoque com o saldo inicial informado.
func NovoItem(id, nome string, tipo Tipo, saldoInicial valueobject.Quantidade) (*Item, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(nome) == "" {
		return nil, ErrItemInvalido
	}
	if tipo != Peca && tipo != Insumo {
		return nil, ErrItemInvalido
	}
	return &Item{id: id, nome: nome, tipo: tipo, saldo: saldoInicial}, nil
}

// ID devolve o identificador do item.
func (i *Item) ID() string { return i.id }

// Nome devolve o nome do item.
func (i *Item) Nome() string { return i.nome }

// Tipo devolve se o item é peça ou insumo.
func (i *Item) Tipo() Tipo { return i.tipo }

// Saldo devolve a quantidade disponível.
func (i *Item) Saldo() valueobject.Quantidade { return i.saldo }

// Repor soma a quantidade ao saldo (entrada de estoque).
func (i *Item) Repor(q valueobject.Quantidade) error {
	if q.Valor() == 0 {
		return ErrQuantidadeZero
	}
	novo, err := valueobject.NovaQuantidade(i.saldo.Valor() + q.Valor())
	if err != nil {
		return err
	}
	i.saldo = novo
	return nil
}

// Baixar subtrai a quantidade do saldo. Sem saldo suficiente, recusa e nada muda (RN09).
func (i *Item) Baixar(q valueobject.Quantidade) error {
	if q.Valor() == 0 {
		return ErrQuantidadeZero
	}
	if q.Valor() > i.saldo.Valor() {
		return ErrSaldoInsuficiente
	}
	novo, err := valueobject.NovaQuantidade(i.saldo.Valor() - q.Valor())
	if err != nil {
		return err
	}
	i.saldo = novo
	return nil
}
