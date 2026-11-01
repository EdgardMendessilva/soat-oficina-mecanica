package cadastro

import (
	"strings"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Veiculo é o automóvel de um cliente, identificado pela placa (RN02).
type Veiculo struct {
	id         string
	clienteID  string
	placa      valueobject.Placa
	fabricante string
	modelo     string
	ano        int
	cor        string
}

const (
	anoMinimo = 1900
	anoMaximo = 2100
)

// NovoVeiculo valida os dados e cria o veículo. A placa segue o padrão antigo ou o Mercosul.
func NovoVeiculo(id, clienteID, placa, fabricante, modelo string, ano int, cor string) (*Veiculo, error) {
	fabricante, modelo = strings.TrimSpace(fabricante), strings.TrimSpace(modelo)
	if strings.TrimSpace(id) == "" || strings.TrimSpace(clienteID) == "" || fabricante == "" || modelo == "" {
		return nil, ErrDadosDoVeiculo
	}
	p, err := valueobject.NovaPlaca(placa)
	if err != nil {
		return nil, err
	}
	if ano < anoMinimo || ano > anoMaximo {
		return nil, ErrAnoInvalido
	}
	return &Veiculo{id: id, clienteID: clienteID, placa: p, fabricante: fabricante, modelo: modelo, ano: ano, cor: strings.TrimSpace(cor)}, nil
}

// Getters.
func (v *Veiculo) ID() string               { return v.id }
func (v *Veiculo) ClienteID() string        { return v.clienteID }
func (v *Veiculo) Placa() valueobject.Placa { return v.placa }
func (v *Veiculo) Fabricante() string       { return v.fabricante }
func (v *Veiculo) Modelo() string           { return v.modelo }
func (v *Veiculo) Ano() int                 { return v.ano }
func (v *Veiculo) Cor() string              { return v.cor }
