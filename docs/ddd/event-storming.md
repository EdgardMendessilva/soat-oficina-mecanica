# Event Storming

Resultado do Event Storming do domínio da oficina, em três fluxos. Cores usadas, como na aula:

| Cor | Elemento |
|---|---|
| Laranja | Evento de domínio (verbo no passado) |
| Azul | Comando (a intenção de um ator) |
| Amarelo claro | Ator |
| Amarelo escuro | Agregado |
| Lilás | Política ou regra de negócio |
| Rosa | Sistema externo |
| Vermelho | Ponto de atenção (hot spot) |

## Atores

| Ator | Papel |
|---|---|
| Cliente | Dono do veículo; acompanha a OS e aprova ou reprova o orçamento |
| Atendente | Cadastra clientes e veículos, abre a OS, envia o orçamento, entrega o veículo |
| Mecânico | Faz o diagnóstico, inclui serviços e peças, finaliza a execução |
| Gerente | Aplica desconto, cadastra funcionários, gerencia estoque, vê o tempo médio |
| Administrador | RH ou dono; cadastra funcionários |
| Serviço de e-mail (externo) | Entregaria o link de primeiro acesso (RN11 e RN13); fora do MVP entregue |

## Fluxo 1 — Criação da Ordem de Serviço

```mermaid
flowchart LR
  classDef ator fill:#fff7c2,stroke:#b59b00,color:#222
  classDef cmd fill:#a8d4ff,stroke:#2a6db0,color:#222
  classDef evt fill:#ffb766,stroke:#c46a00,color:#222
  classDef pol fill:#d9c2f0,stroke:#7a4fa8,color:#222
  classDef agg fill:#f2d600,stroke:#8a7a00,color:#222

  A1([Atendente]):::ator --> C1[Cadastrar cliente]:::cmd --> G1{{Cliente}}:::agg --> E1(Cliente cadastrado):::evt
  P1[/CPF ou CNPJ válido e único RN01/]:::pol -.-> C1
  A1 --> C2[Cadastrar veículo]:::cmd --> G2{{Veículo}}:::agg --> E4(Veículo cadastrado):::evt
  P3[/Placa válida e única RN02/]:::pol -.-> C2
  A1 --> C3[Abrir OS]:::cmd --> G3{{Ordem de Serviço}}:::agg --> E5(OS aberta, status Recebida):::evt
  M1([Mecânico]):::ator --> C4[Iniciar diagnóstico]:::cmd --> G3 --> E6(Diagnóstico iniciado):::evt
  M1 --> C5[Incluir serviço]:::cmd --> G3 --> E7(Serviço incluído):::evt
  M1 --> C6[Incluir peça ou insumo]:::cmd --> G3 --> E8(Peça incluída):::evt
  P4[/Itens só durante o diagnóstico/]:::pol -.-> C5
  P5[/A peça precisa existir no estoque/]:::pol -.-> C6
  E7 --> E9(Orçamento calculado automaticamente):::evt
  E8 --> E9
  P6[/Total = serviços + peças x quantidade RN04/]:::pol -.-> E9
  A1 --> C7[Enviar orçamento]:::cmd --> G3 --> E10(Orçamento enviado, Aguardando aprovação):::evt
  P7[/Orçamento precisa de ao menos um item/]:::pol -.-> C7
  K1([Cliente]):::ator --> C8[Aprovar orçamento]:::cmd --> G3 --> E11(Orçamento aprovado, OS Em execução):::evt
  K1 --> C9[Reprovar orçamento]:::cmd --> G3 --> E12(Orçamento reprovado, OS Finalizada):::evt
  P8[/Só o cliente dono decide RN06 e RN15/]:::pol -.-> C8
  P9[/Aprovado: isenta o valor do orçamento RN08/]:::pol -.-> E11
  P10[/Reprovado: paga só o valor do orçamento RN05/]:::pol -.-> E12
```

**Primeiro acesso por e-mail (RN11 e RN13):** foi modelado no Event Storming e está nas regras de negócio, mas **não faz parte do MVP entregue**. No MVP o cliente e o funcionário entram com CPF e senha de usuários criados pela inicialização (administrador e usuários de demonstração). O fluxo previsto é: o atendente cadastra o cliente, o sistema envia um link de uso único (sistema externo, rosa), e o cliente define a senha pela política da RN12.

**Pontos de atenção**

- A reprovação não passa por execução: a OS vai de Aguardando aprovação direto para Finalizada.
- Um funcionário nunca decide pelo cliente, mesmo o Administrador (RN15).

## Fluxo 2 — Acompanhamento da Ordem de Serviço

