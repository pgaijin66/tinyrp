package lb

import (
	"net/url"
	"testing"
	"time"

	"github.com/pgaijin66/tinyrp/internal/cb"
)

func makeBackends(n int) []Backend {
	backends := make([]Backend, n)
	for i := range backends {
		u, _ := url.Parse("http://localhost:9000")
		backends[i] = Backend{
			URL: u,
			CB:  cb.New(3, 100*time.Millisecond),
		}
	}
	return backends
}

func TestRoundRobinDistribution(t *testing.T) {
	rr := NewRoundRobin(makeBackends(3))
	counts := [3]int{}
	for i := 0; i < 300; i++ {
		b := rr.Next()
		for j, bb := range rr.backends {
			if b.URL == bb.URL && b.CB == bb.CB {
				counts[j]++
				break
			}
		}
	}
	for i, c := range counts {
		if c != 100 {
			t.Errorf("backend %d got %d requests, expected 100", i, c)
		}
	}
}

func TestSkipsOpenCircuitBreaker(t *testing.T) {
	backends := makeBackends(3)
	// trip backend 0
	for i := 0; i < 3; i++ {
		backends[0].CB.RecordFailure()
	}

	rr := NewRoundRobin(backends)
	for i := 0; i < 20; i++ {
		b := rr.Next()
		if b.CB == backends[0].CB {
			t.Fatal("should not pick backend with open circuit breaker")
		}
	}
}

func TestFallbackWhenAllOpen(t *testing.T) {
	backends := makeBackends(2)
	for i := range backends {
		for j := 0; j < 3; j++ {
			backends[i].CB.RecordFailure()
		}
	}
	rr := NewRoundRobin(backends)

	// should still return something (fallback)
	b := rr.Next()
	if b.URL == nil {
		t.Fatal("should return a backend even when all are open")
	}
}
