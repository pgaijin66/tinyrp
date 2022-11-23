package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/pgaijin66/lightweight-reverse-proxy/internal/configs"
)

// ping returns a "pong" message
func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// Run starts server and listens on defined port
func Run() error {
	// load configurations from config file
	config, err := configs.NewConfiguration()
	if err != nil {
		log.Fatalf("could not load configuration: %v", err)
	}

	// creating a new router
	router := http.NewServeMux()

	// health check
	router.HandleFunc("/ping", ping)

	// Iterating over the hosts
	for _, resources := range configs.Config.Resources {
		URL, err := url.Parse(configs.Config.Resources[0].Destination_URL)
		if err != nil {
			log.Fatal(err)
		}
		proxy, _ := NewProxy(URL)
		router.HandleFunc(resources.Endpoint, ProxyRequestHandler(proxy))
	}

	fmt.Println("Starting and Listening server at port: ", config.Server.Listen_port)
	switch config.Server.Scheme {
	case "http":
		http.ListenAndServe(config.Server.Host+":"+config.Server.Listen_port, router)
	default:
		return fmt.Errorf("server scheme not supported")
	}
	return nil
}
