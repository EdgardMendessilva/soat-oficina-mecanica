# ADR 0003 — Arquitetura: monólito em camadas com DDD

- **Status:** Aceito
- **Data:** 2026-10-16

## Contexto

O problema da oficina é organização: atendimento, diagnóstico, orçamento, execução, estoque e entrega são controlados em papel e planilha, sem integração. O enunciado pede um MVP back-end monolítico, organizado em camadas e modelado com Domain-Driven Design. Nas próximas fases o sistema deve evoluir para Clean Architecture e microsserviços.

## Decisão

Construir um **monólito modular em camadas**, com os contextos do domínio separados em pacotes:

| Camada | Responsabilidade |
|---|---|
| **Interfaces (HTTP)** | Rotas REST, validação de entrada, autenticação, documentação Swagger |
| **Aplicação** | Casos de uso: orquestram o domínio, os repositórios e a transação |
| **Domínio** | Regras de negócio puras: agregados, entidades, objetos de valor, eventos |
| **Infraestrutura** | Banco de dados, geração de token, configuração |

Contextos do domínio (bounded contexts):

| Contexto | Conteúdo |
|---|---|
| **Ordem de Serviço** (core) | OS, orçamento, status, desconto, isenção, decisão do cliente |
| **Cadastro** | Clientes e veículos |
| **Catálogo** | Serviços oferecidos e preços |
| **Estoque** | Peças e insumos, saldo, reserva e baixa |
| **Identidade e Acesso** | Funcionários, perfis, login de funcionário e de cliente |

Regra de dependência: Interfaces → Aplicação → Domínio, e Infraestrutura → Domínio. O domínio não depende de nenhuma outra camada.

## Justificativa

- Atende ao requisito de monólito em camadas sem o custo de operar serviços distribuídos.
- Separar os contextos desde já prepara a extração para microsserviços nas próximas fases.
- O domínio isolado é testável com testes unitários puros, onde fica a cobertura de 80%.

## Alternativas consideradas

- **Microsserviços desde o início:** fora do escopo do MVP e do enunciado, com custo operacional alto.
- **Monólito sem separação por contexto:** mais rápido no início, mas acopla regras e dificulta a evolução.

## Consequências

- Repositórios são definidos como interfaces no domínio e implementados na infraestrutura.
- Um contexto não altera os dados de outro diretamente; a coordenação acontece na camada de aplicação.
- A OS referencia cliente, veículo, serviços e peças por identificador.
