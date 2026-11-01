package ordemservico

import (
	"errors"
	"strings"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// TetoDescontoPontosBase é o teto de 10% do desconto (1000 pontos-base), RN07.
const TetoDescontoPontosBase int64 = 1000

// Erros de desconto.
var (
	ErrDescontoSemOrcamentoFechado = errors.New("o desconto só vale com o orçamento fechado (aprovado pelo cliente)")
	ErrDescontoAcimaDoTeto         = errors.New("desconto acima do teto de 10%")
	ErrDescontoSemAutorizacao      = errors.New("o desconto precisa da autorização do gerente")
	ErrDescontoJaAplicado          = errors.New("a OS já tem desconto aplicado")
	ErrBaseDeDescontoInvalida      = errors.New("base de cálculo do desconto inválida")
)

// BaseDesconto indica sobre o que o percentual incide (RN07).
type BaseDesconto string

// Bases de cálculo aceitas.
const (
	BaseTotal     BaseDesconto = "serviços + peças"
	BaseMaoDeObra BaseDesconto = "mão de obra"
)

// Desconto guarda o que foi concedido e quem autorizou (RN07).
type Desconto struct {
	PontosBase    int64
	Base          BaseDesconto
	Valor         valueobject.Dinheiro
	AutorizadoPor string
	Em            time.Time
}

// AplicarDesconto concede o desconto. A checagem da senha do gerente é feita na camada
// de aplicação; aqui o domínio exige o ID do gerente que autorizou e registra quem, quando,
// o percentual e a base. Só vale com o orçamento fechado, até o teto de 10%.
func (o *OrdemDeServico) AplicarDesconto(pontosBase int64, base BaseDesconto, gerenteID string, agora time.Time) error {
	if o.decisao != Aprovado {
		return ErrDescontoSemOrcamentoFechado
	}
	if o.desconto != nil {
		return ErrDescontoJaAplicado
	}
	if strings.TrimSpace(gerenteID) == "" {
		return ErrDescontoSemAutorizacao
	}
	if pontosBase > TetoDescontoPontosBase {
		return ErrDescontoAcimaDoTeto
	}
	var valorBase valueobject.Dinheiro
	switch base {
	case BaseTotal:
		valorBase = o.TotalDoOrcamento()
	case BaseMaoDeObra:
		valorBase = o.MaoDeObra()
	default:
		return ErrBaseDeDescontoInvalida
	}
	valor, err := valorBase.Percentual(pontosBase)
	if err != nil {
		return err
	}
	o.desconto = &Desconto{PontosBase: pontosBase, Base: base, Valor: valor, AutorizadoPor: gerenteID, Em: agora}
	return nil
}
