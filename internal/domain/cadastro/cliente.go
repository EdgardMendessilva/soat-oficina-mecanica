package cadastro

import (
	"errors"
	"strings"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Erros do cadastro.
var (
	ErrNomeObrigatorio = errors.New("o nome é obrigatório")
	ErrEmailInvalido   = errors.New("e-mail inválido")
	ErrAnoInvalido     = errors.New("ano do veículo inválido")
	ErrDadosDoVeiculo  = errors.New("identificador, cliente, fabricante e modelo do veículo são obrigatórios")
)

// Cliente é a pessoa ou empresa dona dos veículos, identificada por CPF ou CNPJ (RN01).
type Cliente struct {
	id        string
	documento valueobject.Documento
	nome      string
	email     string
	telefone  string
	endereco  string
}

// NovoCliente valida os dados e cria o cliente. O CPF ou CNPJ precisa ter dígitos verificadores válidos.
func NovoCliente(id, documento, nome, email, telefone, endereco string) (*Cliente, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrDadosDoVeiculo
	}
	doc, err := valueobject.NovoDocumento(documento)
	if err != nil {
		return nil, err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, ErrNomeObrigatorio
	}
	email = strings.TrimSpace(email)
	if at := strings.Index(email, "@"); at < 1 || !strings.Contains(email[at:], ".") || strings.ContainsAny(email, " \t") {
		return nil, ErrEmailInvalido
	}
	return &Cliente{id: id, documento: doc, nome: nome, email: email,
		telefone: strings.TrimSpace(telefone), endereco: strings.TrimSpace(endereco)}, nil
}

// Getters.
func (c *Cliente) ID() string                       { return c.id }
func (c *Cliente) Documento() valueobject.Documento { return c.documento }
func (c *Cliente) Nome() string                     { return c.nome }
func (c *Cliente) Email() string                    { return c.email }
func (c *Cliente) Telefone() string                 { return c.telefone }
func (c *Cliente) Endereco() string                 { return c.endereco }
