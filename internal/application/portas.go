package application

import (
	"context"
	"errors"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/catalogo"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
)

// ErrNaoEncontrado é devolvido pelos repositórios quando o registro não existe.
var ErrNaoEncontrado = errors.New("registro não encontrado")

// OrdemRepositorio é a porta de persistência das OS, implementada na infraestrutura.
type OrdemRepositorio interface {
	BuscarPorID(ctx context.Context, id string) (*ordemservico.OrdemDeServico, error)
	Salvar(ctx context.Context, os *ordemservico.OrdemDeServico) error
	ListarTodas(ctx context.Context) ([]*ordemservico.OrdemDeServico, error)
}

// EstoqueRepositorio é a porta de persistência dos itens de estoque.
type EstoqueRepositorio interface {
	BuscarPorID(ctx context.Context, id string) (*estoque.Item, error)
	Salvar(ctx context.Context, item *estoque.Item) error
}

// Transacao executa a função numa única transação: se ela devolver erro, nada é gravado.
// Na infraestrutura é uma transação PostgreSQL, com SELECT ... FOR UPDATE no saldo.
type Transacao interface {
	Executar(ctx context.Context, fn func(ctx context.Context) error) error
}

// AutorizadorGerente confere a senha de login do gerente que autoriza um desconto (RN07).
type AutorizadorGerente interface {
	SenhaConfere(ctx context.Context, gerenteID, senha string) (bool, error)
}

// Relogio entrega a hora atual; existe para os testes controlarem o tempo.
type Relogio interface {
	Agora() time.Time
}

// UsuarioRepositorio busca quem tenta entrar, pelo CPF já normalizado.
type UsuarioRepositorio interface {
	BuscarPorDocumento(ctx context.Context, documento string) (identidade.Usuario, error)
}

// VerificadorSenha compara a senha digitada com o hash guardado.
type VerificadorSenha interface {
	Conferir(senha, hash string) bool
}

// ClienteRepositorio é a porta de persistência dos clientes.
type ClienteRepositorio interface {
	BuscarPorID(ctx context.Context, id string) (*cadastro.Cliente, error)
	BuscarPorDocumento(ctx context.Context, documento string) (*cadastro.Cliente, error)
	Salvar(ctx context.Context, c *cadastro.Cliente) error
	ListarTodos(ctx context.Context) ([]*cadastro.Cliente, error)
}

// VeiculoRepositorio é a porta de persistência dos veículos.
type VeiculoRepositorio interface {
	BuscarPorID(ctx context.Context, id string) (*cadastro.Veiculo, error)
	BuscarPorPlaca(ctx context.Context, placa string) (*cadastro.Veiculo, error)
	Salvar(ctx context.Context, v *cadastro.Veiculo) error
	ListarDoCliente(ctx context.Context, clienteID string) ([]*cadastro.Veiculo, error)
}

// ServicoRepositorio é a porta de persistência do catálogo de serviços.
type ServicoRepositorio interface {
	BuscarPorID(ctx context.Context, id string) (*catalogo.Servico, error)
	Salvar(ctx context.Context, s *catalogo.Servico) error
	Listar(ctx context.Context) ([]*catalogo.Servico, error)
	Remover(ctx context.Context, id string) error
}

// VerificadorDeVeiculo confere se o veículo existe e pertence ao cliente, ao abrir a OS.
type VerificadorDeVeiculo interface {
	VeiculoDoCliente(ctx context.Context, clienteID, veiculoID string) error
}
