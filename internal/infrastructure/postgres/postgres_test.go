package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/postgres"
)

// Os testes de integração usam um PostgreSQL real, apontado por TEST_DATABASE_URL
// (por exemplo o do docker compose). Sem a variável, são pulados. Eles apagam as
// tabelas de dados do banco apontado: use um banco de testes.

type relogio struct{ t time.Time }

func (r *relogio) Agora() time.Time { r.t = r.t.Add(time.Minute); return r.t }

type ambiente struct {
	db       *sql.DB
	ordens   *application.OrdemDeServicoService
	estoque  *application.EstoqueService
	repoOS   *postgres.Ordens
	repoItem *postgres.Estoque
}

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida; teste de integração pulado")
	}
	ctx := context.Background()
	db, err := postgres.Abrir(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `TRUNCATE os_historico, os_pecas, os_servicos, ordens_servico, itens_estoque, usuarios, veiculos, clientes, servicos_catalogo`); err != nil {
		t.Fatal(err)
	}

	usuarios := postgres.NovosUsuarios(db)
	if err := postgres.Semear(ctx, usuarios, "", "", true); err != nil {
		t.Fatal(err)
	}
	repoOS, repoItem, tx := postgres.NovasOrdens(db), postgres.NovoEstoque(db), postgres.NovaTransacao(db)
	return &ambiente{
		db:       db,
		ordens:   application.NovoOrdemDeServicoService(repoOS, repoItem, tx, usuarios, &relogio{t: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}),
		estoque:  application.NovoEstoqueService(repoItem, tx),
		repoOS:   repoOS,
		repoItem: repoItem,
	}
}

