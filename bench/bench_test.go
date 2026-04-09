package bench

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pgaijin66/tinyrp/internal/lb"
)

// shared tuned transport matching the real proxy
var benchTransport = &http.Transport{
	MaxIdleConns:        2048,
	MaxIdleConnsPerHost: 512,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  true,
	ForceAttemptHTTP2:   true,
	ReadBufferSize:      64 * 1024,
	WriteBufferSize:     64 * 1024,
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
		return &b
	},
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		dst[k] = vv
	}
}

func proxyDo(w http.ResponseWriter, r *http.Request, targetHost, targetScheme string) {
	origURL := r.URL
	origHost := r.Host

	r.URL = &url.URL{
		Scheme:   targetScheme,
		Host:     targetHost,
		Path:     origURL.Path,
		RawQuery: origURL.RawQuery,
	}
	r.Host = targetHost
	r.RequestURI = ""

	resp, err := benchTransport.RoundTrip(r)

	r.URL = origURL
	r.Host = origHost

	if err != nil {
		http.Error(w, "bad gateway", 502)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	bufp := bufPool.Get().(*[]byte)
	io.CopyBuffer(w, resp.Body, *bufp)
	bufPool.Put(bufp)
}

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

func buildRouter(backendURL string) http.Handler {
	u, _ := url.Parse(backendURL)
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/proxy")
		proxyDo(w, r, u.Host, u.Scheme)
	})
	return mux
}

func BenchmarkProxySmallBody(b *testing.B) {
	backend := newBackend()
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConnsPerHost: 256,
			DisableCompression:  true,
		},
	}
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

func BenchmarkProxyLargeBody(b *testing.B) {
	backend := newLargeBackend(1024 * 1024)
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConnsPerHost: 256,
			DisableCompression:  true,
		},
	}
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

func BenchmarkDirectVsProxy(b *testing.B) {
	backend := newBackend()
	defer backend.Close()
	proxy := httptest.NewServer(buildRouter(backend.URL))
	defer proxy.Close()

	b.Run("direct", func(b *testing.B) {
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext,
				MaxIdleConnsPerHost: 256,
				DisableCompression:  true,
			},
		}
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
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext,
				MaxIdleConnsPerHost: 256,
				DisableCompression:  true,
			},
		}
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
