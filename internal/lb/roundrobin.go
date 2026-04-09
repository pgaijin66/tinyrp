package lb

import (
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/pgaijin66/tinyrp/internal/cb"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	CB    *cb.CircuitBreaker
}

type RoundRobin struct {
	backends []Backend
	counter  atomic.Uint64
}

func NewRoundRobin(backends []Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

// Next returns the next healthy backend, skipping open circuit breakers.
// Falls back to plain round-robin if all are tripped.
func (rr *RoundRobin) Next() Backend {
	total := uint64(len(rr.backends))
	start := rr.counter.Add(1)
	for i := uint64(0); i < total; i++ {
		b := rr.backends[(start+i)%total]
		if b.CB == nil || b.CB.Allow() {
			return b
		}
	}
	// all open — try anyway
	return rr.backends[start%total]
}

func (rr *RoundRobin) Len() int {
	return len(rr.backends)
}
