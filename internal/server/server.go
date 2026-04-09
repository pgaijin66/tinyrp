package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pgaijin66/tinyrp/internal/configs"
)

func Run() error {
	config, err := configs.Load("data/config.yaml")
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)

	for _, resource := range config.Resources {
		url, err := url.Parse(resource.DestinationURL)
		if err != nil {
			return fmt.Errorf("invalid destination URL %q: %w", resource.DestinationURL, err)
		}
		proxy := NewProxy(url)
		mux.HandleFunc(resource.Endpoint, ProxyRequestHandler(proxy, url, resource.Endpoint))
	}

	addr := config.Server.Host + ":" + config.Server.ListenPort
	return http.ListenAndServe(addr, mux)
}
