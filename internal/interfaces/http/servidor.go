// Package http expõe a API REST: rotas, autenticação JWT e tradução de erros.
package http

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

//go:embed openapi.json
var openapi []byte

// Servidor junta os casos de uso e monta as rotas.
type Servidor struct {
	ordens   *application.OrdemDeServicoService
	estoque  *application.EstoqueService
	login    *application.AutenticacaoService
	cadastro *application.CadastroService
	catalogo *application.CatalogoService
	jwt      *seguranca.EmissorJWT
}

// NovoServidor monta o servidor.
func NovoServidor(o *application.OrdemDeServicoService, e *application.EstoqueService, l *application.AutenticacaoService, c *application.CadastroService, cat *application.CatalogoService, j *seguranca.EmissorJWT) *Servidor {
	return &Servidor{ordens: o, estoque: e, login: l, cadastro: c, catalogo: cat, jwt: j}
}

// Perfis que podem cada tipo de ação (seção 3 do escopo). Gerente e Administrador
// herdam os papéis de funcionário; o desconto é só do gerente (RN07).
var (
	atendimento = []identidade.Perfil{identidade.Atendente, identidade.Gerente, identidade.Administrador}
	oficina     = []identidade.Perfil{identidade.Mecanico, identidade.Gerente, identidade.Administrador}
	funcionario = []identidade.Perfil{identidade.Atendente, identidade.Mecanico, identidade.Gerente, identidade.Administrador}
	gerencia    = []identidade.Perfil{identidade.Gerente}
)

// Rotas devolve o handler com todas as rotas.
func (s *Servidor) Rotas() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		escrever(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openapi)
	})
	m.HandleFunc("GET /swagger", paginaSwagger)
	m.HandleFunc("POST /auth/login", s.postLogin)

	m.HandleFunc("POST /ordens", s.exigir(s.postOrdem, atendimento...))
	m.HandleFunc("GET /ordens", s.exigir(s.getOrdens))
	m.HandleFunc("GET /ordens/{id}", s.exigir(s.getOrdem))
	m.HandleFunc("POST /ordens/{id}/diagnostico", s.exigir(s.acao(s.ordens.IniciarDiagnostico), oficina...))
	m.HandleFunc("POST /ordens/{id}/servicos", s.exigir(s.postServico, oficina...))
	m.HandleFunc("POST /ordens/{id}/pecas", s.exigir(s.postPeca, oficina...))
	m.HandleFunc("POST /ordens/{id}/orcamento", s.exigir(s.acao(s.ordens.EnviarOrcamento), atendimento...))
	m.HandleFunc("POST /ordens/{id}/aprovacao", s.exigir(s.decisao(true), identidade.Cliente))
	m.HandleFunc("POST /ordens/{id}/reprovacao", s.exigir(s.decisao(false), identidade.Cliente))
	m.HandleFunc("POST /ordens/{id}/desconto", s.exigir(s.postDesconto, gerencia...))
	m.HandleFunc("POST /ordens/{id}/finalizacao", s.exigir(s.acao(s.ordens.Finalizar), oficina...))
	m.HandleFunc("POST /ordens/{id}/entrega", s.exigir(s.acao(s.ordens.Entregar), atendimento...))
	m.HandleFunc("GET /relatorios/tempo-medio-execucao", s.exigir(s.getTempoMedio, gerencia...))

	m.HandleFunc("POST /clientes", s.exigir(s.postCliente, atendimento...))
	m.HandleFunc("GET /clientes", s.exigir(s.getClientes, funcionario...))
	m.HandleFunc("GET /clientes/{id}", s.exigir(s.getCliente))
	m.HandleFunc("POST /veiculos", s.exigir(s.postVeiculo, atendimento...))
	m.HandleFunc("GET /veiculos", s.exigir(s.getVeiculos))
	m.HandleFunc("GET /veiculos/{id}", s.exigir(s.getVeiculo))

	m.HandleFunc("POST /servicos", s.exigir(s.postServicoCatalogo, gerencia...))
	m.HandleFunc("GET /servicos", s.exigir(s.getServicosCatalogo, funcionario...))
	m.HandleFunc("GET /servicos/{id}", s.exigir(s.getServicoCatalogo, funcionario...))
	m.HandleFunc("PUT /servicos/{id}", s.exigir(s.putServicoCatalogo, gerencia...))
	m.HandleFunc("DELETE /servicos/{id}", s.exigir(s.deleteServicoCatalogo, gerencia...))

	m.HandleFunc("POST /estoque", s.exigir(s.postItem, gerencia...))
	m.HandleFunc("GET /estoque/{id}", s.exigir(s.getItem, funcionario...))
	m.HandleFunc("POST /estoque/{id}/reposicao", s.exigir(s.postReposicao, gerencia...))
	return m
}

func lerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		escreverErro(w, http.StatusBadRequest, "json_invalido", "corpo da requisição inválido: "+err.Error())
		return false
	}
	return true
}

func (s *Servidor) postLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Documento string `json:"documento"`
		Senha     string `json:"senha"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	u, err := s.login.Login(r.Context(), req.Documento, req.Senha)
	if err != nil {
		if errors.Is(err, application.ErrCredenciaisInvalidas) {
			escreverErro(w, http.StatusUnauthorized, "credenciais_invalidas", err.Error())
			return
		}
		responderErro(w, err)
		return
	}
	tok, exp, err := s.jwt.Emitir(seguranca.Claims{Sub: u.ID, Perfil: string(u.Perfil), ClienteID: u.ClienteID})
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, map[string]any{"token": tok, "tipo": "Bearer", "expiraEm": exp.UTC().Format(time.RFC3339), "perfil": u.Perfil, "nome": u.Nome})
}

func novoID(prefixo string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefixo + "-" + hex.EncodeToString(b)
}

func (s *Servidor) postOrdem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClienteID                string `json:"clienteId"`
		VeiculoID                string `json:"veiculoId"`
		ValorDoOrcamentoCentavos int64  `json:"valorDoOrcamentoCentavos"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	id := novoID("os")
	if err := s.ordens.Abrir(r.Context(), id, req.ClienteID, req.VeiculoID, req.ValorDoOrcamentoCentavos); err != nil {
		responderErro(w, err)
		return
	}
	s.devolverOrdem(w, r, id, http.StatusCreated)
}

func (s *Servidor) devolverOrdem(w http.ResponseWriter, r *http.Request, id string, status int) {
	o, err := s.ordens.Consultar(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, status, paraView(o))
}

func (s *Servidor) getOrdens(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	var lista []*ordemservico.OrdemDeServico
	var err error
	if identidade.Perfil(u.Perfil) == identidade.Cliente {
		lista, err = s.ordens.ListarDoCliente(r.Context(), u.ClienteID)
	} else {
		lista, err = s.ordens.Listar(r.Context())
	}
	if err != nil {
		responderErro(w, err)
		return
	}
	out := make([]ordemView, 0, len(lista))
	for _, o := range lista {
		out = append(out, paraView(o))
	}
	escrever(w, http.StatusOK, out)
}

// getOrdem: o cliente só enxerga a própria OS (RN10); funcionário enxerga qualquer uma.
func (s *Servidor) getOrdem(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	id := r.PathValue("id")
	var o *ordemservico.OrdemDeServico
	var err error
	if identidade.Perfil(u.Perfil) == identidade.Cliente {
		o, err = s.ordens.ConsultarDoCliente(r.Context(), id, u.ClienteID)
	} else {
		o, err = s.ordens.Consultar(r.Context(), id)
	}
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraView(o))
}

// acao cobre as transições sem corpo (diagnóstico, orçamento, finalização, entrega).
func (s *Servidor) acao(f func(ctxCtx, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := f(r.Context(), id); err != nil {
			responderErro(w, err)
			return
		}
		s.devolverOrdem(w, r, id, http.StatusOK)
	}
}

