// Command api inicia o servidor HTTP do Sistema Integrado da Oficina Mecânica.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/seguranca"
	apihttp "github.com/EdgardMendessilva/soat-oficina-mecanica/internal/interfaces/http"
)

type relogio struct{}

func (relogio) Agora() time.Time { return time.Now() }

func main() {
	porta := os.Getenv("PORT")
	if porta == "" {
		porta = "8080"
	}

	jwt, err := seguranca.NovoEmissorJWT(os.Getenv("JWT_SECRET"), 8*time.Hour, nil)
	if err != nil {
		log.Fatal(err)
	}

	ordens, itens, tx, usuarios := memoria.NovasOrdens(), memoria.NovoEstoque(), &memoria.Transacao{}, memoria.NovosUsuarios()
	if err := memoria.Semear(usuarios, os.Getenv("ADMIN_DOCUMENTO"), os.Getenv("ADMIN_SENHA"), os.Getenv("SEED_DEMO") == "true"); err != nil {
		log.Fatal(err)
	}

	srv := apihttp.NovoServidor(
		application.NovoOrdemDeServicoService(ordens, itens, tx, usuarios, relogio{}),
		application.NovoEstoqueService(itens, tx),
		application.NovoAutenticacaoService(usuarios, seguranca.Verificador{}),
		jwt,
	)

	log.Printf("api ouvindo na porta %s (Swagger em /swagger)", porta)
	s := &http.Server{Addr: ":" + porta, Handler: srv.Rotas(), ReadHeaderTimeout: 5 * time.Second}
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
