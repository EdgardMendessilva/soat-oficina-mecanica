# Relação entre a documentação e o código

Onde cada conceito do DDD está no repositório, e o que ainda não foi implementado.

## Camadas

| Camada | Pasta |
|---|---|
| Interfaces (HTTP, JWT, Swagger) | `internal/interfaces/http` |
| Aplicação (casos de uso e portas) | `internal/application` |
| Domínio | `internal/domain/*` |
| Infraestrutura | `internal/infrastructure/postgres`, `memoria`, `seguranca` |
| Ponto de entrada | `cmd/api/main.go` |

## Contextos delimitados

| Contexto | Pacote do domínio | Casos de uso | Rotas |
|---|---|---|---|
| Ordem de Serviço (core) | `internal/domain/ordemservico` | `ordem_servico_service.go` | `/ordens/...` |
| Cadastro | `internal/domain/cadastro` | `cadastro_service.go` | `/clientes`, `/veiculos` |
| Catálogo | `internal/domain/catalogo` | `catalogo_service.go` | `/servicos` |
| Estoque | `internal/domain/estoque` | `estoque_service.go` | `/estoque` |
| Identidade e Acesso | `internal/domain/identidade` | `autenticacao_service.go` | `/auth/login` |

## Agregados, entidades e objetos de valor

| Conceito | Tipo | Arquivo |
|---|---|---|
| Ordem de Serviço | Agregado raiz | `internal/domain/ordemservico/ordem_de_servico.go` |
| Serviço e Peça da OS | Entidades internas | `internal/domain/ordemservico/itens.go` |
| Desconto | Entidade interna | `internal/domain/ordemservico/desconto.go` |
| Mudança de status | Entidade interna (histórico) | `internal/domain/ordemservico/ordem_de_servico.go` |
| Status e transições | Objeto de valor | `internal/domain/ordemservico/status.go` |
| Decisão | Objeto de valor | `internal/domain/ordemservico/ordem_de_servico.go` |
| Item de Estoque | Agregado raiz | `internal/domain/estoque/item.go` |
| Cliente | Agregado raiz | `internal/domain/cadastro/cliente.go` |
| Veículo | Agregado raiz | `internal/domain/cadastro/veiculo.go` |
| Serviço do catálogo | Agregado raiz | `internal/domain/catalogo/servico.go` |
| Usuário e Perfil | Agregado raiz e objeto de valor | `internal/domain/identidade/usuario.go` |
| Documento (CPF ou CNPJ) | Objeto de valor | `internal/domain/valueobject/documento.go` |
| Placa | Objeto de valor | `internal/domain/valueobject/placa.go` |
| Dinheiro (centavos) | Objeto de valor | `internal/domain/valueobject/dinheiro.go` |
| Quantidade | Objeto de valor | `internal/domain/valueobject/quantidade.go` |

## Regras de negócio e onde são garantidas

| Regra | Onde |
|---|---|
| RN01 CPF ou CNPJ válido e único | `valueobject.NovoDocumento`; unicidade em `CadastroService.CadastrarCliente` e `UNIQUE` no banco |
| RN02 Placa válida e única | `valueobject.NovaPlaca`; unicidade em `CadastroService.CadastrarVeiculo` e `UNIQUE` no banco |
| RN03 Fluxo de status | `ordemservico/status.go` (mapa de transições) |
| RN04 Orçamento automático | `TotalDoOrcamento()` |
| RN05 e RN08 Cobrança e isenção | `ValorAPagar()` |
| RN06 e RN15 Só o cliente dono decide | `Aprovar` e `Reprovar` (`ErrSomenteDonoDecide`); rotas exigem perfil Cliente |
| RN07 Desconto | `AplicarDesconto` (teto, base, orçamento aprovado); senha em `OrdemDeServicoService.AplicarDesconto` |
| RN09 Baixa de estoque na aprovação | `AprovarOrcamento` numa transação; `SELECT ... FOR UPDATE` em `infrastructure/postgres/estoque.go` e `CHECK (saldo >= 0)` |
| RN10 Isolamento do cliente | `ConsultarDoCliente`, `ListarDoCliente` e as rotas de cliente e veículo |
| RN12 Política de senha | `identidade.ValidarSenha`; hash PBKDF2 em `infrastructure/seguranca/senha.go` |
| RN14 Primeiro Administrador | `memoria.UsuariosIniciais` (variáveis `ADMIN_DOCUMENTO` e `ADMIN_SENHA`) |

## Eventos de domínio

O MVP não tem barramento de eventos. Os eventos do Event Storming aparecem no código assim:

| Evento | Registro no código |
|---|---|
| Mudanças de status (aberta, diagnóstico, enviado, em execução, finalizada, entregue) | histórico da OS (`Historico()`, tabela `os_historico`) |
| Orçamento aprovado ou reprovado | `Decisao`, quem decidiu e quando (`DecididoPor`, `DecididoEm`) |
| Desconto concedido | `Desconto` com quem autorizou, quando, percentual e base |
| Estoque baixado ou reposto | saldo do item de estoque |

## O que o Event Storming prevê e o MVP não implementa

| Item | Situação |
|---|---|
| Primeiro acesso por e-mail com link de uso único (RN11, RN13) | Modelado e documentado; não implementado. Usuários vêm da inicialização (administrador e demonstração) |
| Cadastro de funcionários pela API (RN13, RN14) | Não implementado; o primeiro Administrador e os usuários de demonstração nascem na inicialização |
| Exclusão de item de estoque | Não implementada; há cadastro, consulta e reposição |
| Isenção manual pelo gerente (P7) | Fora do MVP, pendência aberta nas regras |
