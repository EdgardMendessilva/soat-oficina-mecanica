package valueobject

import (
	"errors"
	"regexp"
	"strings"
)

// ErrPlacaInvalida indica que a placa não segue o padrão antigo nem o Mercosul.
var ErrPlacaInvalida = errors.New("placa inválida: use o padrão antigo (ABC1234) ou Mercosul (ABC1D23)")

var (
	padraoAntigo   = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)
	padraoMercosul = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
)

// Placa é uma placa de veículo válida, em maiúsculas e sem hífen.
type Placa struct{ valor string }

// NovaPlaca valida e normaliza uma placa. Hífen e minúsculas são aceitos na entrada.
func NovaPlaca(entrada string) (Placa, error) {
	v := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(entrada), "-", ""))
	if !padraoAntigo.MatchString(v) && !padraoMercosul.MatchString(v) {
		return Placa{}, ErrPlacaInvalida
	}
	return Placa{valor: v}, nil
}

// String devolve a placa normalizada.
func (p Placa) String() string { return p.valor }
