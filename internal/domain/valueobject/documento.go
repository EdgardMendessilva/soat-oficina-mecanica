// Package valueobject reúne os objetos de valor compartilhados pelos contextos do domínio.
package valueobject

import (
	"errors"
	"strings"
)

// Erros de validação de Documento.
var (
	ErrDocumentoInvalido = errors.New("documento inválido: informe um CPF (11 dígitos) ou CNPJ (14 dígitos) com dígitos verificadores corretos")
)

// TipoDocumento indica se o documento é um CPF ou um CNPJ.
type TipoDocumento string

// Tipos de documento aceitos.
const (
	CPF  TipoDocumento = "CPF"
	CNPJ TipoDocumento = "CNPJ"
)

// Documento é um CPF ou CNPJ válido, guardado só com dígitos.
type Documento struct {
	numero string
	tipo   TipoDocumento
}

// NovoDocumento valida e normaliza um CPF ou CNPJ. Pontos, traço e barra são aceitos na entrada.
func NovoDocumento(entrada string) (Documento, error) {
	numero := soDigitos(entrada)
	switch len(numero) {
	case 11:
		if !cpfValido(numero) {
			return Documento{}, ErrDocumentoInvalido
		}
		return Documento{numero: numero, tipo: CPF}, nil
	case 14:
		if !cnpjValido(numero) {
			return Documento{}, ErrDocumentoInvalido
		}
		return Documento{numero: numero, tipo: CNPJ}, nil
	default:
		return Documento{}, ErrDocumentoInvalido
	}
}

// Numero devolve o documento só com dígitos.
func (d Documento) Numero() string { return d.numero }

// Tipo devolve CPF ou CNPJ.
func (d Documento) Tipo() TipoDocumento { return d.tipo }

// String devolve o documento só com dígitos.
func (d Documento) String() string { return d.numero }

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r != '.' && r != '-' && r != '/' && r != ' ' {
			return "" // qualquer outro caractere invalida a entrada
		}
	}
	return b.String()
}

func todosIguais(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func digito(s string, pesos []int) int {
	soma := 0
	for i, p := range pesos {
		soma += int(s[i]-'0') * p
	}
	resto := soma % 11
	if resto < 2 {
		return 0
	}
	return 11 - resto
}

func cpfValido(n string) bool {
	if todosIguais(n) {
		return false
	}
	d1 := digito(n, []int{10, 9, 8, 7, 6, 5, 4, 3, 2})
	d2 := digito(n, []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2})
	return int(n[9]-'0') == d1 && int(n[10]-'0') == d2
}

func cnpjValido(n string) bool {
	if todosIguais(n) {
		return false
	}
	d1 := digito(n, []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	d2 := digito(n, []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	return int(n[12]-'0') == d1 && int(n[13]-'0') == d2
}
