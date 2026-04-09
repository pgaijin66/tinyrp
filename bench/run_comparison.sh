#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIGS="$SCRIPT_DIR/configs"
RESULTS="$SCRIPT_DIR/results"
mkdir -p "$RESULTS"

WRK_THREADS=4
WRK_DURATION=10s
BACKEND_PORT=9090

kill_port() {
    lsof -ti:$1 2>/dev/null | xargs kill -9 2>/dev/null || true
}

cleanup() {
    kill_port 8080
    kill_port 8081
    kill_port 8082
    kill_port 8083
    kill_port $BACKEND_PORT
    nginx -s stop 2>/dev/null || true
    caddy stop 2>/dev/null || true
}
trap cleanup EXIT
cleanup 2>/dev/null || true
sleep 1

echo "=== building ==="
cd "$SCRIPT_DIR/.."
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$SCRIPT_DIR/tinyrp_bin" cmd/main.go
go build -o "$SCRIPT_DIR/backend_bin" bench/backend/main.go

echo "=== starting backend on :$BACKEND_PORT ==="
"$SCRIPT_DIR/backend_bin" "$BACKEND_PORT" &
sleep 1
curl -sf http://127.0.0.1:$BACKEND_PORT/ > /dev/null || { echo "backend failed"; exit 1; }

get_mem() {
    local name="$1"
    local port="$2"
    local pid=$(lsof -ti:$port 2>/dev/null | head -1)
    if [ -n "$pid" ]; then
        local rss=$(ps -o rss= -p "$pid" 2>/dev/null | tr -d ' ')
        if [ -n "$rss" ]; then
            echo "$((rss / 1024)) MB"
        else
            echo "n/a"
        fi
    else
        echo "n/a"
    fi
}

run_wrk() {
    local name="$1"
    local url="$2"
    local conns="$3"
    wrk -t$WRK_THREADS -c$conns -d3s "$url" > /dev/null 2>&1
    wrk -t$WRK_THREADS -c$conns -d$WRK_DURATION --latency "$url" 2>&1
}

echo ""
echo "================================================================"
echo "  PROXY COMPARISON BENCHMARK"
echo "  $(date)"
echo "  $(sysctl -n machdep.cpu.brand_string)"
echo "  wrk: $WRK_THREADS threads, $WRK_DURATION duration"
echo "================================================================"

for CONNS in 100 256; do
    echo ""
    echo "======================== $CONNS connections ========================"

    # Direct
    echo ""
    echo "--- direct (baseline) @ $CONNS conns ---"
    run_wrk "direct" "http://127.0.0.1:$BACKEND_PORT/" $CONNS | tee "$RESULTS/direct_${CONNS}.txt"

    # tinyrp
    echo ""
    echo "--- tinyrp @ $CONNS conns ---"
    cp "$CONFIGS/tinyrp_config.yaml" "$SCRIPT_DIR/../data/config.yaml"
    "$SCRIPT_DIR/tinyrp_bin" 2>/dev/null &
    sleep 1
    MEM_TINYRP_PRE=$(get_mem "tinyrp" 8080)
    run_wrk "tinyrp" "http://127.0.0.1:8080/proxy/test" $CONNS | tee "$RESULTS/tinyrp_${CONNS}.txt"
    MEM_TINYRP_POST=$(get_mem "tinyrp" 8080)
    echo "memory: before=$MEM_TINYRP_PRE after=$MEM_TINYRP_POST"
    kill_port 8080; sleep 1

    # nginx
    echo ""
    echo "--- nginx @ $CONNS conns ---"
    nginx -c "$CONFIGS/nginx.conf"
    sleep 1
    MEM_NGINX_PRE=$(get_mem "nginx" 8081)
    run_wrk "nginx" "http://127.0.0.1:8081/" $CONNS | tee "$RESULTS/nginx_${CONNS}.txt"
    MEM_NGINX_POST=$(get_mem "nginx" 8081)
    echo "memory: before=$MEM_NGINX_PRE after=$MEM_NGINX_POST"
    nginx -s stop 2>/dev/null; sleep 1

    # caddy
    echo ""
    echo "--- caddy @ $CONNS conns ---"
    caddy start --config "$CONFIGS/Caddyfile" --adapter caddyfile 2>/dev/null
    sleep 1
    MEM_CADDY_PRE=$(get_mem "caddy" 8082)
    run_wrk "caddy" "http://127.0.0.1:8082/" $CONNS | tee "$RESULTS/caddy_${CONNS}.txt"
    MEM_CADDY_POST=$(get_mem "caddy" 8082)
    echo "memory: before=$MEM_CADDY_PRE after=$MEM_CADDY_POST"
    caddy stop 2>/dev/null; sleep 1

    # traefik
    echo ""
    echo "--- traefik @ $CONNS conns ---"
    traefik --configFile="$CONFIGS/traefik.yaml" > /dev/null 2>&1 &
    sleep 2
    MEM_TRAEFIK_PRE=$(get_mem "traefik" 8083)
    run_wrk "traefik" "http://127.0.0.1:8083/" $CONNS | tee "$RESULTS/traefik_${CONNS}.txt"
    MEM_TRAEFIK_POST=$(get_mem "traefik" 8083)
    echo "memory: before=$MEM_TRAEFIK_PRE after=$MEM_TRAEFIK_POST"
    kill_port 8083; sleep 1
done

echo ""
echo "================================================================"
echo "  DONE"
echo "================================================================"
