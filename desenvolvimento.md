# Guia de desenvolvimento

Documento de trabalho do projeto. Quem for mexer no código lê este arquivo antes, faz a tarefa da seção 6, atualiza as seções 5 e 7 e responde ao Edgard com o resumo do que foi feito.

## 1. O projeto

Sistema Integrado da Oficina Mecânica: MVP back-end do **Tech Challenge Fase 1** da pós POS TECH "SOAT — Software Architecture" (FIAP + Alura). Feito por uma pessoa só (Edgard Mendes Silva).

- Monolito em camadas, com DDD. API REST em **Go**, banco **PostgreSQL 16**.
- Repositório: https://github.com/EdgardMendessilva/soat-oficina-mecanica
- Tarefas no Jira (projeto TT).
- Prazo de entrega no portal: **03/11/2026**. Meta: entregar em 02/11.

Entregáveis do desafio: repositório público, README completo, documentação DDD, Swagger, Dockerfile e docker-compose, testes com cobertura de 80% nos domínios críticos, relatório de vulnerabilidades, vídeo de até 15 minutos e um PDF único com os links, a branch e o hash do commit final.

## 2. Regras de trabalho (obrigatórias)

- A `main` é protegida: tudo entra por **Pull Request**. Nada de push direto, force push ou reescrita de histórico.
- Uma branch por assunto, a partir da `main`. Commits em português, no formato `feat: descrição (TT-37, TT-38)`.
- Commits e PRs saem no nome do autor: `Edgard Mendes Silva <edgard.nascimento.mendes.silva@gmail.com>`, sem `Co-Authored-By`, sem links de sessão e sem rodapés automáticos nas descrições.
- O CI (`qualidade`) precisa passar: `go vet`, `golangci-lint`, `go test -race` e cobertura mínima de 80% em `internal/domain` + `internal/application`.
- Vocabulário do negócio: **reprovar** (não rejeitar), **funcionário** (não colaborador), **fabricante** (não marca), **valor do orçamento**.
- Dinheiro sempre em **centavos inteiros**. Senha só como hash (PBKDF2), nunca em texto.

## 3. Arquitetura

```
Interfaces (internal/interfaces/http)  ->  Application (internal/application)  ->  Domain (internal/domain)
Infrastructure (internal/infrastructure)  ->  Domain
```

O domínio não importa nada de infraestrutura. As **portas** (interfaces) ficam em `internal/application/portas.go`; a infraestrutura as implementa.

Contextos delimitados: Ordem de Serviço (core), Cadastro, Catálogo, Estoque, Identidade e Acesso. Decisões em `docs/adr/` (0001 a 0003). Escopo e regras de negócio (RN01 a RN15) em `docs/requisitos/escopo-e-regras-de-negocio.md`.

Regras que mais pesam no código:

- **RN03** fluxo de status: Recebida → Em diagnóstico → Aguardando aprovação → Em execução → Finalizada → Entregue. Na reprovação do orçamento vai direto de Aguardando aprovação para Finalizada.
- **RN05** cobrança: orçamento aprovado paga serviços + peças (e o valor do orçamento é isento); reprovado paga só o valor do orçamento.
- **RN06 / RN15** só o cliente dono aprova ou reprova; nenhum funcionário decide por ele.
- **RN07** desconto só com orçamento aprovado, teto de 10%, autorizado pelo gerente com a senha de login.
- **RN09** a aprovação baixa o estoque **na mesma transação**; sem saldo, nada muda.
- **RN10** o cliente só vê as próprias OS.
- **RN12** senha: 8+ caracteres, maiúscula, minúscula, número e caractere especial.

## 4. Como rodar

```
cp .env.example .env        # ajuste JWT_SECRET (16+ caracteres) e as senhas do banco
docker compose up --build   # API em http://localhost:8080 , Swagger em /swagger
go test ./...               # testes
go test -cover ./...        # cobertura (no Windows, -race exige cgo)
```

