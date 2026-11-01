// Package memoria implementa as portas da aplicação com dados em memória.
// Serve para rodar e testar a API sem banco; o adaptador PostgreSQL é a versão persistente.
package memoria

import (
	"context"
	"sync"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/cadastro"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/catalogo"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/estoque"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

// Ordens guarda as OS.
type Ordens struct {
	mu    sync.RWMutex
	dados map[string]*ordemservico.OrdemDeServico
	ordem []string
}

// NovasOrdens cria o repositório vazio.
func NovasOrdens() *Ordens { return &Ordens{dados: map[string]*ordemservico.OrdemDeServico{}} }

// BuscarPorID devolve a OS ou ErrNaoEncontrado.
func (r *Ordens) BuscarPorID(_ context.Context, id string) (*ordemservico.OrdemDeServico, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.dados[id]
	if !ok {
		return nil, application.ErrNaoEncontrado
	}
	return o, nil
}

// Salvar grava ou atualiza a OS.
func (r *Ordens) Salvar(_ context.Context, o *ordemservico.OrdemDeServico) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.dados[o.ID()]; !ok {
		r.ordem = append(r.ordem, o.ID())
	}
	r.dados[o.ID()] = o
	return nil
}

// ListarTodas devolve as OS na ordem de criação.
func (r *Ordens) ListarTodas(_ context.Context) ([]*ordemservico.OrdemDeServico, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ordemservico.OrdemDeServico, 0, len(r.ordem))
	for _, id := range r.ordem {
		out = append(out, r.dados[id])
	}
	return out, nil
}

// Estoque guarda os itens de estoque.
type Estoque struct {
	mu    sync.RWMutex
	dados map[string]*estoque.Item
}

// NovoEstoque cria o repositório vazio.
func NovoEstoque() *Estoque { return &Estoque{dados: map[string]*estoque.Item{}} }

// BuscarPorID devolve o item ou ErrNaoEncontrado.
func (r *Estoque) BuscarPorID(_ context.Context, id string) (*estoque.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.dados[id]
	if !ok {
		return nil, application.ErrNaoEncontrado
	}
	return i, nil
}

// Salvar grava ou atualiza o item.
func (r *Estoque) Salvar(_ context.Context, i *estoque.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[i.ID()] = i
	return nil
}

// Transacao serializa as operações: uma de cada vez, como o SELECT ... FOR UPDATE faria no banco.
type Transacao struct{ mu sync.Mutex }

// Executar roda fn com exclusividade.
func (t *Transacao) Executar(ctx context.Context, fn func(ctx context.Context) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fn(ctx)
}

// Usuarios guarda os usuários e confere senhas.
type Usuarios struct {
	mu     sync.RWMutex
	porID  map[string]identidade.Usuario
	porDoc map[string]string
}

// NovosUsuarios cria o repositório vazio.
func NovosUsuarios() *Usuarios {
	return &Usuarios{porID: map[string]identidade.Usuario{}, porDoc: map[string]string{}}
}

// Adicionar cadastra o usuário.
func (r *Usuarios) Adicionar(u identidade.Usuario) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.porID[u.ID] = u
	r.porDoc[u.Documento] = u.ID
}

// BuscarPorDocumento devolve o usuário do CPF informado.
func (r *Usuarios) BuscarPorDocumento(_ context.Context, documento string) (identidade.Usuario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.porDoc[documento]
	if !ok {
		return identidade.Usuario{}, application.ErrNaoEncontrado
	}
	return r.porID[id], nil
}

// SenhaConfere implementa application.AutorizadorGerente: só vale para perfil Gerente (RN07).
func (r *Usuarios) SenhaConfere(_ context.Context, gerenteID, senha string) (bool, error) {
	r.mu.RLock()
	u, ok := r.porID[gerenteID]
	r.mu.RUnlock()
	if !ok || u.Perfil != identidade.Gerente {
		return false, nil
	}
	return seguranca.ConferirSenha(senha, u.SenhaHash), nil
}

// Clientes guarda os clientes.
type Clientes struct {
	mu    sync.RWMutex
	dados map[string]*cadastro.Cliente
	ordem []string
}

// NovosClientes cria o repositório vazio.
func NovosClientes() *Clientes { return &Clientes{dados: map[string]*cadastro.Cliente{}} }

