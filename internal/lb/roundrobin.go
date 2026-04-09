package lb

import (
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
}

type RoundRobin struct {
	backends []Backend
	counter  atomic.Uint64
}

func NewRoundRobin(backends []Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

func (rr *RoundRobin) Next() Backend {
	n := rr.counter.Add(1)
	return rr.backends[n%uint64(len(rr.backends))]
}

func (rr *RoundRobin) Len() int {
	return len(rr.backends)
}