Usuários de demonstração (`SEED_DEMO=true`), senha **Teste@1234**:

| Perfil | CPF |
|---|---|
| Administrador | 111.444.777-35 |
| Gerente | 529.982.247-25 |
| Atendente | 390.533.447-05 |
| Mecânico | 168.995.350-09 |
| Cliente (`cli-demo`) | 935.411.347-80 |
| Outro cliente (`cli-outro`) | 862.883.667-57 |

## 5. Estado atual

Pronto e na `main`:

- Documentação de escopo e ADRs; esqueleto Go por camadas; proteção da `main`; template de PR; CI.
- Domínio: objetos de valor (CPF/CNPJ, placa, dinheiro, quantidade), Ordem de Serviço (status, orçamento, decisão, desconto), estoque, identidade (perfis e política de senha).
- Casos de uso: fluxo completo da OS, estoque, login.
- API HTTP: login JWT (HS256, só biblioteca padrão), rotas da OS e do estoque, perfis por rota, erros padronizados, Swagger em `/swagger` (`internal/interfaces/http/openapi.json`).
- Dockerfile e docker-compose (API + Postgres 16).

**Limitação atual:** os dados ficam em **memória** (`internal/infrastructure/memoria`) e somem ao reiniciar. O Postgres está no compose, mas a API ainda não usa.

Ainda não existe: persistência PostgreSQL, rotas de cadastro de clientes e veículos, primeiro acesso por e-mail, documentação DDD, README completo, relatório de vulnerabilidades.

## 6. Tarefa em andamento: persistência PostgreSQL (TT-37, TT-38, TT-40)

Objetivo: a API usar o Postgres do compose no lugar da memória.

1. Adicionar o driver (`github.com/jackc/pgx/v5` com `stdlib`, ou `github.com/lib/pq`) e rodar `go mod tidy`.
2. Criar `internal/infrastructure/postgres` com repositórios que implementem `OrdemRepositorio`, `EstoqueRepositorio`, `UsuarioRepositorio` e `AutorizadorGerente`, e uma `Transacao` real: `BEGIN`/`COMMIT`, o contexto carregando o `*sql.Tx`, e `SELECT ... FOR UPDATE` no saldo do estoque (RN09).
3. Migrations SQL embutidas (`go:embed`) aplicadas na inicialização: usuários, ordens, serviços e peças da OS, histórico de status, estoque. Valores em `BIGINT` (centavos).
4. Em `cmd/api/main.go`: se `DATABASE_URL` existir, usar Postgres; senão, manter a memória. Fazer o seed dos usuários demo também no Postgres, de forma idempotente (`memoria.Semear` tem a lógica de referência).
5. Testes de integração com banco real, usando o Postgres do compose via variável de ambiente de teste; pular o teste se a variável não existir. Cobrir: fluxo completo da OS e "aprovação sem estoque suficiente não altera nada".
6. Rodar `go test ./...`, `go vet ./...` e `docker compose up --build`, e validar pelo Swagger o fluxo: login → abrir OS → diagnóstico → serviço e peça → orçamento → aprovação do cliente → finalização → entrega.

Não mexer em `internal/interfaces/http` (outra frente trabalha lá). Se precisar de uma porta nova, ajustar `portas.go` e avisar.

## 7. Registro do que foi feito

Quem mexer no projeto acrescenta uma entrada aqui, no fim, no formato abaixo, e atualiza a seção 5.

```
### AAAA-MM-DD — resumo curto
- O que foi feito
- Arquivos principais
- Como validar (comandos)
- O que ficou pendente ou deu problema
```

(sem entradas ainda)

## 8. Ao terminar, responder ao Edgard com

1. O que foi feito, em poucas linhas.
2. Quais comandos rodou e o resultado (`go test ./...`, `go vet ./...`, `docker compose up --build`, o fluxo pelo Swagger).
3. O que não conseguiu fazer ou ficou duvidoso.
4. O link do PR.
