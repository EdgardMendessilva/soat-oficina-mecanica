# Escopo do MVP e regras de negócio

- **Projeto:** Sistema Integrado da Oficina Mecânica — Tech Challenge Fase 1 (SOAT)
- **Data:** 2026-10-16
- **Status:** Em definição (há regras pendentes, ver seção 5)

## 1. Problema

A oficina controla atendimento, diagnóstico, orçamento, execução e entrega com anotações manuais e planilhas. Isso causa erros de priorização, falhas no controle de peças, dificuldade de acompanhar o status, perda de histórico de clientes e veículos e um fluxo de orçamento ineficiente. O objetivo é integrar tudo num sistema único.

## 2. Escopo

### 2.1 Dentro do MVP

| Item | Origem |
|---|---|
| Cadastro de cliente: CPF (validado), nome, e-mail, telefone, endereço | Enunciado + decisão do autor |
| Cadastro de veículo: placa (validada), fabricante, modelo, ano, cor | Enunciado + decisão do autor |
| Cadastro de funcionário: CPF (validado), nome, cargo, perfil de acesso | Decisão do autor |
| Login de funcionário (CPF + senha) com JWT e perfis de acesso; cadastro feito pelo gerente | Enunciado (JWT) + decisão do autor (perfis) |
| Administrador (RH ou dono) cadastra colaboradores; o primeiro é criado na primeira inicialização | Decisão do autor |
| Login de cliente (CPF + senha), com primeiro acesso por código entregue no balcão; vê e decide apenas sobre as OS dos próprios veículos | Decisão do autor |
| Ordem de serviço com serviços, peças, orçamento automático e status | Enunciado |
| Aprovação ou rejeição do orçamento pelo cliente via API | Enunciado |
| Desconto de até 10% com orçamento fechado, autorizado com a senha do gerente | Decisão do autor |
| Isenção do valor do orçamento pelo gerente | Decisão do autor |
| CRUD de serviços, peças e insumos com controle de estoque | Enunciado |
| Tempo médio de execução dos serviços | Enunciado |

### 2.2 Fora do MVP (evolução futura)

| Item | Motivo |
|---|---|
| Várias unidades (lojas) | Muda o modelo inteiro (estoque e OS por unidade); não exigido no enunciado |

## 3. Perfis de acesso

| Perfil | Pode |
|---|---|
| **Cliente** | Acesso fechado: acompanhar o fluxo das OS dos próprios veículos, aprovar ou reprovar o orçamento |
| **Atendente** | Cadastrar clientes e veículos, abrir OS, enviar orçamento |
| **Mecânico** | Registrar diagnóstico, incluir serviços e peças, iniciar e finalizar execução |
| **Gerente** | Tudo do atendente e do mecânico, aplicar desconto, isentar orçamento, cadastrar funcionários e definir perfis, gerenciar estoque e catálogo |
| **Administrador** | RH ou dono da oficina; cadastra colaboradores; acesso a todos os papéis de funcionário |

> A divisão de permissões entre atendente e mecânico é uma proposta inicial e pode ser ajustada no Event Storming.

## 4. Regras de negócio

### RN01 — Validação de CPF
Todo CPF (cliente e funcionário) é validado pelos dígitos verificadores. CPF inválido impede o cadastro. Não pode haver dois cadastros com o mesmo CPF.

### RN02 — Validação de placa
A placa aceita o padrão antigo (ABC1234) e o Mercosul (ABC1D23). Placa inválida impede o cadastro. Não pode haver dois veículos com a mesma placa.

### RN03 — Status da OS
Recebida → Em diagnóstico → Aguardando aprovação → Em execução → Finalizada → Entregue. O status muda apenas pelas ações do sistema e não pode pular etapas.

### RN04 — Orçamento automático
O total do orçamento é a soma dos serviços mais as peças multiplicadas pela quantidade. Ele é recalculado sempre que um item muda, até o envio ao cliente.

### RN05 — Cobrança conforme a decisão do cliente

| Decisão | Cliente paga |
|---|---|
| **Aprovado** (orçamento fechado) | Serviços + peças. **Não paga o valor do orçamento.** |
| **Rejeitado** | **Apenas o valor do orçamento** |

Na rejeição, a OS vai para Finalizada sem execução e depois para Entregue, quando o veículo é devolvido. Nenhuma peça é baixada do estoque.

### RN06 — Quem decide o orçamento
Somente o cliente dono do veículo, autenticado, pode aprovar ou rejeitar o orçamento da OS. O sistema registra quem decidiu e quando.

### RN07 — Desconto
**Quando:** somente com o orçamento fechado (aprovado pelo cliente).

