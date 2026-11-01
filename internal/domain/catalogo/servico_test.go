package catalogo

import (
	"errors"
	"testing"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

func TestServicoDoCatalogo(t *testing.T) {
	preco, _ := valueobject.NovoDinheiro(15000)
	s, err := NovoServico("s1", " Troca de óleo ", preco)
	if err != nil || s.Nome() != "Troca de óleo" || s.Preco().Centavos() != 15000 {
		t.Fatalf("serviço errado: %v %+v", err, s)
	}
	novo, _ := valueobject.NovoDinheiro(18000)
	if err := s.Alterar("Troca de óleo e filtro", novo); err != nil || s.Preco().Centavos() != 18000 {
		t.Fatalf("alteração falhou: %v", err)
	}
	if err := s.Alterar("  ", novo); !errors.Is(err, ErrServicoInvalido) {
		t.Errorf("aceitou nome vazio: %v", err)
	}
	if _, err := NovoServico("", "x", preco); !errors.Is(err, ErrServicoInvalido) {
		t.Errorf("aceitou sem id: %v", err)
	}
}
