package ordemservico

import (
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

func TestReconstituirPreservaOEstado(t *testing.T) {
	agora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	orc, _ := valueobject.NovoDinheiro(5000)
	preco, _ := valueobject.NovoDinheiro(20000)
	q, _ := valueobject.NovaQuantidade(2)
	desc := &Desconto{PontosBase: 500, Base: BaseTotal, Valor: preco, AutorizadoPor: "ger-1", Em: agora}

	o := Reconstituir(Dados{
		ID: "os-1", ClienteID: "cli-1", VeiculoID: "v-1", ValorDoOrcamento: orc,
		Status:   EmExecucao,
		Servicos: []Servico{{Descricao: "Revisão", Preco: preco}},
		Pecas:    []Peca{{PecaID: "p-1", Descricao: "Filtro", PrecoUnitario: preco, Quantidade: q}},
		Decisao:  Aprovado, DecididoPor: "cli-1", DecididoEm: agora, Desconto: desc,
		Historico: []MudancaDeStatus{{Para: Recebida, Em: agora}},
	})

	if o.ID() != "os-1" || o.Status() != EmExecucao || o.Decisao() != Aprovado || o.Desconto() != desc {
		t.Fatalf("estado não preservado: %+v", o)
	}
	if len(o.Servicos()) != 1 || len(o.Pecas()) != 1 || len(o.Historico()) != 1 {
		t.Fatal("itens ou histórico não preservados")
	}
	// segue as regras do agregado a partir do estado recuperado
	if err := o.Finalizar(agora.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if o.Status() != Finalizada {
		t.Fatalf("status = %s", o.Status())
	}
}