```mermaid
flowchart LR
  classDef ator fill:#fff7c2,stroke:#b59b00,color:#222
  classDef cmd fill:#a8d4ff,stroke:#2a6db0,color:#222
  classDef evt fill:#ffb766,stroke:#c46a00,color:#222
  classDef pol fill:#d9c2f0,stroke:#7a4fa8,color:#222
  classDef agg fill:#f2d600,stroke:#8a7a00,color:#222
  classDef hot fill:#ff7070,stroke:#a00000,color:#222

  S1(Recebida):::evt --> S2(Em diagnóstico):::evt --> S3(Aguardando aprovação):::evt --> S4(Em execução):::evt --> S5(Finalizada):::evt --> S6(Entregue):::evt
  S3 -. reprovação .-> S5
  P1[/O status não pula etapa RN03/]:::pol -.-> S2

  GE([Gerente]):::ator --> D1[Aplicar desconto]:::cmd --> OS{{Ordem de Serviço}}:::agg --> D2(Desconto concedido):::evt
  P2[/Só com orçamento aprovado/]:::pol -.-> D1
  P3[/Teto de 10 por cento RN07/]:::pol -.-> D1
  P4[/Senha do gerente confere/]:::pol -.-> D1
  H1{{Desconto acima do teto ou senha errada é recusado}}:::hot -.-> D1

  ME([Mecânico]):::ator --> F1[Finalizar execução]:::cmd --> OS --> S5
  AT([Atendente]):::ator --> F2[Entregar veículo]:::cmd --> OS --> S6
  CL([Cliente]):::ator --> Q1[Consultar OS]:::cmd --> OS --> Q2(OS consultada):::evt
  P5[/Cliente vê só as próprias OS RN10/]:::pol -.-> Q1
  GE --> Q3[Consultar tempo médio]:::cmd --> Q4(Tempo médio de execução apurado):::evt
```

**Pontos de atenção**

- O desconto não pode ser aplicado duas vezes na mesma OS.
- O tempo médio considera só as OS que já passaram por Em execução e saíram dela.

## Fluxo 3 — Gestão de peças e insumos

```mermaid
flowchart LR
  classDef ator fill:#fff7c2,stroke:#b59b00,color:#222
  classDef cmd fill:#a8d4ff,stroke:#2a6db0,color:#222
  classDef evt fill:#ffb766,stroke:#c46a00,color:#222
  classDef pol fill:#d9c2f0,stroke:#7a4fa8,color:#222
  classDef agg fill:#f2d600,stroke:#8a7a00,color:#222
  classDef hot fill:#ff7070,stroke:#a00000,color:#222

  GE([Gerente]):::ator --> C1[Cadastrar item]:::cmd --> IT{{Item de Estoque}}:::agg --> E1(Item cadastrado com saldo inicial):::evt
  GE --> C2[Repor estoque]:::cmd --> IT --> E2(Estoque reposto):::evt
  GE --> C3[Consultar saldo]:::cmd --> IT --> E3(Saldo consultado):::evt
  P1[/Identificador único por item/]:::pol -.-> C1
  P2[/Reposição maior que zero/]:::pol -.-> C2

  CL([Cliente]):::ator --> A1[Aprovar orçamento]:::cmd --> OS{{Ordem de Serviço}}:::agg
  OS --> P3[/Ao aprovar, baixar as peças da OS RN09/]:::pol --> C4[Baixar estoque]:::cmd --> IT --> E4(Estoque baixado):::evt
  P4[/Saldo de todos os itens conferido antes de mudar qualquer coisa/]:::pol -.-> C4
  H1{{Saldo insuficiente: aprovação recusada e nada muda}}:::hot -.-> C4
  P5[/Baixa e mudança de status na mesma transação/]:::pol -.-> A1
```

**Pontos de atenção**

- Duas aprovações ao mesmo tempo disputando a última unidade: o banco trava a linha do item (`SELECT ... FOR UPDATE`) e só uma aprova.
- Peça repetida na mesma OS é somada antes de conferir o saldo.

## Eventos de domínio

| Evento | Quem causa | Contexto |
|---|---|---|
| Cliente cadastrado | Atendente | Cadastro |
| Veículo cadastrado | Atendente | Cadastro |
| OS aberta | Atendente | Ordem de Serviço |
| Diagnóstico iniciado | Mecânico | Ordem de Serviço |
| Serviço incluído | Mecânico | Ordem de Serviço |
| Peça incluída | Mecânico | Ordem de Serviço |
| Orçamento calculado | Sistema | Ordem de Serviço |
| Orçamento enviado | Atendente | Ordem de Serviço |
| Orçamento aprovado | Cliente | Ordem de Serviço |
| Orçamento reprovado | Cliente | Ordem de Serviço |
| Estoque baixado | Sistema, na aprovação | Estoque |
| Desconto concedido | Gerente | Ordem de Serviço |
| Execução finalizada | Mecânico | Ordem de Serviço |
| Veículo entregue | Atendente | Ordem de Serviço |
| Item de estoque cadastrado | Gerente | Estoque |
| Estoque reposto | Gerente | Estoque |

