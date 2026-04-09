package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pgaijin66/tinyrp/internal/cb"
	"github.com/pgaijin66/tinyrp/internal/configs"
	"github.com/pgaijin66/tinyrp/internal/lb"
	"github.com/pgaijin66/tinyrp/internal/retry"
)

func Run() error {
	config, err := configs.Load("data/config.yaml")
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	for _, resource := range config.Resources {
		backends, err := buildBackends(resource.Backends())
		if err != nil {
			return fmt.Errorf("resource %q: %w", resource.Name, err)
		}
		rr := lb.NewRoundRobin(backends)
		// Go 1.22+ catch-all: /server1/ matches /server1/anything
		pattern := resource.Endpoint + "/"
		endpoint := resource.Endpoint
		mux.HandleFunc(pattern, makeHandler(rr, endpoint))
	}

	addr := config.Server.Host + ":" + config.Server.ListenPort
	ln, err := newReusePortListener(addr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

func buildBackends(urls []string) ([]lb.Backend, error) {
	backends := make([]lb.Backend, 0, len(urls))
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		backends = append(backends, lb.Backend{
			URL: u,
			CB:  cb.New(5, 10*time.Second),
		})
	}
	return backends, nil
}

func makeHandler(rr *lb.RoundRobin, endpoint string) http.HandlerFunc {
	rc := retry.Default

	return func(w http.ResponseWriter, r *http.Request) {
		origHost := r.Host
		path := strings.TrimPrefix(r.URL.Path, endpoint)
		r.URL.Path = path

		for attempt := 0; attempt < rc.MaxAttempts; attempt++ {
			if attempt > 0 {
				time.Sleep(rc.Delay(attempt - 1))
			}

			backend := rr.Next()
			r.Host = origHost

			status := proxyDo(w, r, backend.URL.Host, backend.URL.Scheme)

			if backend.CB != nil {
				if status >= 500 {
					backend.CB.RecordFailure()
				} else {
					backend.CB.RecordSuccess()
				}
			}

			if (status == 502 || status == 503 || status == 504) && rr.Len() > 1 {
				continue
			}
			return
		}
	}
}
