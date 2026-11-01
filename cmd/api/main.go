// Command api inicia o servidor HTTP do Sistema Integrado da Oficina Mecânica.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	porta := os.Getenv("PORT")
	if porta == "" {
		porta = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	})

	log.Printf("api ouvindo na porta %s", porta)
	if err := http.ListenAndServe(":"+porta, mux); err != nil {
		log.Fatal(err)
	}
}
