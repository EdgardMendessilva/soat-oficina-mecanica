package seguranca

import (
	"strings"
	"testing"
	"time"
)

func TestJWTEmiteEValida(t *testing.T) {
	agora := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	e, err := NovoEmissorJWT("segredo-de-teste-123456", time.Hour, func() time.Time { return agora })
	if err != nil {
		t.Fatal(err)
	}
	tok, _, _ := e.Emitir(Claims{Sub: "u1", Perfil: "Gerente"})
	c, err := e.Validar(tok)
	if err != nil || c.Sub != "u1" || c.Perfil != "Gerente" {
		t.Fatalf("validação falhou: %v %+v", err, c)
	}
}

func TestJWTRecusaAdulteradoExpiradoELixo(t *testing.T) {
	agora := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	e, _ := NovoEmissorJWT("segredo-de-teste-123456", time.Hour, func() time.Time { return agora })
	tok, _, _ := e.Emitir(Claims{Sub: "u1", Perfil: "Cliente"})

	p := strings.Split(tok, ".")
	if _, err := e.Validar(p[0] + "." + p[1] + "x." + p[2]); err == nil {
		t.Error("aceitou corpo adulterado")
	}
	outro, _ := NovoEmissorJWT("outro-segredo-123456789", time.Hour, func() time.Time { return agora })
	if _, err := outro.Validar(tok); err == nil {
		t.Error("aceitou assinatura de outro segredo")
	}
	depois, _ := NovoEmissorJWT("segredo-de-teste-123456", time.Hour, func() time.Time { return agora.Add(2 * time.Hour) })
	if _, err := depois.Validar(tok); err == nil {
		t.Error("aceitou token expirado")
	}
	if _, err := e.Validar("lixo"); err == nil {
		t.Error("aceitou lixo")
	}
	if _, err := NovoEmissorJWT("curto", time.Hour, nil); err == nil {
		t.Error("aceitou segredo curto")
	}
}

func TestSenha(t *testing.T) {
	h, err := HashSenha("Teste@1234")
	if err != nil || strings.Contains(h, "Teste@1234") {
		t.Fatalf("hash ruim: %v %q", err, h)
	}
	if !ConferirSenha("Teste@1234", h) || ConferirSenha("Errada@123", h) || ConferirSenha("x", "formato-ruim") {
		t.Error("conferência errada")
	}
	h2, _ := HashSenha("Teste@1234")
	if h == h2 {
		t.Error("o sal deveria tornar os hashes diferentes")
	}
}
