package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
	apihttp "github.com/EdgardMendessilva/soat-oficina-mecanica/internal/interfaces/http"
)

type relogio struct{}

func (relogio) Agora() time.Time { return time.Now() }

const (
	cpfGerente   = "529.982.247-25"
	cpfAtendente = "390.533.447-05"
	cpfMecanico  = "168.995.350-09"
	cpfCliente   = "935.411.347-80"
	cpfOutro     = "862.883.667-57"
)

func novoServidor(t *testing.T) http.Handler {
	t.Helper()
	jwt, _ := seguranca.NovoEmissorJWT("segredo-de-teste-123456", time.Hour, nil)
	ordens, itens, tx, usuarios := memoria.NovasOrdens(), memoria.NovoEstoque(), &memoria.Transacao{}, memoria.NovosUsuarios()
	clientes, veiculos, servicos := memoria.NovosClientes(), memoria.NovosVeiculos(), memoria.NovosServicos()
	if err := memoria.Semear(usuarios, "", "", true); err != nil {
		t.Fatal(err)
	}
	if err := memoria.SemearCadastro(context.Background(), clientes, veiculos); err != nil {
		t.Fatal(err)
	}
	cad := application.NovoCadastroService(clientes, veiculos, tx)
	return apihttp.NovoServidor(
		application.NovoOrdemDeServicoService(ordens, itens, tx, usuarios, relogio{}).ComCadastro(cad),
		application.NovoEstoqueService(itens, tx),
		application.NovoAutenticacaoService(usuarios, seguranca.Verificador{}),
		cad,
		application.NovoCatalogoService(servicos, tx),
		jwt,
	).Rotas()
}

type cliente struct {
	t *testing.T
	h http.Handler
}

func (c cliente) chamar(metodo, caminho, token string, corpo any) (int, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	if corpo != nil {
		_ = json.NewEncoder(&buf).Encode(corpo)
	}
	req := httptest.NewRequest(metodo, caminho, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	var saida map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &saida)
	return rec.Code, saida
}

func (c cliente) login(doc string) string {
	c.t.Helper()
	code, r := c.chamar("POST", "/auth/login", "", map[string]string{"documento": doc, "senha": memoria.SenhaDemo})
	if code != 200 {
		c.t.Fatalf("login de %s falhou: %d %v", doc, code, r)
	}
	return r["token"].(string)
}

func (c cliente) espera(esperado int, metodo, caminho, token string, corpo any) map[string]any {
	c.t.Helper()
	code, r := c.chamar(metodo, caminho, token, corpo)
	if code != esperado {
		c.t.Fatalf("%s %s: esperava %d, veio %d: %v", metodo, caminho, esperado, code, r)
	}
	return r
}

