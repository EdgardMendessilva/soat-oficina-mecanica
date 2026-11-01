-- Cadastro (clientes e veículos) e catálogo de serviços. Preço em BIGINT (centavos).

CREATE TABLE clientes (
    id        TEXT PRIMARY KEY,
    documento TEXT NOT NULL UNIQUE,
    nome      TEXT NOT NULL,
    email     TEXT NOT NULL,
    telefone  TEXT NOT NULL DEFAULT '',
    endereco  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE veiculos (
    id         TEXT PRIMARY KEY,
    cliente_id TEXT NOT NULL REFERENCES clientes (id),
    placa      TEXT NOT NULL UNIQUE,
    fabricante TEXT NOT NULL,
    modelo     TEXT NOT NULL,
    ano        INTEGER NOT NULL CHECK (ano BETWEEN 1900 AND 2100),
    cor        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX veiculos_cliente_idx ON veiculos (cliente_id);

CREATE TABLE servicos_catalogo (
    id    TEXT PRIMARY KEY,
    nome  TEXT NOT NULL,
    preco BIGINT NOT NULL CHECK (preco >= 0)
);