// BuscarPorID devolve o cliente ou ErrNaoEncontrado.
func (r *Clientes) BuscarPorID(_ context.Context, id string) (*cadastro.Cliente, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.dados[id]; ok {
		return c, nil
	}
	return nil, application.ErrNaoEncontrado
}

// BuscarPorDocumento devolve o cliente do CPF ou CNPJ (só dígitos).
func (r *Clientes) BuscarPorDocumento(_ context.Context, documento string) (*cadastro.Cliente, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.dados {
		if c.Documento().Numero() == documento {
			return c, nil
		}
	}
	return nil, application.ErrNaoEncontrado
}

// Salvar grava ou atualiza o cliente.
func (r *Clientes) Salvar(_ context.Context, c *cadastro.Cliente) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.dados[c.ID()]; !ok {
		r.ordem = append(r.ordem, c.ID())
	}
	r.dados[c.ID()] = c
	return nil
}

// ListarTodos devolve os clientes na ordem de cadastro.
func (r *Clientes) ListarTodos(_ context.Context) ([]*cadastro.Cliente, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*cadastro.Cliente, 0, len(r.ordem))
	for _, id := range r.ordem {
		out = append(out, r.dados[id])
	}
	return out, nil
}

// Veiculos guarda os veículos.
type Veiculos struct {
	mu    sync.RWMutex
	dados map[string]*cadastro.Veiculo
	ordem []string
}

// NovosVeiculos cria o repositório vazio.
func NovosVeiculos() *Veiculos { return &Veiculos{dados: map[string]*cadastro.Veiculo{}} }

// BuscarPorID devolve o veículo ou ErrNaoEncontrado.
func (r *Veiculos) BuscarPorID(_ context.Context, id string) (*cadastro.Veiculo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if v, ok := r.dados[id]; ok {
		return v, nil
	}
	return nil, application.ErrNaoEncontrado
}

// BuscarPorPlaca devolve o veículo da placa normalizada.
func (r *Veiculos) BuscarPorPlaca(_ context.Context, placa string) (*cadastro.Veiculo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.dados {
		if v.Placa().String() == placa {
			return v, nil
		}
	}
	return nil, application.ErrNaoEncontrado
}

// Salvar grava ou atualiza o veículo.
func (r *Veiculos) Salvar(_ context.Context, v *cadastro.Veiculo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.dados[v.ID()]; !ok {
		r.ordem = append(r.ordem, v.ID())
	}
	r.dados[v.ID()] = v
	return nil
}

// ListarDoCliente devolve os veículos do cliente.
func (r *Veiculos) ListarDoCliente(_ context.Context, clienteID string) ([]*cadastro.Veiculo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*cadastro.Veiculo
	for _, id := range r.ordem {
		if v := r.dados[id]; v.ClienteID() == clienteID {
			out = append(out, v)
		}
	}
	return out, nil
}

// Servicos guarda o catálogo de serviços.
type Servicos struct {
	mu    sync.RWMutex
	dados map[string]*catalogo.Servico
	ordem []string
}

// NovosServicos cria o repositório vazio.
func NovosServicos() *Servicos { return &Servicos{dados: map[string]*catalogo.Servico{}} }

// BuscarPorID devolve o serviço ou ErrNaoEncontrado.
func (r *Servicos) BuscarPorID(_ context.Context, id string) (*catalogo.Servico, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.dados[id]; ok {
		return s, nil
	}
	return nil, application.ErrNaoEncontrado
}

// Salvar grava ou atualiza o serviço.
func (r *Servicos) Salvar(_ context.Context, s *catalogo.Servico) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.dados[s.ID()]; !ok {
		r.ordem = append(r.ordem, s.ID())
	}
	r.dados[s.ID()] = s
	return nil
}

// Listar devolve o catálogo na ordem de cadastro.
func (r *Servicos) Listar(_ context.Context) ([]*catalogo.Servico, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*catalogo.Servico, 0, len(r.ordem))
	for _, id := range r.ordem {
		out = append(out, r.dados[id])
	}
	return out, nil
}

// Remover apaga o serviço do catálogo.
func (r *Servicos) Remover(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.dados[id]; !ok {
		return application.ErrNaoEncontrado
	}
	delete(r.dados, id)
	for i, x := range r.ordem {
		if x == id {
			r.ordem = append(r.ordem[:i], r.ordem[i+1:]...)
			break
		}
	}
	return nil
}
