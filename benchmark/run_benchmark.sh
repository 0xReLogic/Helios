#!/bin/bash

# Benchmark script: Nginx vs Helios
# Apple-to-apple comparison with all fancy features DISABLED

set -e

BENCHMARK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$BENCHMARK_DIR")"
RESULTS_FILE="$BENCHMARK_DIR/results.txt"

echo "NGINX VS HELIOS BENCHMARK"
echo "================================"
echo ""

# Build Helios
echo "[*] Building Helios..."
cd "$PROJECT_ROOT"
go build -o benchmark/helios ./cmd/helios

# Build simple backend
echo "[*] Building test backends..."
go build -o benchmark/backend ./benchmark/simple_backend.go

# Start 3 backend servers
echo "[*] Starting backend servers on ports 9001, 9002, 9003..."
PORT=9001 "$BENCHMARK_DIR/backend" > /dev/null 2>&1 &
BACKEND1_PID=$!
PORT=9002 "$BENCHMARK_DIR/backend" > /dev/null 2>&1 &
BACKEND2_PID=$!
PORT=9003 "$BENCHMARK_DIR/backend" > /dev/null 2>&1 &
BACKEND3_PID=$!

sleep 2
echo "[+] Backends running (PIDs: $BACKEND1_PID, $BACKEND2_PID, $BACKEND3_PID)"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "[*] Cleaning up..."
    kill $BACKEND1_PID $BACKEND2_PID $BACKEND3_PID 2>/dev/null || true
    kill $HELIOS_PID 2>/dev/null || true
    sudo nginx -s stop 2>/dev/null || true
    echo "[+] Cleanup done"
}
trap cleanup EXIT

# Function to run benchmark
run_benchmark() {
    local name=$1
    local url=$2
    local duration=${3:-30s}
    local threads=${4:-4}
    local connections=${5:-100}
    
    echo "[*] Benchmarking: $name"
    echo "   URL: $url"
    echo "   Duration: $duration, Threads: $threads, Connections: $connections"
    echo ""
    
    sleep 2  # Let server stabilize
    
    wrk -t$threads -c$connections -d$duration --latency "$url" 2>&1 | tee -a "$RESULTS_FILE"
    echo ""
    echo "---"
    echo ""
}

# Clear previous results
> "$RESULTS_FILE"

echo "=====================================" | tee "$RESULTS_FILE"
echo "NGINX VS HELIOS BENCHMARK RESULTS" | tee -a "$RESULTS_FILE"
echo "Date: $(date)" | tee -a "$RESULTS_FILE"
echo "=====================================" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

# Test 1: Helios
echo "TEST 1: HELIOS (minimal config)" | tee -a "$RESULTS_FILE"
echo "=================================" | tee -a "$RESULTS_FILE"
"$BENCHMARK_DIR/helios" -config "$BENCHMARK_DIR/helios-minimal.yaml" > /dev/null 2>&1 &
HELIOS_PID=$!
echo "[+] Helios started on port 8080 (PID: $HELIOS_PID)"

run_benchmark "Helios - 100 connections" "http://localhost:8080/" "30s" 4 100
run_benchmark "Helios - 500 connections" "http://localhost:8080/" "30s" 8 500
run_benchmark "Helios - 1000 connections" "http://localhost:8080/" "30s" 12 1000

echo "[*] Stopping Helios..."
kill $HELIOS_PID
wait $HELIOS_PID 2>/dev/null || true
sleep 2

# Test 2: Nginx
echo "" | tee -a "$RESULTS_FILE"
echo "TEST 2: NGINX (minimal config)" | tee -a "$RESULTS_FILE"
echo "=================================" | tee -a "$RESULTS_FILE"
sudo nginx -c "$BENCHMARK_DIR/nginx.conf"
echo "[+] Nginx started on port 8081"

run_benchmark "Nginx - 100 connections" "http://localhost:8081/" "30s" 4 100
run_benchmark "Nginx - 500 connections" "http://localhost:8081/" "30s" 8 500
run_benchmark "Nginx - 1000 connections" "http://localhost:8081/" "30s" 12 1000

echo "[*] Stopping Nginx..."
sudo nginx -s stop
sleep 2

echo "" | tee -a "$RESULTS_FILE"
echo "=====================================" | tee -a "$RESULTS_FILE"
echo "[+] BENCHMARK COMPLETE!" | tee -a "$RESULTS_FILE"
echo "=====================================" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"
echo "[*] Full results saved to: $RESULTS_FILE"
echo ""
echo "[*] Quick summary:"
echo "   - Both tested with 100, 500, and 1000 concurrent connections"
echo "   - All fancy features DISABLED for fair comparison"
echo "   - Same backend servers, same timeouts"
echo ""
