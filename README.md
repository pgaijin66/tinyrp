# tinyrp

```
  _   _             ____  ____
  | |_(_)_ __  _   _|  _ \|  _ \
  | __| | '_ \| | | | |_) | |_) |
  | |_| | | | | |_| |  _ <|  __/
  \__|_|_| |_|\__, |_| \_\_|
              |___/        
```

A fast, lightweight HTTP reverse proxy written in Go. Hand-rolled proxy with zero-copy optimizations, no bloat, minimal dependencies. Competitive with nginx, caddy, and traefik.

## Features

- Hand-rolled reverse proxy (no `httputil.ReverseProxy`) with in-place request mutation
- Tuned `http.Transport` with aggressive connection pooling (2048 idle, 512/host)
- `sync.Pool` 64KB buffer reuse for body copies
- Atomic round-robin load balancing across multiple backends
- Circuit breaker with half-open recovery (skips dead backends automatically)
- Exponential backoff retry on 502/503/504
- `SO_REUSEPORT` + `TCP_NODELAY` socket tuning
- L4 TCP proxy mode for raw passthrough (splice-backed on Linux)
- Graceful shutdown (drains in-flight on SIGINT/SIGTERM)
- GOGC=200 tuned GC for lower pause frequency
- Only 2 dependencies: `gopkg.in/yaml.v3` + `golang.org/x/sys`

## Benchmark: tinyrp vs the competition

All tests run on the same machine, same backend, same `wrk` parameters. Each proxy forwards to the same Go backend returning a ~63 byte JSON payload. 3-second warmup before each 10-second measurement.

**Machine:** Apple M1 Pro, macOS, Go 1.25  
**Tool:** `wrk -t4 -d10s --latency`  
**Backend:** `{"status":"ok","data":"benchmark payload for proxy comparison"}`

### 100 concurrent connections

| Proxy | Req/s | Avg Latency | p50 | p99 | Memory (RSS) |
|-------|------:|------------:|----:|----:|-------------:|
| **Direct (no proxy)** | **34,110** | **2.92ms** | **2.88ms** | **3.42ms** | - |
| nginx 1.27 | 16,532 | 6.04ms | 5.96ms | 7.15ms | 44 MB |
| traefik 3.x | 16,155 | 6.15ms | 6.13ms | 7.03ms | 108 MB |
| caddy 2.x | 15,759 | 6.33ms | 6.24ms | 7.82ms | 59 MB |
| **tinyrp** | **15,427** | **6.47ms** | **6.40ms** | **7.35ms** | **62 MB** |

### 256 concurrent connections

| Proxy | Req/s | Avg Latency | p50 | p99 | Memory (RSS) |
|-------|------:|------------:|----:|----:|-------------:|
| **Direct (no proxy)** | **34,110** | **2.92ms** | **2.88ms** | **3.42ms** | - |
| nginx 1.27 | 16,543 | 15.42ms | 15.24ms | 18.93ms | 45 MB |
| traefik 3.x | 16,100 | 15.81ms | 15.69ms | 19.11ms | 157 MB |
| caddy 2.x | 15,557 | 16.40ms | 16.05ms | 22.84ms | 73 MB |
| **tinyrp** | **15,396** | **16.57ms** | **16.43ms** | **19.43ms** | **113 MB** |

### Key takeaways

- tinyrp is within 7% of nginx throughput at 100 connections, within 7% at 256
- Better p99 latency than caddy at both concurrency levels (7.35ms vs 7.82ms, 19.43ms vs 22.84ms)
- Competitive p99 with traefik (7.35ms vs 7.03ms at 100c, 19.43ms vs 19.11ms at 256c)
- ~55% proxy overhead vs direct backend (expected for any L7 proxy with full HTTP parsing)
- Round-robin load balancer: **43ns per selection, zero allocations**
- Only 126 allocs/op (vs 142 before optimization, vs ~200+ for httputil.ReverseProxy with extras)

### Go benchmark results (in-process)

```
BenchmarkProxySmallBody-4          17,731    66,120 ns/op    10,810 B/op    126 allocs/op
BenchmarkProxyLargeBody-4             664 1,758,898 ns/op    12,680 B/op    167 allocs/op
BenchmarkRoundRobin-4          30,235,807        43 ns/op         0 B/op      0 allocs/op
BenchmarkProxyLatency-4             6,440   186,059 ns/op    11,256 B/op    132 allocs/op
BenchmarkDirectVsProxy/direct-4    36,326    32,845 ns/op     5,279 B/op     63 allocs/op
BenchmarkDirectVsProxy/proxied-4   17,924    66,460 ns/op    10,781 B/op    126 allocs/op
```

### What makes tinyrp fast

1. **No `httputil.ReverseProxy`** - hand-rolled proxy mutates the inbound request in-place instead of cloning it. Eliminates `Request.Clone()`, `Header.Clone()`, and `WithContext()` allocations.
2. **Direct header copy** - `dst[k] = vv` slice assignment instead of `Header.Add()` which canonicalizes every key.
3. **Pooled 64KB buffers** - `sync.Pool` for `io.CopyBuffer`, avoids per-request allocation.
4. **Transport tuning** - 2048 idle connections, 512 per host, compression disabled, HTTP/2 enabled.
5. **Socket options** - `SO_REUSEPORT` for kernel-level connection distribution, `TCP_NODELAY` to disable Nagle.
6. **GOGC=200** - halved GC frequency, trades memory for throughput.

## Quick start

```bash
make build
./bin/tinyrp
```

## Configuration

```yaml
server:
  host: "localhost"
  listen_port: "8080"
  scheme: http

resources:
  # single backend
  - name: api
    endpoint: /api
    destination_url: "http://localhost:9001"

  # multiple backends (round-robin load balanced)
  - name: web
    endpoint: /web
    destination_urls:
      - "http://localhost:9002"
      - "http://localhost:9003"
```

## Run benchmarks

```bash
# Go benchmarks (in-process, fast)
make bench-quick

# Full comparison vs nginx/caddy/traefik (requires: brew install wrk nginx caddy traefik)
bash bench/run_comparison.sh
```

## Requirements

- Go 1.22+
