# tinyrp

```
  _   _             ____  ____
  | |_(_)_ __  _   _|  _ \|  _ \
  | __| | '_ \| | | | |_) | |_) |
  | |_| | | | | |_| |  _ <|  __/
  \__|_|_| |_|\__, |_| \_\_|
              |___/        
```

A fast, lightweight HTTP reverse proxy written in Go. Zero bloat, minimal dependencies, competitive with nginx/caddy/traefik.

## Features

- Tuned `http.Transport` with aggressive connection pooling (1024 idle, 256/host)
- `sync.Pool` buffer reuse (zero allocs on the copy path)
- `httprouter` radix tree routing (O(path length) lookup)
- Atomic round-robin load balancing across multiple backends
- Circuit breaker with half-open recovery (skips dead backends)
- Exponential backoff retry on 502/503/504
- `SO_REUSEPORT` + `TCP_NODELAY` socket tuning
- Graceful shutdown (drains in-flight requests on SIGINT/SIGTERM)
- HTTP/2 backend support
- Single dependency: `gopkg.in/yaml.v3`

## Benchmark: tinyrp vs the competition

All tests run on the same machine, same backend, same `wrk` parameters. Backend returns a small JSON payload (~63 bytes).

**Machine:** Apple M1 Pro, macOS, Go 1.25  
**Tool:** `wrk -t4 -d10s --latency`  
**Backend:** Simple Go HTTP server returning `{"status":"ok","data":"benchmark payload for proxy comparison"}`

### 100 concurrent connections

| Proxy | Req/s | Avg Latency | p50 | p99 | Memory (RSS) |
|-------|------:|------------:|----:|----:|-------------:|
| **Direct (no proxy)** | **33,610** | **2.97ms** | **2.95ms** | **3.47ms** | - |
| nginx 1.27 | 16,537 | 6.08ms | 5.92ms | 7.56ms | 43 MB |
| traefik 3.x | 16,171 | 6.17ms | 6.12ms | 7.29ms | 106 MB |
| caddy 2.x | 15,834 | 6.28ms | 6.16ms | 8.57ms | 59 MB |
| **tinyrp** | **15,034** | **6.63ms** | **6.61ms** | **7.90ms** | **39 MB** |

### 256 concurrent connections

| Proxy | Req/s | Avg Latency | p50 | p99 | Memory (RSS) |
|-------|------:|------------:|----:|----:|-------------:|
| **Direct (no proxy)** | **33,874** | **7.52ms** | **7.44ms** | **8.51ms** | - |
| nginx 1.27 | 16,574 | 15.39ms | 15.20ms | 18.48ms | 45 MB |
| traefik 3.x | 16,051 | 15.87ms | 15.76ms | 19.45ms | 158 MB |
| **tinyrp** | **14,846** | **17.15ms** | **17.07ms** | **21.35ms** | **68 MB** |
| caddy 2.x | 14,837 | 17.21ms | 16.44ms | 32.67ms | 77 MB |

### Key takeaways

- tinyrp matches caddy and is within 10% of nginx/traefik throughput
- Lowest memory usage of all Go proxies (39 MB vs caddy 59 MB vs traefik 106 MB at 100 conns)
- Better p99 than caddy at high concurrency (21ms vs 33ms at 256 conns)
- ~50% proxy overhead vs direct (expected for any L7 proxy)
- Round-robin load balancer selection: **43ns, zero allocations**

### Go benchmark results (in-process)

```
BenchmarkProxySmallBody-4          17,581    66,918 ns/op    12,540 B/op    142 allocs/op
BenchmarkProxyLargeBody-4             552 2,155,940 ns/op    14,209 B/op    186 allocs/op
BenchmarkRoundRobin-4          25,976,719        43 ns/op         0 B/op      0 allocs/op
BenchmarkProxyLatency-4             6,289   189,366 ns/op    12,577 B/op    143 allocs/op
BenchmarkDirectVsProxy/direct-4    34,960    34,824 ns/op     5,832 B/op     67 allocs/op
BenchmarkDirectVsProxy/proxied-4   18,028    66,444 ns/op    12,511 B/op    142 allocs/op
```

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

# Full comparison vs nginx/caddy/traefik (requires brew install wrk nginx caddy traefik)
bash bench/run_comparison.sh
```

## Requirements

- Go 1.23+
