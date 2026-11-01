package cadastro

import (
	"errors"
	"testing"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

func TestNovoClienteAceitaCPFeCNPJ(t *testing.T) {
	for _, doc := range []string{"529.982.247-25", "11.222.333/0001-81"} {
		c, err := NovoCliente("c1", doc, " Maria ", "maria@mail.com", "11999990000", "Rua A")
		if err != nil {
			t.Fatalf("%s: %v", doc, err)
		}
		if c.Nome() != "Maria" || c.Email() != "maria@mail.com" || c.Documento().Numero() == "" {
			t.Errorf("dados errados: %+v", c)
		}
	}
}

func TestNovoClienteRecusaDadosInvalidos(t *testing.T) {
	casos := []struct {
		nome                  string
		id, doc, nomeC, email string
		esperado              error
	}{
		{"CPF inválido", "c1", "111.111.111-11", "Maria", "m@x.com", valueobject.ErrDocumentoInvalido},
		{"sem nome", "c1", "529.982.247-25", "  ", "m@x.com", ErrNomeObrigatorio},
		{"e-mail sem arroba", "c1", "529.982.247-25", "Maria", "maria.mail.com", ErrEmailInvalido},
		{"e-mail sem domínio", "c1", "529.982.247-25", "Maria", "maria@mail", ErrEmailInvalido},
		{"sem id", "", "529.982.247-25", "Maria", "m@x.com", ErrDadosDoVeiculo},
	}
	for _, c := range casos {
		if _, err := NovoCliente(c.id, c.doc, c.nomeC, c.email, "", ""); !errors.Is(err, c.esperado) {
			t.Errorf("%s: esperava %v, veio %v", c.nome, c.esperado, err)
		}
	}
}

func TestNovoVeiculo(t *testing.T) {
	v, err := NovoVeiculo("v1", "c1", "abc-1d23", " Renault ", "Kwid", 2023, "Branco")
	if err != nil {
		t.Fatal(err)
	}
	if v.Placa().String() != "ABC1D23" || v.Fabricante() != "Renault" || v.Ano() != 2023 || v.ClienteID() != "c1" {
		t.Errorf("dados errados: %+v", v)
	}
}

func TestNovoVeiculoRecusaDadosInvalidos(t *testing.T) {
	casos := []struct {
		nome                     string
		id, cli, placa, fab, mod string
		ano                      int
		esperado                 error
	}{
		{"placa inválida", "v1", "c1", "ZZZ", "Renault", "Kwid", 2023, valueobject.ErrPlacaInvalida},
		{"ano antigo", "v1", "c1", "ABC1234", "Renault", "Kwid", 1800, ErrAnoInvalido},
		{"ano futuro demais", "v1", "c1", "ABC1234", "Renault", "Kwid", 3000, ErrAnoInvalido},
		{"sem fabricante", "v1", "c1", "ABC1234", "", "Kwid", 2023, ErrDadosDoVeiculo},
		{"sem modelo", "v1", "c1", "ABC1234", "Renault", "", 2023, ErrDadosDoVeiculo},
		{"sem cliente", "v1", "", "ABC1234", "Renault", "Kwid", 2023, ErrDadosDoVeiculo},
	}
	for _, c := range casos {
		if _, err := NovoVeiculo(c.id, c.cli, c.placa, c.fab, c.mod, c.ano, ""); !errors.Is(err, c.esperado) {
			t.Errorf("%s: esperava %v, veio %v", c.nome, c.esperado, err)
		}
	}
}
