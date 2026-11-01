package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

type relogioFixo struct{}

func (relogioFixo) Agora() time.Time { return time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC) }

func deveSer(t *testing.T, err, esperado error) {
	t.Helper()
	if !errors.Is(err, esperado) {
		t.Fatalf("esperava %v, veio %v", esperado, err)
	}
}

func semErro(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	usuarios := memoria.NovosUsuarios()
	semErro(t, memoria.Semear(usuarios, "", "", true))
	svc := application.NovoAutenticacaoService(usuarios, seguranca.Verificador{})

	u, err := svc.Login(ctx, "529.982.247-25", memoria.SenhaDemo)
	semErro(t, err)
	if u.Perfil != identidade.Gerente {
		t.Errorf("perfil errado: %v", u.Perfil)
	}
	_, err = svc.Login(ctx, "529.982.247-25", "Errada@123")
	deveSer(t, err, application.ErrCredenciaisInvalidas)
	_, err = svc.Login(ctx, "111.111.111-11", memoria.SenhaDemo)
	deveSer(t, err, application.ErrCredenciaisInvalidas)
	_, err = svc.Login(ctx, "935.411.347-80", memoria.SenhaDemo) // cliente sem cadastro no repositório vazio? existe como usuário
	semErro(t, err)
	_, err = svc.Login(ctx, "123.456.789-09", memoria.SenhaDemo) // CPF válido, sem usuário
	deveSer(t, err, application.ErrCredenciaisInvalidas)
}

