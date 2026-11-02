# Relatório de análise de vulnerabilidades

Projeto: soat-oficina-mecanica (Tech Challenge, Fase 1, SOAT)
Data da análise: 02/11/2026

## 1. Resumo

| Item | Resultado |
|---|---|
| Ferramenta | `govulncheck` v1.9.0 (ferramenta oficial do projeto Go, base de dados pkg.go.dev/vuln) |
| Alvo | Todo o código Go do repositório (`./...`) e suas dependências |
| Antes da correção (Go 1.26.5, commit `20702ca`) | **12 vulnerabilidades** que afetavam o código, todas na biblioteca padrão do Go |
| Correção | Atualização do Go para **1.26.10** (`go.mod` e `Dockerfile`) |
| Depois da correção (Go 1.26.10, commit `df5129a`) | **Nenhuma vulnerabilidade encontrada** |

## 2. Como a análise foi feita

```powershell
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

O `govulncheck` analisa o grafo de chamadas do código e só reporta como "afetam o código" as falhas em funções que a aplicação realmente chama, direta ou indiretamente. Isso reduz falsos positivos em relação a ferramentas que apenas listam versões de dependências.

Evidências brutas (saídas completas do comando):

- [`govulncheck-antes.txt`](govulncheck-antes.txt)
- [`govulncheck-depois.txt`](govulncheck-depois.txt)

## 3. Vulnerabilidades encontradas (antes da correção)

Todas estavam na biblioteca padrão do Go, versão 1.26.5. Nenhuma estava no código escrito pelo projeto.

| ID | Pacote | Descrição | Corrigida em |
|---|---|---|---|
| GO-2026-6809 | crypto/x509 | Certificados intermediários comprometidos deixam de ser aceitos | 1.26.10 |
| GO-2026-6617 | net/http | Queda do servidor HTTP/2 por condição de corrida no codificador HPACK | 1.26.9 |
| GO-2026-6613 | net/http | Dessincronização da conexão HTTP/1 após resposta 2xx a CONNECT | 1.26.9 |
| GO-2026-6612 | net/http | Devolução duplicada de controle de fluxo em streams HTTP/2 | 1.26.9 |
| GO-2026-6611 | net/http | Consumo excessivo de CPU por mudanças repetidas da janela inicial | 1.26.9 |
| GO-2026-6608 | mime/multipart, net/textproto | Contorno do limite de memória ao processar cabeçalhos MIME | 1.26.9 |
| GO-2026-6607 | crypto/tls | Referências malformadas na extensão ECH não eram rejeitadas | 1.26.9 |
| GO-2026-6603 | net/http | Esgotamento de memória no servidor HTTP/2 por cabeçalhos Trailer | 1.26.9 |
| GO-2026-6090 | crypto/tls | Sem limite para mensagens de handshake após o handshake | 1.26.6 |
| GO-2026-6089 | net/http | ReadHeaderTimeout não aplicado na checagem de HTTP/2 sem criptografia | 1.26.6 |
| GO-2026-6088 | encoding/xml | Sem limite de profundidade de recursão na decodificação | 1.26.6 |
| GO-2026-5972 | encoding/asn1 | Sem limite máximo de recursão | 1.26.6 |

Origem do alcance: o servidor da API usa `http.Server.ListenAndServe` (`cmd/api/main.go`) e o acesso ao banco passa por `database/sql`, que levam às rotinas afetadas da biblioteca padrão.

Além dessas 12, o scan informou 7 vulnerabilidades em pacotes importados e 3 em módulos requeridos que o código **não chama**. Elas não afetam a aplicação e não exigem ação.

## 4. Causa

O projeto estava fixado no Go 1.26.5 (`go 1.26.5` no `go.mod`) e a imagem de build usava a tag flutuante `golang:1.26-alpine`. As falhas estavam na própria biblioteca padrão dessa versão e já tinham correção publicada em versões posteriores do Go.

## 5. Correção aplicada

Alteração mínima, sem mudança de código da aplicação (PR de atualização do Go, commit `df5129a`):

| Arquivo | Antes | Depois |
|---|---|---|
| `go.mod` | `go 1.26.5` | `go 1.26.10` |
| `Dockerfile` | `FROM golang:1.26-alpine AS build` | `FROM golang:1.26.10-alpine AS build` |

O workflow de CI (`.github/workflows/ci.yml`) usa `go-version-file: go.mod`, então também passa a compilar e testar com o Go 1.26.10. Fixar a versão exata na imagem torna o build reproduzível e impede a volta de uma versão vulnerável.

## 6. Resultado depois da correção

```
> govulncheck ./...
No vulnerabilities found.
```

## 7. Escopo e limites desta análise

- O `govulncheck` cobre o código Go e suas dependências. **Não** analisa a imagem Docker final (Alpine 3.20) nem o PostgreSQL.
- O contêiner final roda como usuário sem privilégios (`app`, UID 10001), em imagem mínima sem compilador.
- As credenciais ficam em variáveis de ambiente (`.env`, fora do repositório). O repositório contém apenas o `.env.example`.
- Como melhoria futura: incluir o `govulncheck` no CI e, se desejado, um scan de imagem (por exemplo, Trivy).