func ok(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func saldo(t *testing.T, a *ambiente, id string) int {
	t.Helper()
	i, err := a.repoItem.BuscarPorID(context.Background(), id)
	ok(t, err)
	return i.Saldo().Valor()
}

func TestFluxoCompletoDaOS(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()

	_, err := a.estoque.Cadastrar(ctx, "filtro", "Filtro de óleo", estoque.Peca, 10)
	ok(t, err)

	ok(t, a.ordens.Abrir(ctx, "os-1", "cli-demo", "veic-1", 5000))
	ok(t, a.ordens.IniciarDiagnostico(ctx, "os-1"))
	ok(t, a.ordens.AdicionarServico(ctx, "os-1", "Troca de óleo", 15000))
	ok(t, a.ordens.AdicionarPeca(ctx, "os-1", "filtro", "Filtro de óleo", 4000, 2))
	ok(t, a.ordens.EnviarOrcamento(ctx, "os-1"))
	ok(t, a.ordens.AprovarOrcamento(ctx, "os-1", "cli-demo"))

	if got := saldo(t, a, "filtro"); got != 8 {
		t.Fatalf("saldo após aprovação = %d, esperava 8", got)
	}

	ok(t, a.ordens.AplicarDesconto(ctx, "os-1", "ger-demo", memoria.SenhaDemo, 1000, ordemservico.BaseTotal))
	ok(t, a.ordens.Finalizar(ctx, "os-1"))
	ok(t, a.ordens.Entregar(ctx, "os-1"))

	o, err := a.ordens.Consultar(ctx, "os-1")
	ok(t, err)
	if o.Status() != ordemservico.Entregue || o.Decisao() != ordemservico.Aprovado || o.DecididoPor() != "cli-demo" {
		t.Fatalf("estado final inesperado: %s / %s / %q", o.Status(), o.Decisao(), o.DecididoPor())
	}
	if len(o.Servicos()) != 1 || len(o.Pecas()) != 1 || o.Pecas()[0].Quantidade.Valor() != 2 {
		t.Fatalf("itens não persistidos: %+v %+v", o.Servicos(), o.Pecas())
	}
	// 15000 + 2×4000 = 23000; desconto de 10% = 2300; a pagar 20700
	pagar, err := o.ValorAPagar()
	ok(t, err)
	if pagar.Centavos() != 20700 {
		t.Fatalf("valor a pagar = %d, esperava 20700", pagar.Centavos())
	}
	if d := o.Desconto(); d == nil || d.AutorizadoPor != "ger-demo" || d.PontosBase != 1000 {
		t.Fatalf("desconto não persistido: %+v", d)
	}
	if h := o.Historico(); len(h) != 6 || h[0].Para != ordemservico.Recebida || h[5].Para != ordemservico.Entregue {
		t.Fatalf("histórico inesperado: %+v", h)
	}
	if _, ok := o.TempoDeExecucao(); !ok {
		t.Fatal("tempo de execução deveria ser calculável a partir do histórico")
	}

	lista, err := a.ordens.Listar(ctx)
	ok(t, err)
	if len(lista) != 1 || lista[0].ID() != "os-1" {
		t.Fatalf("listagem inesperada: %d OS", len(lista))
	}
	if _, err := a.ordens.Consultar(ctx, "nao-existe"); !errors.Is(err, application.ErrOrdemNaoEncontrada) {
		t.Fatalf("esperava ErrOrdemNaoEncontrada, veio %v", err)
	}
}

func TestAprovacaoSemEstoqueSuficienteNaoAlteraNada(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()

	_, err := a.estoque.Cadastrar(ctx, "filtro", "Filtro de óleo", estoque.Peca, 5)
	ok(t, err)
	_, err = a.estoque.Cadastrar(ctx, "vela", "Vela", estoque.Peca, 1)
	ok(t, err)

	ok(t, a.ordens.Abrir(ctx, "os-2", "cli-demo", "veic-1", 5000))
	ok(t, a.ordens.IniciarDiagnostico(ctx, "os-2"))
	ok(t, a.ordens.AdicionarPeca(ctx, "os-2", "filtro", "Filtro", 4000, 3))
	ok(t, a.ordens.AdicionarPeca(ctx, "os-2", "vela", "Vela", 2000, 4)) // só há 1 em estoque
	ok(t, a.ordens.EnviarOrcamento(ctx, "os-2"))

	err = a.ordens.AprovarOrcamento(ctx, "os-2", "cli-demo")
	if !errors.Is(err, application.ErrEstoqueInsuficiente) {
		t.Fatalf("esperava ErrEstoqueInsuficiente, veio %v", err)
	}

	if got := saldo(t, a, "filtro"); got != 5 {
		t.Errorf("saldo do filtro = %d; a baixa do primeiro item deveria ter sido desfeita (5)", got)
	}
	if got := saldo(t, a, "vela"); got != 1 {
		t.Errorf("saldo da vela = %d, esperava 1", got)
	}
	o, err := a.ordens.Consultar(ctx, "os-2")
	ok(t, err)
	if o.Status() != ordemservico.AguardandoAprovacao || o.Decisao() != ordemservico.Pendente {
		t.Errorf("a OS mudou: %s / %s", o.Status(), o.Decisao())
	}
}

func TestBaixaConcorrenteNaoDeixaSaldoNegativo(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()

	_, err := a.estoque.Cadastrar(ctx, "filtro", "Filtro de óleo", estoque.Peca, 3)
	ok(t, err)
	for _, id := range []string{"os-a", "os-b", "os-c"} {
		ok(t, a.ordens.Abrir(ctx, id, "cli-demo", "veic-1", 5000))
		ok(t, a.ordens.IniciarDiagnostico(ctx, id))
		ok(t, a.ordens.AdicionarPeca(ctx, id, "filtro", "Filtro", 4000, 2))
		ok(t, a.ordens.EnviarOrcamento(ctx, id))
	}

	var wg sync.WaitGroup
	erros := make([]error, 3)
	for i, id := range []string{"os-a", "os-b", "os-c"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			erros[i] = a.ordens.AprovarOrcamento(ctx, id, "cli-demo")
		}()
	}
	wg.Wait()

	aprovadas := 0
	for _, e := range erros {
		switch {
		case e == nil:
			aprovadas++
		case !errors.Is(e, application.ErrEstoqueInsuficiente):
			t.Errorf("erro inesperado: %v", e)
		}
	}
	if aprovadas != 1 {
		t.Errorf("aprovadas = %d, esperava 1 (saldo 3, cada OS pede 2)", aprovadas)
	}
	if got := saldo(t, a, "filtro"); got != 1 {
		t.Errorf("saldo final = %d, esperava 1", got)
	}
}

