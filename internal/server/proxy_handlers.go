package server

import (
	"io"
	"net"
	"net/http"
	"net/url"
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
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
		return &b
	},
}

// copyHeaders writes src headers into dst without allocating a new map.
// Uses direct map assignment to avoid the overhead of Header.Add
// which canonicalizes the key on every call.
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		dst[k] = vv
	}
}

// proxyDo forwards the request to the target backend.
// Avoids httputil.ReverseProxy overhead by:
//   - constructing the outbound URL directly (no String() + re-parse)
//   - copying headers without cloning the map
//   - using pooled 64KB buffers for the body copy
func proxyDo(w http.ResponseWriter, r *http.Request, targetHost, targetScheme string) int {
	// Mutate the inbound request in-place for the outbound call.
	// This avoids allocating a new Request, URL, Header map, and context.
	// We restore the original values after RoundTrip returns.
	origURL := r.URL
	origHost := r.Host

	r.URL = &url.URL{
		Scheme:   targetScheme,
		Host:     targetHost,
		Path:     origURL.Path,
		RawQuery: origURL.RawQuery,
	}
	r.Host = targetHost
	r.Header.Set("X-Forwarded-Host", origHost)
	r.RequestURI = "" // required for client requests
	resp, err := transport.RoundTrip(r)

	// restore the original request state
	r.URL = origURL
	r.Host = origHost

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