**Origem:** o cliente reclama na loja, por WhatsApp ou por telefone. O funcionário que atende repassa ao gerente.

**Base de cálculo:** o desconto pode incidir sobre:
- serviços + peças (total do orçamento), ou
- apenas a mão de obra (serviços).

**Limite:** teto de 10%. Qualquer valor acima é recusado.

**Fluxo:**
1. Na OS, aparece o campo de desconto, sem os percentuais pré-definidos.
2. O funcionário contata o cliente e informa o que vai acontecer.
3. O gerente informa o desconto e a base de cálculo.
4. O gerente autoriza com a própria senha de login.
5. Com a autorização válida, o desconto é concedido e o valor a pagar é recalculado.

Sem a senha do gerente, o desconto não é aplicado. O sistema registra quem autorizou, quando, o percentual e a base.

### RN08 — Isenção do valor do orçamento
O gerente pode isentar o cliente do valor do orçamento quando a oficina não consegue resolver o problema do cliente. (Momento da isenção: ver pendência P3.)

### RN09 — Estoque
A baixa das peças acontece na aprovação do orçamento, junto com a mudança de status, numa única transação. Sem saldo suficiente, a aprovação é recusada e nada muda. O saldo nunca fica negativo.

### RN10 — Isolamento de dados do cliente
O cliente autenticado só acessa OS, orçamentos e veículos vinculados ao próprio cadastro. Tentar acessar dados de outro cliente retorna erro, mesmo com login válido.

### RN11 — Cadastro do cliente e primeiro acesso
1. O cliente chega ao balcão e entrega os documentos.
2. O funcionário faz o cadastro do cliente e do veículo. O sistema gera um **código de primeiro acesso**, que o funcionário entrega ao cliente.
3. O cliente baixa o aplicativo e, no primeiro acesso, informa **CPF + código de primeiro acesso** e define a própria senha.
4. O código vale para um único uso. Depois de usado, não ativa a conta de novo.
5. A partir daí o login do cliente é **CPF + senha**, e ele acompanha as OS pelo celular.

O cliente não se cadastra sozinho. Sem o código entregue no balcão, não é possível ativar a conta.

### RN12 — Política de senha
- Mínimo de 8 caracteres.
- Pelo menos 1 letra maiúscula, 1 letra minúscula, 1 número e 1 caractere especial. Exemplo de formato válido: `Teste@1234`.
- Senha fora do padrão é recusada, com mensagem indicando o que falta.
- A senha nunca é armazenada em texto: o banco guarda apenas o hash.
- Vale para clientes e funcionários.

### RN13 — Cadastro do funcionário e primeiro acesso
1. O **gerente** cadastra o funcionário e define o perfil de acesso dele (atendente, mecânico ou gerente).
2. O sistema gera um **código de primeiro acesso** de uso único, entregue pelo gerente ao funcionário.
3. No primeiro acesso, o funcionário informa **CPF + código** e define a própria senha (RN12).
4. A partir daí o login do funcionário é **CPF + senha**.

Somente o gerente (ou o Administrador) cadastra funcionários e altera perfis.

### RN14 — Administrador
- O Administrador é quem cadastra os colaboradores: o RH ou o dono da oficina.
- Na primeira inicialização do sistema, o primeiro Administrador é criado automaticamente.
- Tem acesso a todos os papéis de funcionário.
- A credencial inicial é definida na instalação, nunca fixa no código.

### RN15 — Acesso do cliente
O cliente tem acesso fechado: acompanha o fluxo da OS, aprova ou reprova o orçamento. Nada além disso. Nenhum funcionário, incluindo o Administrador, aprova ou reprova no lugar do cliente.

## 5. Pendências

| # | Pergunta | Impacta |
|---|---|---|
| P3 | A isenção acontece só na rejeição ("não conseguimos resolver") ou também em outros momentos? | RN05, RN08 |

### Pendências resolvidas

| # | Decisão | Regra |
|---|---|---|
| P4 | Funcionário cadastra o cliente no balcão; cliente ativa a conta no app com CPF + código de primeiro acesso e define a senha | RN11 |
| P5 | Funcionário segue o mesmo fluxo: gerente cadastra, funcionário ativa com CPF + código e define a senha; login com CPF + senha | RN13 |
| P6 | Administrador é o RH ou o dono da oficina, cadastra colaboradores; cliente tem acesso fechado (acompanha, aprova, reprova) | RN14, RN15 |
| P1 | Desconto incide sobre serviços + peças ou apenas sobre a mão de obra | RN07 |
| P2 | Teto de 10%; aplicado só com orçamento fechado e autorizado com a senha do gerente | RN07 |