No MVP os eventos não circulam por um barramento. Eles ficam registrados no próprio agregado: cada mudança de status entra no histórico da OS (de, para, quando), e a decisão do cliente guarda quem decidiu e quando. A evolução para microsserviços pode transformá-los em mensagens.

## Comandos

| Comando | Ator | Endpoint |
|---|---|---|
| Cadastrar cliente | Atendente | `POST /clientes` |
| Cadastrar veículo | Atendente | `POST /veiculos` |
| Abrir OS | Atendente | `POST /ordens` |
| Iniciar diagnóstico | Mecânico | `POST /ordens/{id}/diagnostico` |
| Incluir serviço | Mecânico | `POST /ordens/{id}/servicos` |
| Incluir peça ou insumo | Mecânico | `POST /ordens/{id}/pecas` |
| Enviar orçamento | Atendente | `POST /ordens/{id}/orcamento` |
| Aprovar orçamento | Cliente | `POST /ordens/{id}/aprovacao` |
| Reprovar orçamento | Cliente | `POST /ordens/{id}/reprovacao` |
| Aplicar desconto | Gerente | `POST /ordens/{id}/desconto` |
| Finalizar execução | Mecânico | `POST /ordens/{id}/finalizacao` |
| Entregar veículo | Atendente | `POST /ordens/{id}/entrega` |
| Consultar OS | Cliente ou funcionário | `GET /ordens/{id}`, `GET /ordens` |
| Consultar tempo médio | Gerente | `GET /relatorios/tempo-medio-execucao` |
| Cadastrar serviço no catálogo | Gerente | `POST /servicos` (e `PUT`, `DELETE`, `GET`) |
| Cadastrar item de estoque | Gerente | `POST /estoque` |
| Repor estoque | Gerente | `POST /estoque/{id}/reposicao` |
| Consultar saldo | Funcionário | `GET /estoque/{id}` |

## Políticas e regras de negócio

| Política | Regra |
|---|---|
| CPF ou CNPJ válido e único | RN01 |
| Placa válida e única | RN02 |
| O status segue a ordem e não pula etapa | RN03 |
| Orçamento calculado automaticamente | RN04 |
| Cobrança conforme a decisão do cliente | RN05 |
| Só o cliente dono decide | RN06 e RN15 |
| Desconto de até 10%, com orçamento aprovado e senha do gerente | RN07 |
| Isenção do valor do orçamento na aprovação | RN08 |
| Baixa de estoque na aprovação, na mesma transação, sem saldo negativo | RN09 |
| Cliente só vê as próprias OS | RN10 |
| Cadastro no balcão (implementado) e primeiro acesso por e-mail (modelado, fora do MVP entregue) | RN11 e RN13 |
| Senha forte, guardada só como hash | RN12 |
| O primeiro Administrador nasce na inicialização (implementado); o cadastro de funcionários pela API fica fora do MVP entregue | RN14 |

Texto completo de cada regra em [`../requisitos/escopo-e-regras-de-negocio.md`](../requisitos/escopo-e-regras-de-negocio.md).

## Agregados, entidades e objetos de valor

| Agregado (raiz) | Entidades internas | Objetos de valor |
|---|---|---|
| **Ordem de Serviço** | Serviço, Peça, Desconto, Mudança de status | Dinheiro, Quantidade, Status, Decisão, Base do desconto |
| **Item de Estoque** | — | Quantidade, Tipo (Peça ou Insumo) |
| **Cliente** | — | Documento (CPF ou CNPJ) |
| **Veículo** | — | Placa |
| **Usuário** | — | Documento, Perfil |
| **Serviço do catálogo** | — | Dinheiro |

Regra de consistência: tudo o que muda a OS passa pelos métodos da raiz, que garantem as regras. A OS referencia cliente, veículo e peças só por identificador.

## Contextos delimitados

| Contexto | Responsabilidade | Agregados |
|---|---|---|
| **Ordem de Serviço** (core) | O ciclo da OS, orçamento, decisão, desconto | Ordem de Serviço |
| **Cadastro** | Quem é o cliente e qual é o veículo | Cliente, Veículo |
| **Catálogo** | Serviços oferecidos e seus preços | Serviço do catálogo |
| **Estoque** | Peças e insumos, saldo, reposição e baixa | Item de Estoque |
| **Identidade e Acesso** | Login, perfis, senha | Usuário |

O mapa de contexto está em [`diagramas.md`](diagramas.md).
