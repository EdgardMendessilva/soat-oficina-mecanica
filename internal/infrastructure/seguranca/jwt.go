// Package seguranca reúne JWT (HS256) e hash de senha, só com a biblioteca padrão.
package seguranca

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrTokenInvalido cobre assinatura errada, formato quebrado e token expirado.
var ErrTokenInvalido = errors.New("token inválido ou expirado")

// Claims são os dados guardados no token.
type Claims struct {
	Sub       string `json:"sub"`
	Perfil    string `json:"perfil"`
	ClienteID string `json:"clienteId,omitempty"`
	Exp       int64  `json:"exp"`
}

// EmissorJWT assina e valida tokens HS256.
type EmissorJWT struct {
	segredo  []byte
	validade time.Duration
	agora    func() time.Time
}

// NovoEmissorJWT cria o emissor. O segredo vem do ambiente (JWT_SECRET).
func NovoEmissorJWT(segredo string, validade time.Duration, agora func() time.Time) (*EmissorJWT, error) {
	if len(segredo) < 16 {
		return nil, errors.New("JWT_SECRET precisa ter ao menos 16 caracteres")
	}
	if agora == nil {
		agora = time.Now
	}
	return &EmissorJWT{segredo: []byte(segredo), validade: validade, agora: agora}, nil
}

var cabecalho = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

func (e *EmissorJWT) assinar(dados string) string {
	m := hmac.New(sha256.New, e.segredo)
	m.Write([]byte(dados))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Emitir cria um token para o usuário.
func (e *EmissorJWT) Emitir(c Claims) (string, time.Time, error) {
	expira := e.agora().Add(e.validade)
	c.Exp = expira.Unix()
	corpo, err := json.Marshal(c)
	if err != nil {
		return "", time.Time{}, err
	}
	dados := cabecalho + "." + base64.RawURLEncoding.EncodeToString(corpo)
	return dados + "." + e.assinar(dados), expira, nil
}

// Validar confere assinatura e validade e devolve os dados do token.
func (e *EmissorJWT) Validar(token string) (Claims, error) {
	partes := strings.Split(token, ".")
	if len(partes) != 3 || partes[0] != cabecalho {
		return Claims{}, ErrTokenInvalido
	}
	esperada := e.assinar(partes[0] + "." + partes[1])
	if !hmac.Equal([]byte(esperada), []byte(partes[2])) {
		return Claims{}, ErrTokenInvalido
	}
	corpo, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return Claims{}, ErrTokenInvalido
	}
	var c Claims
	if err := json.Unmarshal(corpo, &c); err != nil || c.Sub == "" {
		return Claims{}, ErrTokenInvalido
	}
	if e.agora().Unix() >= c.Exp {
		return Claims{}, ErrTokenInvalido
	}
	return c, nil
}
