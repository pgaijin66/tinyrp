package server

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

var transport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          2048,
	MaxIdleConnsPerHost:   512,
	MaxConnsPerHost:       0,
	IdleConnTimeout:       90 * time.Second,
	DisableCompression:    true,
	ForceAttemptHTTP2:     true,
	ReadBufferSize:        64 * 1024,
	WriteBufferSize:       64 * 1024,
	ExpectContinueTimeout: 0,
	DisableKeepAlives:     false,
}

// buffer pool: 64KB buffers reused across requests
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
		return &b
	},
}

// copyHeaders writes src headers into dst without allocating a new map.
// This avoids the Header.Clone() that httputil.ReverseProxy does.
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// proxyDo is a hand-rolled reverse proxy that avoids the allocations
// in httputil.ReverseProxy:
//   - no Request.Clone (reuses the inbound request directly)
//   - no Header.Clone (copies headers field by field)
//   - uses pooled buffers for io.Copy
//   - single shared transport with aggressive connection reuse
func proxyDo(w http.ResponseWriter, r *http.Request, targetHost, targetScheme string) int {
	outURL := *r.URL
	outURL.Host = targetHost
	outURL.Scheme = targetScheme

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, outURL.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return 400
	}
	copyHeaders(outReq.Header, r.Header)
	outReq.Header.Set("X-Forwarded-Host", r.Host)
	outReq.Host = targetHost
	outReq.ContentLength = r.ContentLength

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return 502
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	bufp := bufPool.Get().(*[]byte)
	io.CopyBuffer(w, resp.Body, *bufp)
	bufPool.Put(bufp)

	return resp.StatusCode
}
