package application

import (
	"context"
	"errors"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
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
