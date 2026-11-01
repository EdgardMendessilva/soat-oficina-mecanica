package ordemservico

import (
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Dados é o estado completo de uma OS já existente, usado pela infraestrutura para
// reconstituí-la a partir do armazenamento.
type Dados struct {
	ID, ClienteID, VeiculoID string
	ValorDoOrcamento         valueobject.Dinheiro
	Status                   Status
	Servicos                 []Servico
	Pecas                    []Peca
	Decisao                  Decisao
	DecididoPor              string
	DecididoEm               time.Time
	Desconto                 *Desconto
	Historico                []MudancaDeStatus
}

// Reconstituir recria a OS com o estado guardado, sem reaplicar as regras de transição:
// quem chama só deve passar dados que já foram validados pelo agregado ao serem gravados.
func Reconstituir(d Dados) *OrdemDeServico {
	return &OrdemDeServico{
		id: d.ID, clienteID: d.ClienteID, veiculoID: d.VeiculoID,
		valorDoOrcamento: d.ValorDoOrcamento,
		status:           d.Status,
		servicos:         append([]Servico(nil), d.Servicos...),
		pecas:            append([]Peca(nil), d.Pecas...),
		decisao:          d.Decisao,
		decididoPor:      d.DecididoPor,
		decididoEm:       d.DecididoEm,
		desconto:         d.Desconto,
		historico:        append([]MudancaDeStatus(nil), d.Historico...),
	}
}
