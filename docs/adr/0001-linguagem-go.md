# ADR 0001 — Linguagem: Go

- **Status:** Aceito
- **Data:** 2026-10-16

## Contexto

A oficina precisa de um back-end (MVP) que organize atendimento, ordens de serviço, orçamento, estoque e acompanhamento pelo cliente. O enunciado pede API REST, monólito em camadas, Docker e cobertura mínima de 80% nos domínios críticos. O sistema será evoluído nas próximas fases da pós (containers, Kubernetes, microsserviços, serverless).

## Decisão

Usar **Go** como linguagem do back-end.

## Justificativa

- Gera um binário único e estático: imagem Docker pequena e boa base para Kubernetes nas próximas fases.
- A biblioteca padrão cobre HTTP, JSON e testes sem framework pesado, o que deixa as camadas explícitas.
- `go test` mede cobertura por pacote, facilitando provar os 80% nas camadas de domínio e aplicação.
- Interfaces implícitas favorecem inversão de dependência: o domínio não conhece banco nem HTTP.
- Ferramentas de segurança do próprio ecossistema: `govulncheck` e `gosec`.
- Experiência prévia do autor com Go.

## Alternativas consideradas

- **Java/Spring Boot:** muito usado no mercado, mas traz mais configuração e "mágica" de framework, o que esconde as camadas.
- **Node.js/TypeScript:** produtivo, porém exige mais disciplina para tipagem forte no domínio e mais dependências externas.

## Consequências

- Mais código explícito de inicialização e injeção de dependências, compensado pela clareza das camadas.
- Bibliotecas externas entram só quando necessárias e cada uma é avaliada no scan de vulnerabilidades.
