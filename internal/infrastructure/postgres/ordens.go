package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/ordemservico"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/valueobject"
)

// Ordens implementa application.OrdemRepositorio.
type Ordens struct {
	db *sql.DB
	tx *Transacao
}

// NovasOrdens cria o repositório de OS.
func NovasOrdens(db *sql.DB) *Ordens { return &Ordens{db: db, tx: NovaTransacao(db)} }

// BuscarPorID devolve a OS completa ou ErrNaoEncontrado.
func (r *Ordens) BuscarPorID(ctx context.Context, id string) (*ordemservico.OrdemDeServico, error) {
	lista, err := r.carregar(ctx, true, `WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	if len(lista) == 0 {
		return nil, application.ErrNaoEncontrado
	}
	return lista[0], nil
}

// ListarTodas devolve as OS na ordem de criação.
func (r *Ordens) ListarTodas(ctx context.Context) ([]*ordemservico.OrdemDeServico, error) {
	return r.carregar(ctx, false, ``)
}

// Salvar grava ou atualiza a OS inteira (cabeçalho, serviços, peças e histórico) de forma
// atômica: usa a transação do contexto ou abre uma própria.
func (r *Ordens) Salvar(ctx context.Context, o *ordemservico.OrdemDeServico) error {
	return r.tx.Executar(ctx, func(ctx context.Context) error {
		ex := executorDe(ctx, r.db)

		var (
			decididoEm         sql.NullTime
			dPontos            sql.NullInt64
			dBase, dAutorizado sql.NullString
			dValor             sql.NullInt64
			dEm                sql.NullTime
		)
		if !o.DecididoEm().IsZero() {
			decididoEm = sql.NullTime{Time: o.DecididoEm(), Valid: true}
		}
		if d := o.Desconto(); d != nil {
			dPontos = sql.NullInt64{Int64: d.PontosBase, Valid: true}
			dBase = sql.NullString{String: string(d.Base), Valid: true}
			dValor = sql.NullInt64{Int64: d.Valor.Centavos(), Valid: true}
			dAutorizado = sql.NullString{String: d.AutorizadoPor, Valid: true}
			dEm = sql.NullTime{Time: d.Em, Valid: true}
		}

		_, err := ex.ExecContext(ctx, `
			INSERT INTO ordens_servico (id, cliente_id, veiculo_id, valor_orcamento, status, decisao, decidido_por, decidido_em,
				desconto_pontos_base, desconto_base, desconto_valor, desconto_autorizado_por, desconto_em)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				cliente_id = EXCLUDED.cliente_id, veiculo_id = EXCLUDED.veiculo_id,
				valor_orcamento = EXCLUDED.valor_orcamento, status = EXCLUDED.status, decisao = EXCLUDED.decisao,
				decidido_por = EXCLUDED.decidido_por, decidido_em = EXCLUDED.decidido_em,
				desconto_pontos_base = EXCLUDED.desconto_pontos_base, desconto_base = EXCLUDED.desconto_base,
				desconto_valor = EXCLUDED.desconto_valor, desconto_autorizado_por = EXCLUDED.desconto_autorizado_por,
				desconto_em = EXCLUDED.desconto_em`,
			o.ID(), o.ClienteID(), o.VeiculoID(), o.ValorDoOrcamento().Centavos(), string(o.Status()),
			string(o.Decisao()), o.DecididoPor(), decididoEm, dPontos, dBase, dValor, dAutorizado, dEm)
		if err != nil {
			return err
		}

		for _, tabela := range []string{"os_servicos", "os_pecas", "os_historico"} {
			if _, err := ex.ExecContext(ctx, `DELETE FROM `+tabela+` WHERE os_id = $1`, o.ID()); err != nil {
				return err
			}
		}
		for i, s := range o.Servicos() {
			if _, err := ex.ExecContext(ctx,
				`INSERT INTO os_servicos (os_id, posicao, descricao, preco) VALUES ($1, $2, $3, $4)`,
				o.ID(), i, s.Descricao, s.Preco.Centavos()); err != nil {
				return err
			}
		}
		for i, p := range o.Pecas() {
			if _, err := ex.ExecContext(ctx,
				`INSERT INTO os_pecas (os_id, posicao, peca_id, descricao, preco_unitario, quantidade) VALUES ($1, $2, $3, $4, $5, $6)`,
				o.ID(), i, p.PecaID, p.Descricao, p.PrecoUnitario.Centavos(), p.Quantidade.Valor()); err != nil {
				return err
			}
		}
		for i, m := range o.Historico() {
			if _, err := ex.ExecContext(ctx,
				`INSERT INTO os_historico (os_id, posicao, de, para, em) VALUES ($1, $2, $3, $4, $5)`,
				o.ID(), i, string(m.De), string(m.Para), m.Em); err != nil {
				return err
			}
		}
		return nil
	})
}

// carregar lê as OS que atendem ao filtro (cláusula WHERE sobre ordens_servico) com os
// itens e o histórico, em quatro consultas, sem uma por OS.
func (r *Ordens) carregar(ctx context.Context, travar bool, filtro string, args ...any) ([]*ordemservico.OrdemDeServico, error) {
	ex := executorDe(ctx, r.db)
	bloqueio := ""
	if _, ok := emTransacao(ctx); ok && travar {
		bloqueio = " FOR UPDATE" // outra transação não altera a mesma OS até o commit
	}

	linhas, err := ex.QueryContext(ctx, `
		SELECT id, cliente_id, veiculo_id, valor_orcamento, status, decisao, decidido_por, decidido_em,
			desconto_pontos_base, desconto_base, desconto_valor, desconto_autorizado_por, desconto_em
		FROM ordens_servico `+filtro+` ORDER BY sequencia`+bloqueio, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = linhas.Close() }()

	var ordem []string
	dados := map[string]*ordemservico.Dados{}
	for linhas.Next() {
		var (
			d                  ordemservico.Dados
			orcamento          int64
			status, decisao    string
			decididoEm         sql.NullTime
			dPontos, dValor    sql.NullInt64
			dBase, dAutorizado sql.NullString
			dEm                sql.NullTime
		)
		if err := linhas.Scan(&d.ID, &d.ClienteID, &d.VeiculoID, &orcamento, &status, &decisao, &d.DecididoPor, &decididoEm,
			&dPontos, &dBase, &dValor, &dAutorizado, &dEm); err != nil {
			return nil, err
		}
		if d.ValorDoOrcamento, err = valueobject.NovoDinheiro(orcamento); err != nil {
			return nil, err
		}
		d.Status, d.Decisao = ordemservico.Status(status), ordemservico.Decisao(decisao)
		if decididoEm.Valid {
			d.DecididoEm = decididoEm.Time
		}
		if dPontos.Valid {
			valor, err := valueobject.NovoDinheiro(dValor.Int64)
			if err != nil {
				return nil, err
			}
			d.Desconto = &ordemservico.Desconto{
				PontosBase: dPontos.Int64, Base: ordemservico.BaseDesconto(dBase.String),
				Valor: valor, AutorizadoPor: dAutorizado.String, Em: dEm.Time,
			}
		}
		ordem = append(ordem, d.ID)
		dados[d.ID] = &d
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}
	_ = linhas.Close()
	if len(ordem) == 0 {
		return nil, nil
	}

	sub := `(SELECT id FROM ordens_servico ` + filtro + `)`

	if err := r.lerFilhos(ctx, `SELECT os_id, descricao, preco FROM os_servicos WHERE os_id IN `+sub+` ORDER BY os_id, posicao`, args,
		func(rs *sql.Rows) error {
			var id, desc string
			var preco int64
			if err := rs.Scan(&id, &desc, &preco); err != nil {
				return err
			}
			p, err := valueobject.NovoDinheiro(preco)
			if err != nil {
				return err
			}
			dados[id].Servicos = append(dados[id].Servicos, ordemservico.Servico{Descricao: desc, Preco: p})
			return nil
		}); err != nil {
		return nil, err
	}

	if err := r.lerFilhos(ctx, `SELECT os_id, peca_id, descricao, preco_unitario, quantidade FROM os_pecas WHERE os_id IN `+sub+` ORDER BY os_id, posicao`, args,
		func(rs *sql.Rows) error {
			var id, pecaID, desc string
			var preco int64
			var qtd int
			if err := rs.Scan(&id, &pecaID, &desc, &preco, &qtd); err != nil {
				return err
			}
			p, err := valueobject.NovoDinheiro(preco)
			if err != nil {
				return err
			}
			q, err := valueobject.NovaQuantidade(qtd)
			if err != nil {
				return err
			}
			dados[id].Pecas = append(dados[id].Pecas, ordemservico.Peca{PecaID: pecaID, Descricao: desc, PrecoUnitario: p, Quantidade: q})
			return nil
		}); err != nil {
		return nil, err
	}

	if err := r.lerFilhos(ctx, `SELECT os_id, de, para, em FROM os_historico WHERE os_id IN `+sub+` ORDER BY os_id, posicao`, args,
		func(rs *sql.Rows) error {
			var id, de, para string
			var em time.Time
			if err := rs.Scan(&id, &de, &para, &em); err != nil {
				return err
			}
			dados[id].Historico = append(dados[id].Historico, ordemservico.MudancaDeStatus{
				De: ordemservico.Status(de), Para: ordemservico.Status(para), Em: em,
			})
			return nil
		}); err != nil {
		return nil, err
	}

	out := make([]*ordemservico.OrdemDeServico, 0, len(ordem))
	for _, id := range ordem {
		out = append(out, ordemservico.Reconstituir(*dados[id]))
	}
	return out, nil
}

func (r *Ordens) lerFilhos(ctx context.Context, consulta string, args []any, linha func(*sql.Rows) error) error {
	rs, err := executorDe(ctx, r.db).QueryContext(ctx, consulta, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rs.Close() }()
	for rs.Next() {
		if err := linha(rs); err != nil {
			return err
		}
	}
	return rs.Err()
}
