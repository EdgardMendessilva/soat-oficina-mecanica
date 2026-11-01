# Diagramas

## 1. Arquitetura em camadas

```mermaid
flowchart TB
  subgraph Interfaces
    HTTP[Rotas REST, JWT, Swagger]
  end
  subgraph Aplicacao[Aplicação]
    UC[Casos de uso e portas]
  end
  subgraph Dominio[Domínio]
    DOM[Agregados, entidades, objetos de valor]
  end
  subgraph Infra[Infraestrutura]
    PG[(PostgreSQL)]
    SEG[JWT e hash de senha]
    MEM[Repositórios em memória]
  end
  HTTP --> UC --> DOM
  Infra -. implementa as portas .-> UC
  PG --> DOM
```

O domínio não depende de nenhuma outra camada. As portas (repositórios, transação, relógio, autorizador do gerente) são interfaces da camada de aplicação, implementadas na infraestrutura.

## 2. Mapa de contextos

```mermaid
flowchart LR
  ID[Identidade e Acesso]
  CAD[Cadastro]
  CAT[Catálogo]
  OS[Ordem de Serviço - core]
  EST[Estoque]

  ID -- "quem é o usuário e o perfil" --> OS
  ID --> CAD
  CAD -- "clienteId, veiculoId" --> OS
  CAT -- "serviço e preço" --> OS
  OS -- "baixa na aprovação (mesma transação)" --> EST
  EST -- "peça existe e tem saldo" --> OS
```

| Relação | Tipo | Como funciona |
|---|---|---|
| Cadastro → Ordem de Serviço | Fornecedor e cliente | A OS guarda só `clienteId` e `veiculoId` |
| Catálogo → Ordem de Serviço | Fornecedor e cliente | A OS guarda a descrição e o preço do serviço no momento da inclusão |
| Ordem de Serviço ↔ Estoque | Parceria, coordenada pela camada de aplicação | A aprovação confere e baixa o saldo numa única transação |
| Identidade e Acesso → demais | Conformista | Os demais aceitam o perfil e o cliente que vêm do token |

Um contexto não altera os dados de outro diretamente: a coordenação acontece nos casos de uso (ADR 0003).

## 3. Máquina de estados da OS

```mermaid
stateDiagram-v2
  state "Em diagnóstico" as EmDiagnostico
  state "Aguardando aprovação" as AguardandoAprovacao
  state "Em execução" as EmExecucao
  [*] --> Recebida: abrir OS
  Recebida --> EmDiagnostico: iniciar diagnóstico
  EmDiagnostico --> AguardandoAprovacao: enviar orçamento
  AguardandoAprovacao --> EmExecucao: cliente aprova (baixa o estoque)
  AguardandoAprovacao --> Finalizada: cliente reprova
  EmExecucao --> Finalizada: finalizar
  Finalizada --> Entregue: entregar
  Entregue --> [*]
```

## 4. Agregados e objetos de valor

```mermaid
classDiagram
  class OrdemDeServico {
    id
    clienteId
    veiculoId
    status
    decisao
    +IniciarDiagnostico()
    +AdicionarServico()
    +AdicionarPeca()
    +EnviarOrcamento()
    +Aprovar(quemDecide)
    +Reprovar(quemDecide)
    +AplicarDesconto()
    +Finalizar()
    +Entregar()
    +TotalDoOrcamento() Dinheiro
    +ValorAPagar() Dinheiro
  }
  class Servico { descricao }
  class Peca { pecaId, descricao }
  class Desconto { pontosBase, base, autorizadoPor, em }
  class MudancaDeStatus { de, para, em }
  class ItemDeEstoque {
    id
    nome
    tipo
    +Baixar(q)
    +Repor(q)
  }
  class Usuario { id, documento, perfil, senhaHash }
  class Dinheiro { centavos }
  class Quantidade { valor }
  class Documento { numero, tipo }
  class Placa { valor }

  OrdemDeServico "1" *-- "0..*" Servico
  OrdemDeServico "1" *-- "0..*" Peca
  OrdemDeServico "1" *-- "0..1" Desconto
  OrdemDeServico "1" *-- "1..*" MudancaDeStatus
  Servico --> Dinheiro
  Peca --> Dinheiro
  Peca --> Quantidade
  ItemDeEstoque --> Quantidade
  Usuario --> Documento
  Peca ..> ItemDeEstoque : referencia por pecaId
```

## 5. Sequência: aprovação do orçamento com baixa de estoque (RN06, RN09)

```mermaid
sequenceDiagram
  actor C as Cliente
  participant API as API HTTP
  participant UC as Caso de uso
  participant OS as OS (domínio)
  participant BD as PostgreSQL

  C->>API: POST /ordens/{id}/aprovacao (token)
  API->>UC: AprovarOrcamento(id, clienteId do token)
  UC->>BD: BEGIN
  UC->>BD: busca a OS (FOR UPDATE)
  UC->>BD: busca os itens de estoque (FOR UPDATE)
  UC->>UC: confere o saldo de todos os itens
  alt saldo insuficiente
    UC->>BD: ROLLBACK
    UC-->>API: erro de estoque
    API-->>C: 409, nada foi alterado
  else saldo suficiente
    UC->>OS: Aprovar(clienteId)
    OS-->>UC: ok, status Em execução
    UC->>BD: grava a baixa e a OS
    UC->>BD: COMMIT
    API-->>C: 200 com a OS atualizada
  end
```

## 6. Modelo de dados

```mermaid
erDiagram
  usuarios {
    text id PK
    text documento UK
    text perfil
    text senha_hash
  }
  itens_estoque {
    text id PK
    text nome
    text tipo
    int saldo
  }
  ordens_servico {
    text id PK
    text cliente_id
    text veiculo_id
    bigint valor_orcamento
    text status
    text decisao
  }
  os_servicos {
    text os_id FK
    int posicao
    text descricao
    bigint preco
  }
  os_pecas {
    text os_id FK
    int posicao
    text peca_id
    bigint preco_unitario
    int quantidade
  }
  os_historico {
    text os_id FK
    int posicao
    text de
    text para
    timestamptz em
  }
  ordens_servico ||--o{ os_servicos : tem
  ordens_servico ||--o{ os_pecas : tem
  ordens_servico ||--|{ os_historico : registra
  os_pecas }o..|| itens_estoque : "referencia por peca_id"
```

Dinheiro em `BIGINT` de centavos. O saldo do estoque tem `CHECK (saldo >= 0)`, a última defesa contra saldo negativo.
