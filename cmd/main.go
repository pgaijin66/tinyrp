package main

import (
	"log"

	"github.com/pgaijin66/lightweight-reverse-proxy/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatalf("could not start the server: %v", err)
	}
}
