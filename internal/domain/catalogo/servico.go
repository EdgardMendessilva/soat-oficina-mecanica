// Package catalogo reúne os serviços que a oficina oferece e seus preços.
package catalogo

import (
	"errors"
	"strings"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// ErrServicoInvalido indica serviço sem identificador ou sem nome.
var ErrServicoInvalido = errors.New("o serviço precisa de identificador e nome")

// Servico é um serviço de mão de obra do catálogo, com o preço de tabela.
type Servico struct {
	id    string
	nome  string
	preco valueobject.Dinheiro
}

// NovoServico valida e cria o serviço do catálogo.
func NovoServico(id, nome string, preco valueobject.Dinheiro) (*Servico, error) {
	nome = strings.TrimSpace(nome)
	if strings.TrimSpace(id) == "" || nome == "" {
		return nil, ErrServicoInvalido
	}
	return &Servico{id: id, nome: nome, preco: preco}, nil
}

// Alterar troca o nome e o preço do serviço.
func (s *Servico) Alterar(nome string, preco valueobject.Dinheiro) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return ErrServicoInvalido
	}
	s.nome, s.preco = nome, preco
	return nil
}

// Getters.
func (s *Servico) ID() string                  { return s.id }
func (s *Servico) Nome() string                { return s.nome }
func (s *Servico) Preco() valueobject.Dinheiro { return s.preco }
