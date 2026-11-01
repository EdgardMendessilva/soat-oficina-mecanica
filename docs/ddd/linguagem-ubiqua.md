# Linguagem ubíqua e glossário

Os termos abaixo são os mesmos nas conversas com o negócio, na documentação, na API e no código. Onde o enunciado do Tech Challenge usa outra palavra, a tabela de equivalências no fim mostra a correspondência.

## Glossário do domínio

| Termo | Significado no negócio | No código |
|---|---|---|
| **Ordem de Serviço (OS)** | Registro de tudo o que acontece com um veículo na oficina, da chegada à entrega. Agregado raiz do contexto core. | `ordemservico.OrdemDeServico` |
| **Status** | Etapa da OS: Recebida, Em diagnóstico, Aguardando aprovação, Em execução, Finalizada, Entregue. Só muda pelas ações do sistema e não pula etapa (RN03). | `ordemservico.Status` |
| **Diagnóstico** | Avaliação do mecânico que define o que precisa ser feito. É o único momento em que serviços e peças podem ser incluídos. | `IniciarDiagnostico`, `ErrItensSoNoDiagnostico` |
| **Serviço** | Trabalho de mão de obra incluído na OS, com descrição e preço. | `ordemservico.Servico` |
| **Peça** | Item físico do estoque usado na OS, com preço unitário e quantidade. | `ordemservico.Peca`, `estoque.Peca` |
| **Insumo** | Material de consumo da oficina (óleo, lubrificante) controlado no estoque como a peça. | `estoque.Insumo` |
| **Mão de obra** | Soma dos serviços da OS. | `MaoDeObra()` |
| **Orçamento** | Cálculo do que a OS custa: serviços + peças × quantidade, refeito a cada mudança (RN04). | `TotalDoOrcamento()` |
| **Valor do orçamento** | Taxa que o cliente paga pela avaliação quando **reprova** o orçamento. Se aprova, é isento (RN05, RN08). | `valorDoOrcamento` |
| **Decisão** | Resposta do cliente ao orçamento: Pendente, Aprovado ou Reprovado. Só o cliente dono decide (RN06). | `ordemservico.Decisao` |
| **Aprovar** | O cliente concorda com o orçamento. A OS vai para Em execução e as peças saem do estoque, na mesma transação (RN09). | `Aprovar`, `AprovarOrcamento` |
| **Reprovar** | O cliente recusa o orçamento. A OS vai direto para Finalizada, sem execução e sem baixa de estoque. | `Reprovar`, `ReprovarOrcamento` |
| **Valor a pagar** | O que o cliente deve no fim: aprovado paga serviços + peças menos o desconto; reprovado paga só o valor do orçamento. | `ValorAPagar()` |
| **Desconto** | Redução de até 10% concedida só com orçamento aprovado, incidindo sobre o total ou só sobre a mão de obra, autorizada pelo gerente com a própria senha (RN07). | `ordemservico.Desconto` |
| **Pontos-base** | Unidade do percentual: 100 pontos-base = 1%, o teto de 10% é 1000. | `TetoDescontoPontosBase` |
| **Isenção** | Perdão do valor do orçamento quando o cliente aprova. Automática (RN08). | `ValorAPagar()` |
| **Tempo médio de execução** | Média do tempo que as OS ficaram Em execução. | `TempoDeExecucao()`, `TempoMedioDeExecucao` |
| **Cliente** | Pessoa ou empresa dona do veículo, identificada por CPF ou CNPJ. Acesso fechado: acompanha e decide (RN15). | perfil `identidade.Cliente` |
| **Veículo** | Automóvel do cliente, identificado pela placa. | contexto Cadastro |
| **Funcionário** | Quem trabalha na oficina: atendente, mecânico, gerente ou administrador. | `identidade.Usuario` |
| **Perfil** | Papel de acesso: Cliente, Atendente, Mecânico, Gerente, Administrador. | `identidade.Perfil` |
| **Estoque** | Saldo de peças e insumos. Nunca fica negativo (RN09). | `estoque.Item` |
| **Reposição** | Entrada de unidades no estoque. | `Repor` |
| **Baixa** | Saída de unidades do estoque, feita na aprovação do orçamento. | `Baixar` |
| **Documento** | CPF (11 dígitos) ou CNPJ (14 dígitos) com dígitos verificadores válidos. | `valueobject.Documento` |
| **Placa** | Placa do veículo, padrão antigo (ABC1234) ou Mercosul (ABC1D23). | `valueobject.Placa` |

## Equivalência com o enunciado

| O enunciado diz | Neste projeto | Motivo |
|---|---|---|
| rejeição do orçamento | **reprovação** | termo do negócio da oficina |
| marca do veículo | **fabricante** | termo do negócio da oficina |
| colaborador | **funcionário** | termo do negócio da oficina |
| peças e insumos | **itens de estoque** do tipo Peça ou Insumo | os dois seguem o mesmo controle de saldo |

## Termos que não usamos

Rejeitar/rejeição, colaborador e marca (no sentido de fabricante). Usar sempre o termo da coluna "Neste projeto".
