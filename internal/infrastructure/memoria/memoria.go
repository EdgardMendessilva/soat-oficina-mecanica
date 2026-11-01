// Package memoria implementa as portas da aplicação com dados em memória.
// Serve para rodar e testar a API sem banco; o adaptador PostgreSQL é a versão persistente.
package memoria

import (
	"context"
	"sync"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
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