func TestFluxoCompletoDaOS(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ger, ate, mec, cli := c.login(cpfGerente), c.login(cpfAtendente), c.login(cpfMecanico), c.login(cpfCliente)

	c.espera(201, "POST", "/estoque", ger, map[string]any{"id": "filtro", "nome": "Filtro de óleo", "saldo": 5})

	os := c.espera(201, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": "veic-demo", "valorDoOrcamentoCentavos": 25000})
	id := os["id"].(string)
	base := "/ordens/" + id

	c.espera(200, "POST", base+"/diagnostico", mec, nil)
	c.espera(200, "POST", base+"/servicos", mec, map[string]any{"descricao": "Troca de óleo", "precoCentavos": 10000})
	c.espera(200, "POST", base+"/pecas", mec, map[string]any{"pecaId": "filtro", "descricao": "Filtro", "precoUnitarioCentavos": 5000, "quantidade": 2})
	r := c.espera(200, "POST", base+"/orcamento", ate, nil)
	if r["status"] != "Aguardando aprovação" || r["totalCentavos"].(float64) != 20000 {
		t.Fatalf("orçamento errado: %v", r)
	}

	// só o cliente dono decide (RN06, RN15)
	c.espera(403, "POST", base+"/aprovacao", ger, nil)
	c.espera(403, "POST", base+"/aprovacao", c.login(cpfOutro), nil)
	r = c.espera(200, "POST", base+"/aprovacao", cli, nil)
	if r["status"] != "Em execução" {
		t.Fatalf("aprovação não levou a Em execução: %v", r)
	}
	// o estoque foi baixado (RN09)
	if item := c.espera(200, "GET", "/estoque/filtro", ger, nil); item["saldo"].(float64) != 3 {
		t.Fatalf("saldo deveria ser 3: %v", item)
	}

	// desconto: senha errada recusada, teto de 10%, depois certo (RN07)
	c.espera(403, "POST", base+"/desconto", ger, map[string]any{"pontosBase": 500, "base": "total", "senha": "Errada@123"})
	c.espera(422, "POST", base+"/desconto", ger, map[string]any{"pontosBase": 1500, "base": "total", "senha": memoria.SenhaDemo})
	c.espera(403, "POST", base+"/desconto", ate, map[string]any{"pontosBase": 500, "base": "total", "senha": memoria.SenhaDemo})
	r = c.espera(200, "POST", base+"/desconto", ger, map[string]any{"pontosBase": 1000, "base": "total", "senha": memoria.SenhaDemo})
	if r["valorAPagarCentavos"].(float64) != 18000 {
		t.Fatalf("20000 com 10%% deveria dar 18000: %v", r)
	}

	c.espera(200, "POST", base+"/finalizacao", mec, nil)
	r = c.espera(200, "POST", base+"/entrega", ate, nil)
	if r["status"] != "Entregue" {
		t.Fatalf("status final errado: %v", r)
	}
	c.espera(200, "GET", "/relatorios/tempo-medio-execucao", ger, nil)
}

func TestReprovacaoCobraSoOValorDoOrcamento(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ate, mec, cli := c.login(cpfAtendente), c.login(cpfMecanico), c.login(cpfCliente)
	id := c.espera(201, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": "veic-demo", "valorDoOrcamentoCentavos": 25000})["id"].(string)
	base := "/ordens/" + id
	c.espera(200, "POST", base+"/diagnostico", mec, nil)
	c.espera(200, "POST", base+"/servicos", mec, map[string]any{"descricao": "Revisão", "precoCentavos": 90000})
	c.espera(200, "POST", base+"/orcamento", ate, nil)
	r := c.espera(200, "POST", base+"/reprovacao", cli, nil)
	if r["status"] != "Finalizada" || r["valorAPagarCentavos"].(float64) != 25000 {
		t.Fatalf("reprovação errada: %v", r)
	}
}

func TestClienteSoEnxergaAsPropriasOS(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ate := c.login(cpfAtendente)
	id := c.espera(201, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": "veic-demo", "valorDoOrcamentoCentavos": 1000})["id"].(string)

	c.espera(200, "GET", "/ordens/"+id, c.login(cpfCliente), nil)
	c.espera(403, "GET", "/ordens/"+id, c.login(cpfOutro), nil) // RN10
	code, _ := c.chamar("GET", "/ordens", c.login(cpfOutro), nil)
	if code != 200 {
		t.Fatalf("listagem: %d", code)
	}
}

func TestSegurancaDasRotas(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	c.espera(401, "GET", "/ordens", "", nil)
	c.espera(401, "GET", "/ordens", "token-falso", nil)
	c.espera(401, "POST", "/auth/login", "", map[string]string{"documento": cpfGerente, "senha": "Errada@123"})
	c.espera(401, "POST", "/auth/login", "", map[string]string{"documento": "123", "senha": "x"})
	c.espera(400, "POST", "/auth/login", "", map[string]any{"campoInexistente": 1})
	// cliente não cria OS nem mexe em estoque
	cli := c.login(cpfCliente)
	c.espera(403, "POST", "/ordens", cli, map[string]any{"clienteId": "x", "veiculoId": "y"})
	c.espera(403, "POST", "/estoque", cli, map[string]any{"id": "a", "nome": "b"})
	c.espera(404, "GET", "/ordens/nao-existe", c.login(cpfGerente), nil)
}

func TestDocumentacaoEHealth(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	c.espera(200, "GET", "/health", "", nil)
	doc := c.espera(200, "GET", "/openapi.json", "", nil)
	if doc["openapi"] == nil {
		t.Fatal("openapi.json inválido")
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, httptest.NewRequest("GET", "/swagger", nil))
	if rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte("swagger-ui")) {
		t.Fatal("página do Swagger não carregou")
	}
}

func TestEstoqueNaoFicaNegativoNaAprovacao(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ger, ate, mec, cli := c.login(cpfGerente), c.login(cpfAtendente), c.login(cpfMecanico), c.login(cpfCliente)
	c.espera(201, "POST", "/estoque", ger, map[string]any{"id": "vela", "nome": "Vela", "saldo": 1})
	c.espera(409, "POST", "/estoque", ger, map[string]any{"id": "vela", "nome": "Vela", "saldo": 1})
	id := c.espera(201, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": "veic-demo", "valorDoOrcamentoCentavos": 100})["id"].(string)
	base := "/ordens/" + id
	c.espera(200, "POST", base+"/diagnostico", mec, nil)
	c.espera(200, "POST", base+"/pecas", mec, map[string]any{"pecaId": "vela", "descricao": "Vela", "precoUnitarioCentavos": 100, "quantidade": 4})
	c.espera(200, "POST", base+"/orcamento", ate, nil)
	c.espera(409, "POST", base+"/aprovacao", cli, nil)
	if i := c.espera(200, "GET", "/estoque/vela", ger, nil); i["saldo"].(float64) != 1 {
		t.Fatalf("saldo mudou numa aprovação recusada: %v", i)
	}
}

func TestCadastroDeClienteEVeiculo(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ate, ger, cli := c.login(cpfAtendente), c.login(cpfGerente), c.login(cpfCliente)

	// CPF e CNPJ aceitos, inválido recusado, repetido recusado (RN01)
	novo := c.espera(201, "POST", "/clientes", ate, map[string]any{"documento": "111.444.777-35", "nome": "João", "email": "joao@mail.com"})
	c.espera(201, "POST", "/clientes", ate, map[string]any{"documento": "11.222.333/0001-81", "nome": "Oficina Y", "email": "y@mail.com"})
	c.espera(422, "POST", "/clientes", ate, map[string]any{"documento": "111.111.111-11", "nome": "X", "email": "x@mail.com"})
	c.espera(409, "POST", "/clientes", ate, map[string]any{"documento": "111.444.777-35", "nome": "João", "email": "joao@mail.com"})
	c.espera(403, "POST", "/clientes", cli, map[string]any{"documento": "529.982.247-25", "nome": "X", "email": "x@mail.com"})

	// veículo: placa válida e única (RN02), cliente precisa existir
	id := novo["id"].(string)
	v := c.espera(201, "POST", "/veiculos", ate, map[string]any{"clienteId": id, "placa": "def-4e56", "fabricante": "Fiat", "modelo": "Uno", "ano": 2018, "cor": "Preto"})
	if v["placa"] != "DEF4E56" {
		t.Fatalf("placa não normalizada: %v", v)
	}
	c.espera(409, "POST", "/veiculos", ate, map[string]any{"clienteId": id, "placa": "DEF4E56", "fabricante": "Fiat", "modelo": "Uno", "ano": 2018})
	c.espera(422, "POST", "/veiculos", ate, map[string]any{"clienteId": id, "placa": "ZZZ", "fabricante": "Fiat", "modelo": "Uno", "ano": 2018})
	c.espera(404, "POST", "/veiculos", ate, map[string]any{"clienteId": "nao-existe", "placa": "GHI5J67", "fabricante": "Fiat", "modelo": "Uno", "ano": 2018})

	// a OS só abre com veículo do próprio cliente
	c.espera(422, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": v["id"], "valorDoOrcamentoCentavos": 1000})
	c.espera(404, "POST", "/ordens", ate, map[string]any{"clienteId": "cli-demo", "veiculoId": "nao-existe", "valorDoOrcamentoCentavos": 1000})
	c.espera(201, "POST", "/ordens", ate, map[string]any{"clienteId": id, "veiculoId": v["id"], "valorDoOrcamentoCentavos": 1000})

	// isolamento do cliente (RN10)
	c.espera(200, "GET", "/clientes/cli-demo", cli, nil)
	c.espera(403, "GET", "/clientes/cli-outro", cli, nil)
	c.espera(200, "GET", "/veiculos/veic-demo", cli, nil)
	c.espera(403, "GET", "/veiculos/veic-outro", cli, nil)
	c.espera(200, "GET", "/clientes", ger, nil)
	c.espera(403, "GET", "/clientes", cli, nil)
	c.espera(400, "GET", "/veiculos", ger, nil)
	code, _ := c.chamar("GET", "/veiculos?clienteId=cli-demo", ger, nil)
	if code != 200 {
		t.Fatalf("listar veículos: %d", code)
	}
}

func TestCatalogoDeServicos(t *testing.T) {
	c := cliente{t, novoServidor(t)}
	ger, mec, ate := c.login(cpfGerente), c.login(cpfMecanico), c.login(cpfAtendente)

	s := c.espera(201, "POST", "/servicos", ger, map[string]any{"nome": "Alinhamento", "precoCentavos": 12000})
	id := s["id"].(string)
	c.espera(403, "POST", "/servicos", ate, map[string]any{"nome": "X", "precoCentavos": 1})
	c.espera(422, "POST", "/servicos", ger, map[string]any{"nome": " ", "precoCentavos": 1})
	c.espera(422, "POST", "/servicos", ger, map[string]any{"nome": "Negativo", "precoCentavos": -5})

	c.espera(200, "GET", "/servicos", mec, nil)
	if got := c.espera(200, "GET", "/servicos/"+id, mec, nil); got["nome"] != "Alinhamento" {
		t.Fatalf("consulta errada: %v", got)
	}
	if got := c.espera(200, "PUT", "/servicos/"+id, ger, map[string]any{"nome": "Alinhamento 3D", "precoCentavos": 15000}); got["precoCentavos"].(float64) != 15000 {
		t.Fatalf("alteração errada: %v", got)
	}
	c.espera(404, "PUT", "/servicos/nao-existe", ger, map[string]any{"nome": "X", "precoCentavos": 1})
	c.espera(403, "DELETE", "/servicos/"+id, mec, nil)
	c.espera(204, "DELETE", "/servicos/"+id, ger, nil)
	c.espera(404, "GET", "/servicos/"+id, ger, nil)
	c.espera(404, "DELETE", "/servicos/"+id, ger, nil)
}
