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
	outReq := &http.Request{
		Method:        r.Method,
		URL:           &url.URL{
			Scheme:   targetScheme,
			Host:     targetHost,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header, len(r.Header)),
		Body:          r.Body,
		ContentLength: r.ContentLength,
		Host:          targetHost,
	}
	outReq = outReq.WithContext(r.Context())
	copyHeaders(outReq.Header, r.Header)
	outReq.Header.Set("X-Forwarded-Host", r.Host)

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
