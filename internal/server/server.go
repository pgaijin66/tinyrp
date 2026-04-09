package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pgaijin66/tinyrp/internal/configs"
	"github.com/pgaijin66/tinyrp/internal/lb"
)

func Run() error {
	config, err := configs.Load("data/config.yaml")
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}

	router := httprouter.New()
	router.GET("/ping", func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("pong"))
	})

	for _, resource := range config.Resources {
		backends, err := buildBackends(resource.Backends())
		if err != nil {
			return fmt.Errorf("resource %q: %w", resource.Name, err)
		}
		rr := lb.NewRoundRobin(backends)
		registerRoute(router, resource.Endpoint, rr)
	}

	srv := &http.Server{
		Addr:              config.Server.Host + ":" + config.Server.ListenPort,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return srv.ListenAndServe()
}

func buildBackends(urls []string) ([]lb.Backend, error) {
	backends := make([]lb.Backend, 0, len(urls))
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		backends = append(backends, lb.Backend{
			URL:   u,
			Proxy: NewProxy(u),
		})
	}
	return backends, nil
}

func registerRoute(router *httprouter.Router, endpoint string, rr *lb.RoundRobin) {
	handler := makeHandler(rr)
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range methods {
		router.Handle(m, endpoint+"/*path", handler)
	}
}

func makeHandler(rr *lb.RoundRobin) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		backend := rr.Next()
		r.URL.Host = backend.URL.Host
		r.URL.Scheme = backend.URL.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = backend.URL.Host
		r.URL.Path = ps.ByName("path")
		backend.Proxy.ServeHTTP(w, r)
	}
}
