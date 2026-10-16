# ADR 0002 — Banco de dados: PostgreSQL

- **Status:** Aceito
- **Data:** 2026-10-16

## Contexto

Os dados da oficina são fortemente relacionados: o cliente possui veículos, o veículo possui ordens de serviço, e cada OS possui serviços, peças, orçamento e histórico de status. Funcionários e clientes acessam o sistema com login e perfil.

Na aprovação do orçamento várias coisas precisam mudar juntas: o status da OS, a baixa das peças no estoque e o registro de quem aprovou. Se o sistema falhar no meio dessa operação, a oficina pode ficar com estoque baixado e OS sem aprovação, ou o contrário. O enunciado exige tratamento adequado das transações.

## Decisão

Usar **PostgreSQL 16** como banco de dados único do monólito.

## Justificativa

- **Transações ACID:** aprovação do orçamento, baixa de estoque e registro da decisão acontecem numa transação só. Se faltar peça, tudo é desfeito.
- **Controle de concorrência:** `SELECT ... FOR UPDATE` trava o saldo da peça durante a baixa, evitando estoque negativo quando duas OS são aprovadas ao mesmo tempo.
- **Integridade no próprio banco:** chaves estrangeiras, `UNIQUE` (CPF do cliente, placa do veículo, CPF do funcionário) e `CHECK` (saldo em estoque ≥ 0) reforçam as regras do domínio.
- **Consultas simples para gestão:** o tempo médio de execução sai de uma agregação SQL sobre o histórico de status.
- **Ecossistema:** imagem oficial no Docker, suporte a testes de integração com testcontainers e serviços gerenciados (AWS RDS) para fases futuras.

## Alternativas consideradas

- **MongoDB:** flexível para documentos, mas o modelo é relacional e transações entre documentos adicionam complexidade sem ganho.
- **MySQL:** atenderia, mas o PostgreSQL oferece `CHECK` constraints e recursos de bloqueio mais completos.
- **SQLite:** simples para MVP, porém fraco em concorrência de escrita e distante de um ambiente de produção.

## Consequências

- Alterações de estrutura do banco são feitas por migrations versionadas.
- Testes de integração rodam contra PostgreSQL real em container, não contra simulações.
