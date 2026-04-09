package main

import (
	"log"
	"runtime"

	"github.com/pgaijin66/tinyrp/internal/server"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := server.Run(); err != nil {
		log.Fatalf("could not start the server: %v", err)
	}
}
