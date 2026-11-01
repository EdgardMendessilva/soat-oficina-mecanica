package application

import (
	"context"
	"errors"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
)

// Erros do cadastro.
var (
	ErrClienteJaExiste       = errors.New("já existe um cliente com esse CPF ou CNPJ")
	ErrClienteNaoEncontrado  = errors.New("cliente não encontrado")
	ErrPlacaJaCadastrada     = errors.New("já existe um veículo com essa placa")
	ErrVeiculoNaoEncontrado  = errors.New("veículo não encontrado")
	ErrVeiculoDeOutroCliente = errors.New("o veículo não pertence a esse cliente")
)

// CadastroService cadastra e consulta clientes e veículos (RN01, RN02).
type CadastroService struct {
	clientes ClienteRepositorio
	veiculos VeiculoRepositorio
	tx       Transacao
}

// NovoCadastroService monta o serviço.
func NovoCadastroService(c ClienteRepositorio, v VeiculoRepositorio, tx Transacao) *CadastroService {
	return &CadastroService{clientes: c, veiculos: v, tx: tx}
}

// CadastrarCliente valida o CPF ou CNPJ e recusa documento repetido.
func (s *CadastroService) CadastrarCliente(ctx context.Context, id, documento, nome, email, telefone, endereco string) (*cadastro.Cliente, error) {
	c, err := cadastro.NovoCliente(id, documento, nome, email, telefone, endereco)
	if err != nil {
		return nil, err
	}
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		if _, err := s.clientes.BuscarPorDocumento(ctx, c.Documento().Numero()); err == nil {
			return ErrClienteJaExiste
		} else if !errors.Is(err, ErrNaoEncontrado) {
			return err
		}
		return s.clientes.Salvar(ctx, c)
	})
	return c, err
}

// ConsultarCliente devolve o cliente.
func (s *CadastroService) ConsultarCliente(ctx context.Context, id string) (*cadastro.Cliente, error) {
	c, err := s.clientes.BuscarPorID(ctx, id)
	if errors.Is(err, ErrNaoEncontrado) {
		return nil, ErrClienteNaoEncontrado
	}
	return c, err
}

// ListarClientes devolve todos os clientes.
func (s *CadastroService) ListarClientes(ctx context.Context) ([]*cadastro.Cliente, error) {
	return s.clientes.ListarTodos(ctx)
}

// CadastrarVeiculo cadastra o veículo de um cliente existente e recusa placa repetida.
func (s *CadastroService) CadastrarVeiculo(ctx context.Context, id, clienteID, placa, fabricante, modelo string, ano int, cor string) (*cadastro.Veiculo, error) {
	v, err := cadastro.NovoVeiculo(id, clienteID, placa, fabricante, modelo, ano, cor)
	if err != nil {
		return nil, err
	}
	err = s.tx.Executar(ctx, func(ctx context.Context) error {
		if _, err := s.ConsultarCliente(ctx, clienteID); err != nil {
			return err
		}
		if _, err := s.veiculos.BuscarPorPlaca(ctx, v.Placa().String()); err == nil {
			return ErrPlacaJaCadastrada
		} else if !errors.Is(err, ErrNaoEncontrado) {
			return err
		}
		return s.veiculos.Salvar(ctx, v)
	})
	return v, err
}

// ConsultarVeiculo devolve o veículo.
func (s *CadastroService) ConsultarVeiculo(ctx context.Context, id string) (*cadastro.Veiculo, error) {
	v, err := s.veiculos.BuscarPorID(ctx, id)
	if errors.Is(err, ErrNaoEncontrado) {
		return nil, ErrVeiculoNaoEncontrado
	}
	return v, err
}

// ListarVeiculosDoCliente devolve os veículos do cliente.
func (s *CadastroService) ListarVeiculosDoCliente(ctx context.Context, clienteID string) ([]*cadastro.Veiculo, error) {
	return s.veiculos.ListarDoCliente(ctx, clienteID)
}

// VeiculoDoCliente implementa VerificadorDeVeiculo: o cliente e o veículo existem e o veículo é do cliente.
func (s *CadastroService) VeiculoDoCliente(ctx context.Context, clienteID, veiculoID string) error {
	if _, err := s.ConsultarCliente(ctx, clienteID); err != nil {
		return err
	}
	v, err := s.ConsultarVeiculo(ctx, veiculoID)
	if err != nil {
		return err
	}
	if v.ClienteID() != clienteID {
		return ErrVeiculoDeOutroCliente
	}
	return nil
}
