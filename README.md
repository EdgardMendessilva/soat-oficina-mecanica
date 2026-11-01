# Sistema Integrado da Oficina Mecânica

MVP back-end do **Tech Challenge Fase 1** da pós-graduação POS TECH **SOAT — Software Architecture** (FIAP + Alura).

## Objetivo

Substituir o controle manual (papel e planilhas) de uma oficina mecânica por um sistema único que cobre o atendimento, o diagnóstico, o orçamento, a execução e a entrega, com controle de estoque e acompanhamento do status pelo cliente.

## Solução

Uma API REST que organiza o ciclo da **Ordem de Serviço (OS)**:

1. O atendente cadastra o cliente (CPF ou CNPJ) e o veículo (placa válida) e abre a OS.
2. O mecânico faz o diagnóstico e inclui serviços e peças. O **orçamento é calculado automaticamente**.
3. O orçamento é enviado e **só o cliente dono do veículo** aprova ou reprova.
4. Na **aprovação**, as peças saem do estoque **na mesma transação** (sem saldo, nada muda). Na **reprovação**, o cliente paga só o valor do orçamento.
5. O gerente pode dar **desconto de até 10%** com a própria senha. A OS segue: Em execução, Finalizada, Entregue.
6. O gerente acompanha o **tempo médio de execução**.

Status da OS: Recebida → Em diagnóstico → Aguardando aprovação → Em execução → Finalizada → Entregue. Regras de negócio completas em [`docs/requisitos/escopo-e-regras-de-negocio.md`](docs/requisitos/escopo-e-regras-de-negocio.md).

## Arquitetura

Monólito em camadas, modelado com DDD:

```
Interfaces (HTTP, JWT, Swagger)  ->  Aplicação (casos de uso)  ->  Domínio
Infraestrutura (PostgreSQL, segurança)  ->  Domínio
```

Cinco contextos delimitados: **Ordem de Serviço** (core), **Cadastro**, **Catálogo**, **Estoque** e **Identidade e Acesso**. O domínio não depende de nenhuma outra camada. Decisões em [`docs/adr`](docs/adr).

## Tecnologias

| Item | Uso |
|---|---|
| Go 1.26 | Linguagem. Servidor HTTP, JWT e hash de senha com a biblioteca padrão |
| PostgreSQL 16 | Banco de dados |
| pgx v5 | Driver do PostgreSQL |
| Docker e Docker Compose | Build e ambiente completo (API + banco) |
| OpenAPI 3 e Swagger UI | Documentação da API |
| GitHub Actions | CI: `go vet`, `golangci-lint`, testes com `-race`, cobertura mínima de 80% |

## Pré-requisitos

- **Docker** com Docker Compose (caminho recomendado), ou
- **Go 1.26** para rodar sem Docker (os dados ficam em memória).

## Como executar

Com Docker:

```bash
cp .env.example .env     # ajuste JWT_SECRET (16 ou mais caracteres) e a senha do banco
docker compose up --build
```

A API sobe em `http://localhost:8080`, com o PostgreSQL 16 e as migrations aplicadas na inicialização.

Sem Docker (dados em memória, somem ao reiniciar):

```bash
export JWT_SECRET="um-segredo-com-16-ou-mais-caracteres"
export SEED_DEMO=true
go run ./cmd/api
```

No PowerShell: `$env:JWT_SECRET="..."; $env:SEED_DEMO="true"; go run ./cmd/api`.

Com `DATABASE_URL` definida a API usa o PostgreSQL; sem ela, usa a memória.

## Como executar os testes

```bash
go test ./...                 # testes unitários e de API
go test -cover ./...          # com cobertura
go test -race ./...           # com detector de corrida (precisa de cgo; no Windows use go test -cover)
```

Os testes de integração contra o PostgreSQL rodam quando `TEST_DATABASE_URL` aponta para um banco de testes (as tabelas são apagadas), por exemplo:

```bash
docker run -d --name pg-teste -e POSTGRES_PASSWORD=teste -e POSTGRES_DB=teste -p 5433:5432 postgres:16-alpine
TEST_DATABASE_URL="postgres://postgres:teste@localhost:5433/teste?sslmode=disable" go test ./internal/infrastructure/postgres/...
```

O CI roda tudo isso, com PostgreSQL, a cada pull request. A cobertura de `internal/domain` + `internal/application` tem mínimo de 80%.

