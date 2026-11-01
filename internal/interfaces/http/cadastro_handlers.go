package http

import (
	"net/http"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/catalogo"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
)

type clienteView struct {
	ID        string `json:"id"`
	Documento string `json:"documento"`
	Tipo      string `json:"tipoDocumento"`
	Nome      string `json:"nome"`
	Email     string `json:"email"`
	Telefone  string `json:"telefone"`
	Endereco  string `json:"endereco"`
}

func paraCliente(c *cadastro.Cliente) clienteView {
	return clienteView{c.ID(), c.Documento().Numero(), string(c.Documento().Tipo()), c.Nome(), c.Email(), c.Telefone(), c.Endereco()}
}

type veiculoView struct {
	ID         string `json:"id"`
	ClienteID  string `json:"clienteId"`
	Placa      string `json:"placa"`
	Fabricante string `json:"fabricante"`
	Modelo     string `json:"modelo"`
	Ano        int    `json:"ano"`
	Cor        string `json:"cor"`
}

func paraVeiculo(v *cadastro.Veiculo) veiculoView {
	return veiculoView{v.ID(), v.ClienteID(), v.Placa().String(), v.Fabricante(), v.Modelo(), v.Ano(), v.Cor()}
}

type servicoCatalogoView struct {
	ID            string `json:"id"`
	Nome          string `json:"nome"`
	PrecoCentavos int64  `json:"precoCentavos"`
}

func paraServicoCatalogo(s *catalogo.Servico) servicoCatalogoView {
	return servicoCatalogoView{s.ID(), s.Nome(), s.Preco().Centavos()}
}

func ehCliente(r *http.Request) bool {
	return identidade.Perfil(usuarioDe(r).Perfil) == identidade.Cliente
}

func (s *Servidor) postCliente(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Documento string `json:"documento"`
		Nome      string `json:"nome"`
		Email     string `json:"email"`
		Telefone  string `json:"telefone"`
		Endereco  string `json:"endereco"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	c, err := s.cadastro.CadastrarCliente(r.Context(), novoID("cli"), req.Documento, req.Nome, req.Email, req.Telefone, req.Endereco)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusCreated, paraCliente(c))
}

func (s *Servidor) getClientes(w http.ResponseWriter, r *http.Request) {
	lista, err := s.cadastro.ListarClientes(r.Context())
	if err != nil {
		responderErro(w, err)
		return
	}
	out := make([]clienteView, 0, len(lista))
	for _, c := range lista {
		out = append(out, paraCliente(c))
	}
	escrever(w, http.StatusOK, out)
}

// getCliente: o cliente só enxerga o próprio cadastro (RN10).
func (s *Servidor) getCliente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if ehCliente(r) && id != usuarioDe(r).ClienteID {
		responderErro(w, application.ErrAcessoNegado)
		return
	}
	c, err := s.cadastro.ConsultarCliente(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraCliente(c))
}

func (s *Servidor) postVeiculo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClienteID  string `json:"clienteId"`
		Placa      string `json:"placa"`
		Fabricante string `json:"fabricante"`
		Modelo     string `json:"modelo"`
		Ano        int    `json:"ano"`
		Cor        string `json:"cor"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	v, err := s.cadastro.CadastrarVeiculo(r.Context(), novoID("vei"), req.ClienteID, req.Placa, req.Fabricante, req.Modelo, req.Ano, req.Cor)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusCreated, paraVeiculo(v))
}

// getVeiculos: o cliente lista os próprios; o funcionário informa ?clienteId=.
func (s *Servidor) getVeiculos(w http.ResponseWriter, r *http.Request) {
	clienteID := r.URL.Query().Get("clienteId")
	if ehCliente(r) {
		clienteID = usuarioDe(r).ClienteID
	} else if clienteID == "" {
		escreverErro(w, http.StatusBadRequest, "parametro_obrigatorio", "informe ?clienteId=")
		return
	}
	lista, err := s.cadastro.ListarVeiculosDoCliente(r.Context(), clienteID)
	if err != nil {
		responderErro(w, err)
		return
	}
	out := make([]veiculoView, 0, len(lista))
	for _, v := range lista {
		out = append(out, paraVeiculo(v))
	}
	escrever(w, http.StatusOK, out)
}

func (s *Servidor) getVeiculo(w http.ResponseWriter, r *http.Request) {
	v, err := s.cadastro.ConsultarVeiculo(r.Context(), r.PathValue("id"))
	if err != nil {
		responderErro(w, err)
		return
	}
	if ehCliente(r) && v.ClienteID() != usuarioDe(r).ClienteID {
		responderErro(w, application.ErrAcessoNegado)
		return
	}
	escrever(w, http.StatusOK, paraVeiculo(v))
}

type corpoServico struct {
	Nome          string `json:"nome"`
	PrecoCentavos int64  `json:"precoCentavos"`
}

func (s *Servidor) postServicoCatalogo(w http.ResponseWriter, r *http.Request) {
	var req corpoServico
	if !lerJSON(w, r, &req) {
		return
	}
	sv, err := s.catalogo.Criar(r.Context(), novoID("srv"), req.Nome, req.PrecoCentavos)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusCreated, paraServicoCatalogo(sv))
}

func (s *Servidor) getServicosCatalogo(w http.ResponseWriter, r *http.Request) {
	lista, err := s.catalogo.Listar(r.Context())
	if err != nil {
		responderErro(w, err)
		return
	}
	out := make([]servicoCatalogoView, 0, len(lista))
	for _, sv := range lista {
		out = append(out, paraServicoCatalogo(sv))
	}
	escrever(w, http.StatusOK, out)
}

func (s *Servidor) getServicoCatalogo(w http.ResponseWriter, r *http.Request) {
	sv, err := s.catalogo.Consultar(r.Context(), r.PathValue("id"))
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraServicoCatalogo(sv))
}

func (s *Servidor) putServicoCatalogo(w http.ResponseWriter, r *http.Request) {
	var req corpoServico
	if !lerJSON(w, r, &req) {
		return
	}
	sv, err := s.catalogo.Alterar(r.Context(), r.PathValue("id"), req.Nome, req.PrecoCentavos)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraServicoCatalogo(sv))
}

func (s *Servidor) deleteServicoCatalogo(w http.ResponseWriter, r *http.Request) {
	if err := s.catalogo.Remover(r.Context(), r.PathValue("id")); err != nil {
		responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
