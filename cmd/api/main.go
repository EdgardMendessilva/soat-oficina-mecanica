// Command api inicia o servidor HTTP do Sistema Integrado da Oficina Mecânica.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/application"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/memoria"
	"github.com/EdgardMendessilva/soat-oficina-mecanica/internal/infrastructure/postgres"
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

	var (
		ordens   application.OrdemRepositorio
		itens    application.EstoqueRepositorio
		clientes application.ClienteRepositorio
		veiculos application.VeiculoRepositorio
		servicos application.ServicoRepositorio
		tx       application.Transacao
		usuarios interface {
			application.UsuarioRepositorio
			application.AutorizadorGerente
		}
	)
	adminDoc, adminSenha, demo := os.Getenv("ADMIN_DOCUMENTO"), os.Getenv("ADMIN_SENHA"), os.Getenv("SEED_DEMO") == "true"

	if url := os.Getenv("DATABASE_URL"); url != "" {
		ctx := context.Background()
		db, err := postgres.Abrir(ctx, url)
		if err != nil {
			log.Fatal(err)
		}
		u := postgres.NovosUsuarios(db)
		if err := postgres.Semear(ctx, u, adminDoc, adminSenha, demo); err != nil {
			log.Fatal(err)
		}
		ordens, itens, tx, usuarios = postgres.NovasOrdens(db), postgres.NovoEstoque(db), postgres.NovaTransacao(db), u
		clientes, veiculos, servicos = postgres.NovosClientes(db), postgres.NovosVeiculos(db), postgres.NovosServicos(db)
		log.Print("persistência: PostgreSQL")
	} else {
		u := memoria.NovosUsuarios()
		if err := memoria.Semear(u, adminDoc, adminSenha, demo); err != nil {
			log.Fatal(err)
		}
		ordens, itens, tx, usuarios = memoria.NovasOrdens(), memoria.NovoEstoque(), &memoria.Transacao{}, u
		clientes, veiculos, servicos = memoria.NovosClientes(), memoria.NovosVeiculos(), memoria.NovosServicos()
		log.Print("persistência: memória (defina DATABASE_URL para usar o PostgreSQL)")
	}

	if demo {
		if err := memoria.SemearCadastro(context.Background(), clientes, veiculos); err != nil {
			log.Fatal(err)
		}
	}

	cadastroSvc := application.NovoCadastroService(clientes, veiculos, tx)
	srv := apihttp.NovoServidor(
		application.NovoOrdemDeServicoService(ordens, itens, tx, usuarios, relogio{}).ComCadastro(cadastroSvc),
		application.NovoEstoqueService(itens, tx),
		application.NovoAutenticacaoService(usuarios, seguranca.Verificador{}),
		cadastroSvc,
		application.NovoCatalogoService(servicos, tx),
		jwt,
	)

	log.Printf("api ouvindo na porta %s (Swagger em /swagger)", porta)
	s := &http.Server{Addr: ":" + porta, Handler: srv.Rotas(), ReadHeaderTimeout: 5 * time.Second}
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