## Swagger

Com a API no ar: **http://localhost:8080/swagger** (a especificação está em `/openapi.json`).

1. Abra `POST /auth/login`, informe um CPF e a senha de demonstração e copie o `token`.
2. Clique em **Authorize** e cole o token.

Também há uma coleção para Postman ou Insomnia, com o fluxo completo e as recusas esperadas: [`docs/postman/oficina.postman_collection.json`](docs/postman/oficina.postman_collection.json).

## Autenticação e usuários de demonstração

O login é por **CPF + senha**, e devolve um token **JWT** (HS256, 8 horas). As rotas administrativas e de OS exigem o token e o perfil adequado. Com `SEED_DEMO=true` (padrão no `docker-compose.yml`) a API cria um usuário de cada perfil. A senha de todos é `Teste@1234`.

| Perfil | CPF | Pode |
|---|---|---|
| Administrador | 111.444.777-35 | Papéis de funcionário |
| Gerente | 529.982.247-25 | Tudo do atendente e do mecânico, desconto, estoque, catálogo, tempo médio |
| Atendente | 390.533.447-05 | Cadastrar clientes e veículos, abrir OS, enviar orçamento, entregar |
| Mecânico | 168.995.350-09 | Diagnóstico, serviços e peças, finalizar |
| Cliente (`cli-demo`) | 935.411.347-80 | Ver as próprias OS, aprovar ou reprovar o orçamento |
| Outro cliente (`cli-outro`) | 862.883.667-57 | Idem, para testar o isolamento entre clientes |

Em ambiente real, use `SEED_DEMO=false` e defina `ADMIN_DOCUMENTO` e `ADMIN_SENHA` para criar o primeiro Administrador. Esses usuários e CPFs são só de demonstração.

## Banco de dados: PostgreSQL 16

Os dados da oficina são relacionais (cliente, veículo, OS, serviços, peças, estoque) e a aprovação do orçamento precisa mudar várias coisas juntas: status da OS, baixa das peças e registro de quem decidiu.

- **Transações ACID:** a aprovação e a baixa de estoque acontecem numa transação só; faltando peça, nada é gravado.
- **Concorrência:** `SELECT ... FOR UPDATE` trava o saldo durante a baixa, e duas aprovações simultâneas não deixam o estoque negativo.
- **Integridade no banco:** chaves estrangeiras, `UNIQUE` (CPF ou CNPJ, placa) e `CHECK (saldo >= 0)` reforçam as regras do domínio.
- **Operação simples:** imagem oficial no Docker e migrations versionadas aplicadas na inicialização.

Alternativas descartadas (MongoDB, MySQL, SQLite) em [`docs/adr/0002-banco-postgresql.md`](docs/adr/0002-banco-postgresql.md).

## Documentação DDD

[`docs/ddd`](docs/ddd): Event Storming dos três fluxos (criação, acompanhamento e gestão de peças e insumos), eventos, comandos, políticas, atores, agregados, entidades, objetos de valor, contextos delimitados, diagramas, linguagem ubíqua e a relação da documentação com o código. Há também os PDFs `event-storming.pdf` e `diagramas.pdf`.

## Estrutura dos diretórios

```
cmd/api/                       ponto de entrada da API
internal/domain/               regras de negócio (ordemservico, cadastro, catalogo, estoque, identidade, valueobject)
internal/application/          casos de uso e portas (interfaces)
internal/infrastructure/       PostgreSQL (postgres), memória (memoria), JWT e senha (seguranca)
internal/interfaces/http/      rotas REST, autenticação, erros e openapi.json
docs/adr/                      decisões de arquitetura
docs/requisitos/               escopo e regras de negócio
docs/ddd/                      documentação DDD
docs/postman/                  coleção de requisições
.github/workflows/ci.yml       pipeline de integração contínua
Dockerfile, docker-compose.yml build e ambiente completo
```

## O que o MVP não inclui

Primeiro acesso por e-mail e cadastro de funcionários pela API (modelados nas regras RN11 e RN13, fora do MVP entregue: os usuários vêm da inicialização), isenção manual pelo gerente e várias unidades. Detalhes em [`docs/ddd/documentacao-e-codigo.md`](docs/ddd/documentacao-e-codigo.md).

## Vídeo da entrega

<LINK DO VÍDEO NO YOUTUBE>
