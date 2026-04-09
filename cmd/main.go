package main

import (
	"log"
	"runtime"
	"runtime/debug"

	"github.com/pgaijin66/tinyrp/internal/server"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	debug.SetGCPercent(200)

	if err := server.Run(); err != nil {
		log.Fatalf("could not start the server: %v", err)
	}
}
