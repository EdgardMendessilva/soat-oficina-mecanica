package http

import (
	"context"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
)

type ctxCtx = context.Context

type servicoView struct {
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"precoCentavos"`
}

type pecaView struct {
	PecaID                string `json:"pecaId"`
	Descricao             string `json:"descricao"`
	PrecoUnitarioCentavos int64  `json:"precoUnitarioCentavos"`
	Quantidade            int    `json:"quantidade"`
	SubtotalCentavos      int64  `json:"subtotalCentavos"`
}

type descontoView struct {
	PontosBase    int64  `json:"pontosBase"`
	Base          string `json:"base"`
	ValorCentavos int64  `json:"valorCentavos"`
	AutorizadoPor string `json:"autorizadoPor"`
	Em            string `json:"em"`
}

type historicoView struct {
	De   string `json:"de,omitempty"`
	Para string `json:"para"`
	Em   string `json:"em"`
}

type ordemView struct {
	ID                       string          `json:"id"`
	ClienteID                string          `json:"clienteId"`
	VeiculoID                string          `json:"veiculoId"`
	Status                   string          `json:"status"`
	Decisao                  string          `json:"decisao"`
	ValorDoOrcamentoCentavos int64           `json:"valorDoOrcamentoCentavos"`
	MaoDeObraCentavos        int64           `json:"maoDeObraCentavos"`
	TotalCentavos            int64           `json:"totalCentavos"`
	ValorAPagarCentavos      *int64          `json:"valorAPagarCentavos,omitempty"`
	Servicos                 []servicoView   `json:"servicos"`
	Pecas                    []pecaView      `json:"pecas"`
	Desconto                 *descontoView   `json:"desconto,omitempty"`
	Historico                []historicoView `json:"historico"`
}

func paraView(o *ordemservico.OrdemDeServico) ordemView {
	v := ordemView{
		ID: o.ID(), ClienteID: o.ClienteID(), VeiculoID: o.VeiculoID(),
		Status: string(o.Status()), Decisao: string(o.Decisao()),
		ValorDoOrcamentoCentavos: o.ValorDoOrcamento().Centavos(),
		MaoDeObraCentavos:        o.MaoDeObra().Centavos(),
		TotalCentavos:            o.TotalDoOrcamento().Centavos(),
		Servicos:                 []servicoView{}, Pecas: []pecaView{}, Historico: []historicoView{},
	}
	if pagar, err := o.ValorAPagar(); err == nil {
		c := pagar.Centavos()
		v.ValorAPagarCentavos = &c
	}
	for _, s := range o.Servicos() {
		v.Servicos = append(v.Servicos, servicoView{s.Descricao, s.Preco.Centavos()})
	}
	for _, p := range o.Pecas() {
		v.Pecas = append(v.Pecas, pecaView{p.PecaID, p.Descricao, p.PrecoUnitario.Centavos(), p.Quantidade.Valor(), p.Subtotal().Centavos()})
	}
	if d := o.Desconto(); d != nil {
		v.Desconto = &descontoView{d.PontosBase, string(d.Base), d.Valor.Centavos(), d.AutorizadoPor, d.Em.UTC().Format(time.RFC3339)}
	}
	for _, h := range o.Historico() {
		v.Historico = append(v.Historico, historicoView{string(h.De), string(h.Para), h.Em.UTC().Format(time.RFC3339)})
	}
	return v
}
