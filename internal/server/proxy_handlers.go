package server

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type bufferPool struct {
	pool sync.Pool
}

func (b *bufferPool) Get() []byte  { return b.pool.Get().([]byte) }
func (b *bufferPool) Put(buf []byte) { b.pool.Put(buf) }

var pool = &bufferPool{
	pool: sync.Pool{
		New: func() any { return make([]byte, 32*1024) },
	},
}

var transport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:        1024,
	MaxIdleConnsPerHost: 256,
	MaxConnsPerHost:     0,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  true,
	ForceAttemptHTTP2:   true,
	ReadBufferSize:      32 * 1024,
	WriteBufferSize:     32 * 1024,
}

func NewProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.BufferPool = pool
	return proxy
}

func ProxyRequestHandler(proxy *httputil.ReverseProxy, url *url.URL, endpoint string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = url.Host
		r.URL.Scheme = url.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = url.Host

		r.URL.Path = strings.TrimPrefix(r.URL.Path, endpoint)

		proxy.ServeHTTP(w, r)
	}
}
