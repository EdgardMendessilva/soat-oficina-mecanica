package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// corpoErro é o formato único de erro da API.
type corpoErro struct {
	Erro detalheErro `json:"erro"`
}

type detalheErro struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

func escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func escreverErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	escrever(w, status, corpoErro{detalheErro{Codigo: codigo, Mensagem: mensagem}})
}

// classes de erro do domínio → status HTTP. Erro desconhecido vira 500 sem vazar detalhe.
var (
	naoEncontrados = []error{application.ErrOrdemNaoEncontrada, application.ErrPecaNaoEncontrada, application.ErrItemNaoEncontrado}
	proibidos      = []error{application.ErrAcessoNegado, ordemservico.ErrSomenteDonoDecide, application.ErrSenhaDoGerenteInvalida}
	conflitos      = []error{
		ordemservico.ErrTransicaoInvalida, ordemservico.ErrItensSoNoDiagnostico, ordemservico.ErrOrcamentoVazio,
		ordemservico.ErrDecisaoNaoRegistrada, ordemservico.ErrDescontoSemOrcamentoFechado, ordemservico.ErrDescontoJaAplicado,
		application.ErrEstoqueInsuficiente, estoque.ErrSaldoInsuficiente, application.ErrItemJaExiste,
	}
	invalidos = []error{
		ordemservico.ErrDescricaoVazia, ordemservico.ErrQuantidadeInvalida, ordemservico.ErrIDObrigatorio,
		ordemservico.ErrDescontoAcimaDoTeto, ordemservico.ErrBaseDeDescontoInvalida, ordemservico.ErrDescontoSemAutorizacao,
		estoque.ErrItemInvalido, estoque.ErrQuantidadeZero, identidade.ErrSenhaFraca,
		valueobject.ErrValorNegativo, valueobject.ErrQuantidadeNegativa, valueobject.ErrPercentualInvalido,
		valueobject.ErrDocumentoInvalido, valueobject.ErrPlacaInvalida,
	}
)

func algum(err error, lista []error) bool {
	for _, e := range lista {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// responderErro traduz o erro do caso de uso para status + corpo padronizado.
func responderErro(w http.ResponseWriter, err error) {
	switch {
	case algum(err, naoEncontrados):
		escreverErro(w, http.StatusNotFound, "nao_encontrado", err.Error())
	case algum(err, proibidos):
		escreverErro(w, http.StatusForbidden, "acesso_negado", err.Error())
	case algum(err, conflitos):
		escreverErro(w, http.StatusConflict, "regra_de_negocio", err.Error())
	case algum(err, invalidos):
		escreverErro(w, http.StatusUnprocessableEntity, "dados_invalidos", err.Error())
	default:
		log.Printf("erro interno: %v", err)
		escreverErro(w, http.StatusInternalServerError, "erro_interno", "erro interno do servidor")
	}
}
