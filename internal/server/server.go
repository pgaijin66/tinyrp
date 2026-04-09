package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pgaijin66/tinyrp/internal/configs"
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
		u, err := url.Parse(resource.DestinationURL)
		if err != nil {
			return fmt.Errorf("invalid destination URL %q: %w", resource.DestinationURL, err)
		}
		proxy := NewProxy(u)
		registerRoute(router, resource.Endpoint, proxy, u)
	}

	srv := &http.Server{
		Addr:              config.Server.Host + ":" + config.Server.ListenPort,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
	return srv.ListenAndServe()
}

func registerRoute(router *httprouter.Router, endpoint string, proxy *httputil.ReverseProxy, target *url.URL) {
	handler := makeHandler(proxy, target, endpoint)
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range methods {
		router.Handle(m, endpoint+"/*path", handler)
	}
}

func makeHandler(proxy *httputil.ReverseProxy, target *url.URL, endpoint string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = target.Host
		r.URL.Path = ps.ByName("path")
		proxy.ServeHTTP(w, r)
	}
}