func (s *Servidor) decisao(aprovar bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, cliente := r.PathValue("id"), usuarioDe(r).ClienteID
		var err error
		if aprovar {
			err = s.ordens.AprovarOrcamento(r.Context(), id, cliente)
		} else {
			err = s.ordens.ReprovarOrcamento(r.Context(), id, cliente)
		}
		if err != nil {
			responderErro(w, err)
			return
		}
		s.devolverOrdem(w, r, id, http.StatusOK)
	}
}

func (s *Servidor) postServico(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Descricao     string `json:"descricao"`
		PrecoCentavos int64  `json:"precoCentavos"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	if err := s.ordens.AdicionarServico(r.Context(), id, req.Descricao, req.PrecoCentavos); err != nil {
		responderErro(w, err)
		return
	}
	s.devolverOrdem(w, r, id, http.StatusOK)
}

func (s *Servidor) postPeca(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PecaID                string `json:"pecaId"`
		Descricao             string `json:"descricao"`
		PrecoUnitarioCentavos int64  `json:"precoUnitarioCentavos"`
		Quantidade            int    `json:"quantidade"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	if err := s.ordens.AdicionarPeca(r.Context(), id, req.PecaID, req.Descricao, req.PrecoUnitarioCentavos, req.Quantidade); err != nil {
		responderErro(w, err)
		return
	}
	s.devolverOrdem(w, r, id, http.StatusOK)
}

func (s *Servidor) postDesconto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PontosBase int64  `json:"pontosBase"`
		Base       string `json:"base"`
		Senha      string `json:"senha"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	base := ordemservico.BaseDesconto(req.Base)
	switch req.Base {
	case "total":
		base = ordemservico.BaseTotal
	case "mao_de_obra":
		base = ordemservico.BaseMaoDeObra
	}
	id := r.PathValue("id")
	if err := s.ordens.AplicarDesconto(r.Context(), id, usuarioDe(r).Sub, req.Senha, req.PontosBase, base); err != nil {
		responderErro(w, err)
		return
	}
	s.devolverOrdem(w, r, id, http.StatusOK)
}

func (s *Servidor) getTempoMedio(w http.ResponseWriter, r *http.Request) {
	d, n, err := s.ordens.TempoMedioDeExecucao(r.Context())
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, map[string]any{"osConsideradas": n, "tempoMedioSegundos": int64(d.Seconds()), "tempoMedio": d.String()})
}

type itemView struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Tipo  string `json:"tipo"`
	Saldo int    `json:"saldo"`
}

func paraItem(i *estoque.Item) itemView {
	return itemView{ID: i.ID(), Nome: i.Nome(), Tipo: string(i.Tipo()), Saldo: i.Saldo().Valor()}
}

func (s *Servidor) postItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		Nome  string `json:"nome"`
		Tipo  string `json:"tipo"`
		Saldo int    `json:"saldo"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	tipo := estoque.Peca
	switch req.Tipo {
	case "", "peca":
	case "insumo":
		tipo = estoque.Insumo
	default:
		escreverErro(w, http.StatusUnprocessableEntity, "dados_invalidos", `tipo deve ser "peca" ou "insumo"`)
		return
	}
	item, err := s.estoque.Cadastrar(r.Context(), req.ID, req.Nome, tipo, req.Saldo)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusCreated, paraItem(item))
}

func (s *Servidor) getItem(w http.ResponseWriter, r *http.Request) {
	item, err := s.estoque.Consultar(r.Context(), r.PathValue("id"))
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraItem(item))
}

func (s *Servidor) postReposicao(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Quantidade int `json:"quantidade"`
	}
	if !lerJSON(w, r, &req) {
		return
	}
	item, err := s.estoque.Repor(r.Context(), r.PathValue("id"), req.Quantidade)
	if err != nil {
		responderErro(w, err)
		return
	}
	escrever(w, http.StatusOK, paraItem(item))
}

func paginaSwagger(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>Oficina Mecânica - API</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css"></head>
<body><div id="swagger"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script>window.ui = SwaggerUIBundle({url: "/openapi.json", dom_id: "#swagger"});</script></body></html>`)
}
