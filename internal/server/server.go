package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
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

	addr := config.Server.Host + ":" + config.Server.ListenPort
	ln, err := newReusePortListener(addr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	srv := &http.Server{
		Handler:           router,
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
			URL:   u,
			Proxy: NewProxy(u),
			CB:    cb.New(5, 10*time.Second),
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func makeHandler(rr *lb.RoundRobin) httprouter.Handle {
	rc := retry.Default

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		path := ps.ByName("path")
		origHost := r.Header.Get("Host")

		for attempt := 0; attempt < rc.MaxAttempts; attempt++ {
			if attempt > 0 {
				time.Sleep(rc.Delay(attempt - 1))
			}

			backend := rr.Next()
			r.URL.Host = backend.URL.Host
			r.URL.Scheme = backend.URL.Scheme
			r.Header.Set("X-Forwarded-Host", origHost)
			r.Host = backend.URL.Host
			r.URL.Path = path

			rec := &statusRecorder{ResponseWriter: w, status: 200}
			backend.Proxy.ServeHTTP(rec, r)

			if backend.CB != nil {
				if rec.status >= 500 {
					backend.CB.RecordFailure()
				} else {
					backend.CB.RecordSuccess()
				}
			}

			// only retry on gateway errors and only if we have multiple backends
			if rec.status == 502 || rec.status == 503 || rec.status == 504 {
				if rr.Len() > 1 {
					continue
				}
			}
			return
		}
	}
}