func TestSeedIdempotenteEMigracoesRepetiveis(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()

	u := postgres.NovosUsuarios(a.db)
	ok(t, postgres.Semear(ctx, u, "", "", true)) // segunda vez: não duplica nem falha
	ok(t, postgres.Migrar(ctx, a.db))            // migrations já aplicadas são ignoradas

	var n int
	ok(t, a.db.QueryRowContext(ctx, `SELECT count(*) FROM usuarios`).Scan(&n))
	if n != 6 {
		t.Fatalf("usuários = %d, esperava 6", n)
	}
	ger, err := u.BuscarPorDocumento(ctx, "52998224725")
	ok(t, err)
	if conf, _ := u.SenhaConfere(ctx, ger.ID, memoria.SenhaDemo); !conf {
		t.Error("a senha do gerente demo deveria conferir")
	}
	if conf, _ := u.SenhaConfere(ctx, "cli-demo", memoria.SenhaDemo); conf {
		t.Error("cliente não pode autorizar desconto")
	}
}

func TestCadastroECatalogoNoPostgres(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()
	tx := postgres.NovaTransacao(a.db)
	cad := application.NovoCadastroService(postgres.NovosClientes(a.db), postgres.NovosVeiculos(a.db), tx)

	if _, err := cad.CadastrarCliente(ctx, "c1", "529.982.247-25", "Maria", "maria@mail.com", "119", "Rua A"); err != nil {
		t.Fatal(err)
	}
	if _, err := cad.CadastrarCliente(ctx, "c2", "52998224725", "Outra", "o@mail.com", "", ""); !errors.Is(err, application.ErrClienteJaExiste) {
		t.Fatalf("CPF repetido deveria ser recusado: %v", err)
	}
	if _, err := cad.CadastrarVeiculo(ctx, "v1", "c1", "ABC1D23", "Renault", "Kwid", 2023, "Branco"); err != nil {
		t.Fatal(err)
	}
	if _, err := cad.CadastrarVeiculo(ctx, "v2", "c1", "abc-1d23", "Fiat", "Uno", 2010, ""); !errors.Is(err, application.ErrPlacaJaCadastrada) {
		t.Fatalf("placa repetida deveria ser recusada: %v", err)
	}
	if err := cad.VeiculoDoCliente(ctx, "c1", "v1"); err != nil {
		t.Fatal(err)
	}
	if lista, err := cad.ListarVeiculosDoCliente(ctx, "c1"); err != nil || len(lista) != 1 {
		t.Fatalf("listar veículos: %v %d", err, len(lista))
	}
	if lista, err := cad.ListarClientes(ctx); err != nil || len(lista) != 1 {
		t.Fatalf("listar clientes: %v %d", err, len(lista))
	}

	cat := application.NovoCatalogoService(postgres.NovosServicos(a.db), tx)
	if _, err := cat.Criar(ctx, "s1", "Alinhamento", 12000); err != nil {
		t.Fatal(err)
	}
	if s, err := cat.Alterar(ctx, "s1", "Alinhamento 3D", 15000); err != nil || s.Preco().Centavos() != 15000 {
		t.Fatalf("alterar: %v", err)
	}
	if lista, _ := cat.Listar(ctx); len(lista) != 1 {
		t.Fatalf("esperava 1 serviço, veio %d", len(lista))
	}
	if err := cat.Remover(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := cat.Remover(ctx, "s1"); !errors.Is(err, application.ErrServicoNaoEncontrado) {
		t.Fatalf("remover de novo deveria dar não encontrado: %v", err)
	}
}