func TestCadastroDeClientesEVeiculos(t *testing.T) {
	ctx := context.Background()
	tx := &memoria.Transacao{}
	svc := application.NovoCadastroService(memoria.NovosClientes(), memoria.NovosVeiculos(), tx)

	c, err := svc.CadastrarCliente(ctx, "c1", "529.982.247-25", "Maria", "maria@mail.com", "119", "Rua A")
	semErro(t, err)
	if c.ID() != "c1" || c.Telefone() != "119" || c.Endereco() != "Rua A" {
		t.Errorf("cliente errado: %+v", c)
	}
	_, err = svc.CadastrarCliente(ctx, "c2", "52998224725", "Outra", "o@mail.com", "", "")
	deveSer(t, err, application.ErrClienteJaExiste)
	_, err = svc.CadastrarCliente(ctx, "c3", "000", "X", "x@mail.com", "", "")
	if err == nil {
		t.Fatal("aceitou CPF inválido")
	}
	if _, err := svc.ConsultarCliente(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConsultarCliente(ctx, "nao")
	deveSer(t, err, application.ErrClienteNaoEncontrado)
	if lista, _ := svc.ListarClientes(ctx); len(lista) != 1 {
		t.Errorf("esperava 1 cliente, veio %d", len(lista))
	}

	v, err := svc.CadastrarVeiculo(ctx, "v1", "c1", "ABC1D23", "Renault", "Kwid", 2023, "Branco")
	semErro(t, err)
	if v.ID() != "v1" {
		t.Errorf("veículo errado: %+v", v)
	}
	_, err = svc.CadastrarVeiculo(ctx, "v2", "c1", "abc-1d23", "Fiat", "Uno", 2010, "")
	deveSer(t, err, application.ErrPlacaJaCadastrada)
	_, err = svc.CadastrarVeiculo(ctx, "v3", "nao", "DEF4567", "Fiat", "Uno", 2010, "")
	deveSer(t, err, application.ErrClienteNaoEncontrado)
	if _, err := svc.ConsultarVeiculo(ctx, "v1"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConsultarVeiculo(ctx, "nao")
	deveSer(t, err, application.ErrVeiculoNaoEncontrado)
	if lista, _ := svc.ListarVeiculosDoCliente(ctx, "c1"); len(lista) != 1 {
		t.Errorf("esperava 1 veículo, veio %d", len(lista))
	}

	semErro(t, svc.VeiculoDoCliente(ctx, "c1", "v1"))
	deveSer(t, svc.VeiculoDoCliente(ctx, "nao", "v1"), application.ErrClienteNaoEncontrado)
	deveSer(t, svc.VeiculoDoCliente(ctx, "c1", "nao"), application.ErrVeiculoNaoEncontrado)
	_, err = svc.CadastrarCliente(ctx, "c9", "111.444.777-35", "Joao", "j@mail.com", "", "")
	semErro(t, err)
	deveSer(t, svc.VeiculoDoCliente(ctx, "c9", "v1"), application.ErrVeiculoDeOutroCliente)
}

func TestCatalogoDeServicos(t *testing.T) {
	ctx := context.Background()
	svc := application.NovoCatalogoService(memoria.NovosServicos(), &memoria.Transacao{})

	s, err := svc.Criar(ctx, "s1", "Troca de óleo", 15000)
	semErro(t, err)
	if s.Preco().Centavos() != 15000 {
		t.Errorf("preço errado: %v", s.Preco())
	}
	_, err = svc.Criar(ctx, "s1", "Repetido", 100)
	deveSer(t, err, application.ErrServicoJaExiste)
	if _, err := svc.Criar(ctx, "s2", "Negativo", -1); err == nil {
		t.Fatal("aceitou preço negativo")
	}
	if _, err := svc.Criar(ctx, "s3", " ", 1); err == nil {
		t.Fatal("aceitou nome vazio")
	}
	got, err := svc.Alterar(ctx, "s1", "Troca de óleo e filtro", 18000)
	semErro(t, err)
	if got.Nome() != "Troca de óleo e filtro" || got.Preco().Centavos() != 18000 {
		t.Errorf("alteração errada: %+v", got)
	}
	if _, err := svc.Alterar(ctx, "s1", "x", -1); err == nil {
		t.Fatal("aceitou preço negativo na alteração")
	}
	if _, err := svc.Alterar(ctx, "s1", " ", 1); err == nil {
		t.Fatal("aceitou nome vazio na alteração")
	}
	_, err = svc.Alterar(ctx, "nao", "x", 1)
	deveSer(t, err, application.ErrServicoNaoEncontrado)
	if lista, _ := svc.Listar(ctx); len(lista) != 1 {
		t.Errorf("esperava 1 serviço, veio %d", len(lista))
	}
	semErro(t, svc.Remover(ctx, "s1"))
	deveSer(t, svc.Remover(ctx, "s1"), application.ErrServicoNaoEncontrado)
	_, err = svc.Consultar(ctx, "s1")
	deveSer(t, err, application.ErrServicoNaoEncontrado)
}

func TestEstoqueService(t *testing.T) {
	ctx := context.Background()
	svc := application.NovoEstoqueService(memoria.NovoEstoque(), &memoria.Transacao{})

	i, err := svc.Cadastrar(ctx, "oleo", "Óleo 5W30", estoque.Insumo, 4)
	semErro(t, err)
	if i.Saldo().Valor() != 4 || i.Tipo() != estoque.Insumo {
		t.Errorf("item errado: %+v", i)
	}
	_, err = svc.Cadastrar(ctx, "oleo", "Repetido", estoque.Insumo, 1)
	deveSer(t, err, application.ErrItemJaExiste)
	if _, err := svc.Cadastrar(ctx, "x", "Negativo", estoque.Peca, -1); err == nil {
		t.Fatal("aceitou saldo negativo")
	}
	if _, err := svc.Cadastrar(ctx, "", "Sem id", estoque.Peca, 1); err == nil {
		t.Fatal("aceitou item sem id")
	}
	i, err = svc.Repor(ctx, "oleo", 6)
	semErro(t, err)
	if i.Saldo().Valor() != 10 {
		t.Errorf("saldo deveria ser 10, veio %d", i.Saldo().Valor())
	}
	if _, err := svc.Repor(ctx, "oleo", 0); err == nil {
		t.Fatal("aceitou reposição zero")
	}
	if _, err := svc.Repor(ctx, "oleo", -1); err == nil {
		t.Fatal("aceitou reposição negativa")
	}
	_, err = svc.Repor(ctx, "nao", 1)
	deveSer(t, err, application.ErrItemNaoEncontrado)
	_, err = svc.Consultar(ctx, "nao")
	deveSer(t, err, application.ErrItemNaoEncontrado)
}

func TestOrdemComCadastroEConsultas(t *testing.T) {
	ctx := context.Background()
	tx := &memoria.Transacao{}
	clientes, veiculos := memoria.NovosClientes(), memoria.NovosVeiculos()
	semErro(t, memoria.SemearCadastro(ctx, clientes, veiculos))
	cad := application.NovoCadastroService(clientes, veiculos, tx)
	os := application.NovoOrdemDeServicoService(memoria.NovasOrdens(), memoria.NovoEstoque(), tx, memoria.NovosUsuarios(), relogioFixo{}).ComCadastro(cad)

	semErro(t, os.Abrir(ctx, "os1", "cli-demo", "veic-demo", 25000))
	deveSer(t, os.Abrir(ctx, "os2", "cli-demo", "veic-outro", 1), application.ErrVeiculoDeOutroCliente)
	deveSer(t, os.Abrir(ctx, "os3", "cli-demo", "nao", 1), application.ErrVeiculoNaoEncontrado)
	deveSer(t, os.Abrir(ctx, "os4", "nao", "veic-demo", 1), application.ErrClienteNaoEncontrado)
	semErro(t, os.Abrir(ctx, "os5", "cli-outro", "veic-outro", 1))

	if o, err := os.Consultar(ctx, "os1"); err != nil || o.ClienteID() != "cli-demo" {
		t.Fatalf("consulta falhou: %v", err)
	}
	if todas, _ := os.Listar(ctx); len(todas) != 2 {
		t.Errorf("esperava 2 OS, veio %d", len(todas))
	}
	if minhas, _ := os.ListarDoCliente(ctx, "cli-demo"); len(minhas) != 1 || minhas[0].ID() != "os1" {
		t.Errorf("o cliente deveria ver só a própria OS: %v", minhas)
	}
}
