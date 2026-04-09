package bench

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pgaijin66/tinyrp/internal/lb"
)

// shared tuned transport (mirrors the real proxy)
var benchTransport = &http.Transport{
	MaxIdleConns:        1024,
	MaxIdleConnsPerHost: 256,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  true,
	ForceAttemptHTTP2:   true,
	ReadBufferSize:      32 * 1024,
	WriteBufferSize:     32 * 1024,
}

type bufPool struct{ pool sync.Pool }

func (b *bufPool) Get() []byte  { return b.pool.Get().([]byte) }
func (b *bufPool) Put(buf []byte) { b.pool.Put(buf) }

var sharedPool = &bufPool{
	pool: sync.Pool{New: func() any { return make([]byte, 32*1024) }},
}

// backend returns a minimal test server that responds with a small body
func newBackend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	}))
}

func newLargeBackend(size int) *httptest.Server {
	body := make([]byte, size)
	for i := range body {
		body[i] = 'x'
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(body)
	}))
}

func buildProxy(backendURL string) *httputil.ReverseProxy {
	u, _ := url.Parse(backendURL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = benchTransport
	proxy.BufferPool = sharedPool
	return proxy
}

func buildRouter(backendURL string) http.Handler {
	u, _ := url.Parse(backendURL)
	proxy := buildProxy(backendURL)

	router := httprouter.New()
	router.Handle("GET", "/proxy/*path", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.URL.Host = u.Host
		r.URL.Scheme = u.Scheme
		r.Host = u.Host
		r.URL.Path = ps.ByName("path")
		proxy.ServeHTTP(w, r)
	})
	return router
}

// BenchmarkProxySmallBody measures throughput for small responses
func BenchmarkProxySmallBody(b *testing.B) {
	backend := newBackend()
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	client := proxy.Client()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(proxy.URL + "/proxy/test")
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkProxyLargeBody measures throughput for 1MB responses
func BenchmarkProxyLargeBody(b *testing.B) {
	backend := newLargeBackend(1024 * 1024)
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	client := proxy.Client()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(proxy.URL + "/proxy/test")
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkRoundRobin measures the load balancer selection overhead
func BenchmarkRoundRobin(b *testing.B) {
	backends := make([]lb.Backend, 4)
	for i := range backends {
		u, _ := url.Parse("http://localhost:9000")
		backends[i] = lb.Backend{URL: u}
	}
	rr := lb.NewRoundRobin(backends)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr.Next()
		}
	})
}

// BenchmarkProxyLatency measures single request p50/p99 latency
func BenchmarkProxyLatency(b *testing.B) {
	backend := newBackend()
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	client := proxy.Client()
	b.ResetTimer()

	for b.Loop() {
		start := time.Now()
		resp, err := client.Get(proxy.URL + "/proxy/test")
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		b.ReportMetric(float64(time.Since(start).Microseconds()), "us/req")
	}
}

// BenchmarkDirectVsProxy compares direct backend access to proxied access
func BenchmarkDirectVsProxy(b *testing.B) {
	backend := newBackend()
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	b.Run("direct", func(b *testing.B) {
		client := backend.Client()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(backend.URL + "/test")
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})

	b.Run("proxied", func(b *testing.B) {
		client := proxy.Client()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(proxy.URL + "/proxy/test")
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})
}
