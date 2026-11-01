package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/domain/identidade"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
)

type chaveCtx struct{}

func usuarioDe(r *http.Request) seguranca.Claims {
	c, _ := r.Context().Value(chaveCtx{}).(seguranca.Claims)
	return c
}

// exigir protege a rota: precisa de token válido e de um dos perfis informados.
// Sem perfis, qualquer usuário autenticado entra.
func (s *Servidor) exigir(h http.HandlerFunc, perfis ...identidade.Perfil) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cab := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(cab, "Bearer ")
		if !ok || token == "" {
			escreverErro(w, http.StatusUnauthorized, "nao_autenticado", "envie o token no cabeçalho Authorization: Bearer <token>")
			return
		}
		claims, err := s.jwt.Validar(token)
		if err != nil {
			escreverErro(w, http.StatusUnauthorized, "nao_autenticado", err.Error())
			return
		}
		if len(perfis) > 0 && !identidade.Perfil(claims.Perfil).EstaEntre(perfis...) {
			escreverErro(w, http.StatusForbidden, "acesso_negado", "o perfil "+claims.Perfil+" não pode executar esta ação")
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), chaveCtx{}, claims)))
	}
}
