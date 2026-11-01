package seguranca

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

const iteracoes = 210000

// HashSenha gera "pbkdf2$<sal>$<hash>"; a senha nunca é guardada em texto (RN12).
func HashSenha(senha string) (string, error) {
	sal := make([]byte, 16)
	if _, err := rand.Read(sal); err != nil {
		return "", err
	}
	h, err := pbkdf2.Key(sha256.New, senha, sal, iteracoes, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2$%s$%s", base64.RawStdEncoding.EncodeToString(sal), base64.RawStdEncoding.EncodeToString(h)), nil
}

// ConferirSenha compara a senha com o hash guardado, em tempo constante.
func ConferirSenha(senha, guardado string) bool {
	p := strings.Split(guardado, "$")
	if len(p) != 3 || p[0] != "pbkdf2" {
		return false
	}
	sal, err1 := base64.RawStdEncoding.DecodeString(p[1])
	esperado, err2 := base64.RawStdEncoding.DecodeString(p[2])
	if err1 != nil || err2 != nil {
		return false
	}
	h, err := pbkdf2.Key(sha256.New, senha, sal, iteracoes, 32)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(h, esperado) == 1
}

// Verificador adapta ConferirSenha à porta application.VerificadorSenha.
type Verificador struct{}

// Conferir compara a senha com o hash.
func (Verificador) Conferir(senha, hash string) bool { return ConferirSenha(senha, hash) }
