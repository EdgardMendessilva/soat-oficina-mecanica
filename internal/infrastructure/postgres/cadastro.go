package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/catalogo"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Clientes implementa application.ClienteRepositorio.
type Clientes struct{ db *sql.DB }

// NovosClientes cria o repositório de clientes.
func NovosClientes(db *sql.DB) *Clientes { return &Clientes{db: db} }

const colunasCliente = `SELECT id, documento, nome, email, telefone, endereco FROM clientes`

type varredor interface{ Scan(dest ...any) error }

func lerCliente(l varredor) (*cadastro.Cliente, error) {
	var id, doc, nome, email, tel, end string
	if err := l.Scan(&id, &doc, &nome, &email, &tel, &end); err != nil {
		return nil, err
	}
	return cadastro.NovoCliente(id, doc, nome, email, tel, end)
}

func (r *Clientes) um(ctx context.Context, filtro, arg string) (*cadastro.Cliente, error) {
	c, err := lerCliente(executorDe(ctx, r.db).QueryRowContext(ctx, colunasCliente+` WHERE `+filtro+` = $1`, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNaoEncontrado
	}
	return c, err
}

// BuscarPorID devolve o cliente ou ErrNaoEncontrado.
func (r *Clientes) BuscarPorID(ctx context.Context, id string) (*cadastro.Cliente, error) {
	return r.um(ctx, "id", id)
}

// BuscarPorDocumento devolve o cliente do CPF ou CNPJ (só dígitos).
func (r *Clientes) BuscarPorDocumento(ctx context.Context, documento string) (*cadastro.Cliente, error) {
	return r.um(ctx, "documento", documento)
}

// Salvar grava ou atualiza o cliente.
func (r *Clientes) Salvar(ctx context.Context, c *cadastro.Cliente) error {
	_, err := executorDe(ctx, r.db).ExecContext(ctx,
		`INSERT INTO clientes (id, documento, nome, email, telefone, endereco) VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE SET documento = EXCLUDED.documento, nome = EXCLUDED.nome,
		   email = EXCLUDED.email, telefone = EXCLUDED.telefone, endereco = EXCLUDED.endereco`,
		c.ID(), c.Documento().Numero(), c.Nome(), c.Email(), c.Telefone(), c.Endereco())
	return err
}

// ListarTodos devolve os clientes por nome.
func (r *Clientes) ListarTodos(ctx context.Context) ([]*cadastro.Cliente, error) {
	linhas, err := executorDe(ctx, r.db).QueryContext(ctx, colunasCliente+` ORDER BY nome, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = linhas.Close() }()
	var out []*cadastro.Cliente
	for linhas.Next() {
		c, err := lerCliente(linhas)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, linhas.Err()
}

// Veiculos implementa application.VeiculoRepositorio.
type Veiculos struct{ db *sql.DB }

// NovosVeiculos cria o repositório de veículos.
func NovosVeiculos(db *sql.DB) *Veiculos { return &Veiculos{db: db} }

const colunasVeiculo = `SELECT id, cliente_id, placa, fabricante, modelo, ano, cor FROM veiculos`

func lerVeiculo(l varredor) (*cadastro.Veiculo, error) {
	var id, cli, placa, fab, mod, cor string
	var ano int
	if err := l.Scan(&id, &cli, &placa, &fab, &mod, &ano, &cor); err != nil {
		return nil, err
	}
	return cadastro.NovoVeiculo(id, cli, placa, fab, mod, ano, cor)
}

func (r *Veiculos) um(ctx context.Context, filtro, arg string) (*cadastro.Veiculo, error) {
	v, err := lerVeiculo(executorDe(ctx, r.db).QueryRowContext(ctx, colunasVeiculo+` WHERE `+filtro+` = $1`, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNaoEncontrado
	}
	return v, err
}

// BuscarPorID devolve o veículo ou ErrNaoEncontrado.
func (r *Veiculos) BuscarPorID(ctx context.Context, id string) (*cadastro.Veiculo, error) {
	return r.um(ctx, "id", id)
}

// BuscarPorPlaca devolve o veículo da placa normalizada.
func (r *Veiculos) BuscarPorPlaca(ctx context.Context, placa string) (*cadastro.Veiculo, error) {
	return r.um(ctx, "placa", placa)
}

// Salvar grava ou atualiza o veículo.
func (r *Veiculos) Salvar(ctx context.Context, v *cadastro.Veiculo) error {
	_, err := executorDe(ctx, r.db).ExecContext(ctx,
		`INSERT INTO veiculos (id, cliente_id, placa, fabricante, modelo, ano, cor) VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET cliente_id = EXCLUDED.cliente_id, placa = EXCLUDED.placa,
		   fabricante = EXCLUDED.fabricante, modelo = EXCLUDED.modelo, ano = EXCLUDED.ano, cor = EXCLUDED.cor`,
		v.ID(), v.ClienteID(), v.Placa().String(), v.Fabricante(), v.Modelo(), v.Ano(), v.Cor())
	return err
}

// ListarDoCliente devolve os veículos do cliente.
func (r *Veiculos) ListarDoCliente(ctx context.Context, clienteID string) ([]*cadastro.Veiculo, error) {
	linhas, err := executorDe(ctx, r.db).QueryContext(ctx, colunasVeiculo+` WHERE cliente_id = $1 ORDER BY placa`, clienteID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = linhas.Close() }()
	var out []*cadastro.Veiculo
	for linhas.Next() {
		v, err := lerVeiculo(linhas)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, linhas.Err()
}

// Servicos implementa application.ServicoRepositorio (catálogo).
type Servicos struct{ db *sql.DB }

// NovosServicos cria o repositório do catálogo.
func NovosServicos(db *sql.DB) *Servicos { return &Servicos{db: db} }

func lerServico(l varredor) (*catalogo.Servico, error) {
	var id, nome string
	var preco int64
	if err := l.Scan(&id, &nome, &preco); err != nil {
		return nil, err
	}
	p, err := valueobject.NovoDinheiro(preco)
	if err != nil {
		return nil, err
	}
	return catalogo.NovoServico(id, nome, p)
}

// BuscarPorID devolve o serviço ou ErrNaoEncontrado.
func (r *Servicos) BuscarPorID(ctx context.Context, id string) (*catalogo.Servico, error) {
	s, err := lerServico(executorDe(ctx, r.db).QueryRowContext(ctx, `SELECT id, nome, preco FROM servicos_catalogo WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNaoEncontrado
	}
	return s, err
}

// Salvar grava ou atualiza o serviço.
func (r *Servicos) Salvar(ctx context.Context, s *catalogo.Servico) error {
	_, err := executorDe(ctx, r.db).ExecContext(ctx,
		`INSERT INTO servicos_catalogo (id, nome, preco) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET nome = EXCLUDED.nome, preco = EXCLUDED.preco`,
		s.ID(), s.Nome(), s.Preco().Centavos())
	return err
}

// Listar devolve o catálogo por nome.
func (r *Servicos) Listar(ctx context.Context) ([]*catalogo.Servico, error) {
	linhas, err := executorDe(ctx, r.db).QueryContext(ctx, `SELECT id, nome, preco FROM servicos_catalogo ORDER BY nome, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = linhas.Close() }()
	var out []*catalogo.Servico
	for linhas.Next() {
		s, err := lerServico(linhas)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, linhas.Err()
}

// Remover apaga o serviço do catálogo.
func (r *Servicos) Remover(ctx context.Context, id string) error {
	res, err := executorDe(ctx, r.db).ExecContext(ctx, `DELETE FROM servicos_catalogo WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return application.ErrNaoEncontrado
	}
	return nil
}
