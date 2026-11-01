-- Valores monetários em BIGINT (centavos inteiros).

CREATE TABLE usuarios (
    id         TEXT PRIMARY KEY,
    nome       TEXT NOT NULL,
    documento  TEXT NOT NULL UNIQUE,
    perfil     TEXT NOT NULL,
    cliente_id TEXT NOT NULL DEFAULT '',
    senha_hash TEXT NOT NULL
);

CREATE TABLE itens_estoque (
    id    TEXT PRIMARY KEY,
    nome  TEXT NOT NULL,
    tipo  TEXT NOT NULL CHECK (tipo IN ('Peça', 'Insumo')),
    saldo INTEGER NOT NULL CHECK (saldo >= 0)
);

CREATE TABLE ordens_servico (
    id                      TEXT PRIMARY KEY,
    sequencia               BIGSERIAL NOT NULL,
    cliente_id              TEXT NOT NULL,
    veiculo_id              TEXT NOT NULL,
    valor_orcamento         BIGINT NOT NULL CHECK (valor_orcamento >= 0),
    status                  TEXT NOT NULL,
    decisao                 TEXT NOT NULL,
    decidido_por            TEXT NOT NULL DEFAULT '',
    decidido_em             TIMESTAMPTZ,
    desconto_pontos_base    BIGINT,
    desconto_base           TEXT,
    desconto_valor          BIGINT,
    desconto_autorizado_por TEXT,
    desconto_em             TIMESTAMPTZ
);
CREATE INDEX ordens_servico_cliente_idx ON ordens_servico (cliente_id);

CREATE TABLE os_servicos (
    os_id     TEXT NOT NULL REFERENCES ordens_servico (id) ON DELETE CASCADE,
    posicao   INTEGER NOT NULL,
    descricao TEXT NOT NULL,
    preco     BIGINT NOT NULL CHECK (preco >= 0),
    PRIMARY KEY (os_id, posicao)
);

CREATE TABLE os_pecas (
    os_id          TEXT NOT NULL REFERENCES ordens_servico (id) ON DELETE CASCADE,
    posicao        INTEGER NOT NULL,
    peca_id        TEXT NOT NULL,
    descricao      TEXT NOT NULL,
    preco_unitario BIGINT NOT NULL CHECK (preco_unitario >= 0),
    quantidade     INTEGER NOT NULL CHECK (quantidade > 0),
    PRIMARY KEY (os_id, posicao)
);

CREATE TABLE os_historico (
    os_id   TEXT NOT NULL REFERENCES ordens_servico (id) ON DELETE CASCADE,
    posicao INTEGER NOT NULL,
    de      TEXT NOT NULL DEFAULT '',
    para    TEXT NOT NULL,
    em      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (os_id, posicao)
);
