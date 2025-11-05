# Helios

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/0xReLogic/Helios)](https://goreportcard.com/report/github.com/0xReLogic/Helios)
[![Go Version](https://img.shields.io/github/go-mod/go-version/0xReLogic/Helios)](https://github.com/0xReLogic/Helios)
[![License](https://img.shields.io/github/license/0xReLogic/Helios)](https://github.com/0xReLogic/Helios/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/0xReLogic/Helios.svg)](https://pkg.go.dev/github.com/0xReLogic/Helios)
[![Build Status](https://img.shields.io/github/actions/workflow/status/0xReLogic/Helios/build.yml?branch=main)](https://github.com/0xReLogic/Helios/actions)

</div>

Ultra-fast, production-grade L7 reverse proxy and load balancer - simple, extensible, and reliable.

## Overview

Helios is a modern, production-grade reverse proxy and load balancer for microservices. It combines intelligent routing (Round Robin, Least Connections, Weighted, IP Hash), active/passive health checks, low-overhead WebSocket/TLS termination, runtime control via the Admin API, and a pluggable middleware system delivering high throughput, low latency, and effortless operations.

## Features

- **HTTP Reverse Proxy**: Efficiently forwards HTTP requests to backend servers
- **WebSocket Proxy**: Full support for proxying WebSocket connections with connection pooling
- **TLS/SSL Termination**: Secures traffic by terminating TLS connections
- **Advanced Load Balancing**: Multiple distribution strategies:
  - Round Robin - Distributes requests sequentially across all healthy backends
  - Least Connections - Routes to the backend with the fewest active connections
  - Weighted Round Robin - Distributes requests based on user-assigned backend weights
  - IP Hash - Ensures requests from the same client IP are routed to the same backend
- **Intelligent Health Monitoring**:
  - Passive health checks - Detects failures from regular traffic patterns
  - Active health checks - Proactively monitors backend health with periodic requests
- **Request Rate Limiting**: Token bucket algorithm with proper IP parsing to prevent abuse and ensure fair usage
- **Circuit Breaker Pattern**: Prevents cascading failures by temporarily blocking requests to unhealthy services
- **Comprehensive Timeout Controls**:
  - Server-side timeouts (read, write, idle, handler, shutdown)
  - Backend-specific timeouts (dial, read, idle)
  - Protection against slow-read/write attacks
- **WebSocket Connection Pooling**: Per-backend connection pools with configurable idle limits
- **Metrics and Monitoring**:
  - Real-time metrics collection with Exponential Moving Average (EMA)
  - Lock-free atomic operations for minimal overhead
  - Memory-bounded metrics (~60% less GC pressure)
  - Health status endpoints
  - Backend performance monitoring
  - Request/response statistics
- **Configuration**: Simple YAML-based configuration with comprehensive validation
- **Performance**: 
  - Low memory footprint and high throughput
  - Optimized string operations (2.5x faster parsing)
  - Object pooling for zero-allocation metric copies
- **Reliability**: Automatic failover when backends become unhealthy
- **Admin API**: Runtime backend management, strategy switching, and JSON metrics/health
- **Structured Logging**: Configurable JSON or text logs with request/trace identifiers
- **Plugin Middleware**: Configurable middleware chain with built-in plugins:
  - Logging - Request/response logging with trace IDs
  - Size Limit - DoS protection via payload size limits
  - Gzip Compression - Response compression with 10MB buffer limit
  - Headers - Custom header injection
  - Request ID - Auto-generated request identifiers

## Architecture

```mermaid
graph TD
    Client([Client]) -->|HTTP Request| RateLimit[Rate Limiter]

    subgraph "Helios Load Balancer"
        RateLimit --> CircuitBreaker[Circuit Breaker]
        CircuitBreaker --> Helios[Helios Proxy]
        Helios --> LoadBalancer[Load Balancing Strategy]
        Helios --> HealthChecker[Health Checker]
        Helios --> MetricsCollector[Metrics Collector]

        subgraph "Health Monitoring"
            HealthChecker --> PassiveChecks[Passive Health Checks]
            HealthChecker --> ActiveChecks[Active Health Checks]
        end

        subgraph "Load Balancing Strategies"
            LoadBalancer --> RoundRobin[Round Robin]
            LoadBalancer --> LeastConn[Least Connections]
            LoadBalancer --> WeightedRR[Weighted Round Robin]
            LoadBalancer --> IPHash[IP Hash]
        end

        subgraph "Monitoring & Metrics"
            MetricsCollector --> MetricsAPI[Metrics API :9090]
            MetricsCollector --> HealthAPI[Health API :9090]
        end
    end

    Helios -->|Forward Request| Backend1[Backend Server 1]
    Helios -->|Forward Request| Backend2[Backend Server 2]
    Helios -->|Forward Request| Backend3[Backend Server 3]

    ActiveChecks -.->|Health Probe| Backend1
    ActiveChecks -.->|Health Probe| Backend2
    ActiveChecks -.->|Health Probe| Backend3

    Backend1 -->|Response| Helios
    Backend2 -->|Response| Helios
    Backend3 -->|Response| Helios

    Helios -->|HTTP Response| Client

    MetricsAPI -.->|Monitoring Data| Monitoring[Monitoring System]
    HealthAPI -.->|Health Status| Monitoring
```

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Git (for cloning the repository)

### Installation

#### From Source

1. Clone the repository:

   ```bash
   git clone https://github.com/0xReLogic/Helios.git
   cd Helios
   ```

2. Build the project:

   ```bash
   go build -o helios.exe ./cmd/helios
   ```

3. Run Helios:
   ```bash
   ./helios.exe
   ```

#### Using Pre-built Binaries

1. Download the latest release from the [Releases page](https://github.com/0xReLogic/Helios/releases)
2. Extract the archive
3. Run the executable:
   ```bash
   ./helios.exe
   ```

### Running Test Backends

For testing purposes, Helios includes simple backend servers:

```bash
# Build the backend server
go build -o backend.exe ./cmd/backend

# Run multiple backend servers
./backend.exe --port=8081 --id=1
./backend.exe --port=8082 --id=2
./backend.exe --port=8083 --id=3
```

On Windows, you can use the provided batch script:

```bash
start_backends.bat
```

## Configuration

Helios is configured via `helios.yaml`:

```yaml
server:
  port: 8080 # Port for the proxy server
  tls:
    enabled: true # Enable TLS/SSL termination
    certFile: "certs/cert.pem" # Path to TLS certificate file
    keyFile: "certs/key.pem" # Path to TLS private key file
  timeouts:
    read: 15 # ReadTimeout in seconds (protects against slow-read attacks)
    write: 15 # WriteTimeout in seconds (prevents slow writes)
    idle: 60 # IdleTimeout in seconds (keep-alive timeout)
    handler: 30 # Handler timeout in seconds (end-to-end request timeout)
    shutdown: 30 # Graceful shutdown timeout in seconds
    backend_dial: 10 # Backend connection dial timeout in seconds
    backend_read: 30 # Backend response read timeout in seconds
    backend_idle: 90 # Backend idle connection timeout in seconds

backends:
  - name: "server1"
    address: "http://localhost:8081"
    weight: 5
  - name: "server2"
    address: "http://localhost:8082"
    weight: 2
  - name: "server3"
    address: "http://localhost:8083"
    weight: 1

load_balancer:
  strategy: "round_robin" # Options: "round_robin", "least_connections", "weighted_round_robin", "ip_hash"
  websocket_pool:
    enabled: true # Enable WebSocket connection pooling
    max_idle_per_backend: 10 # Maximum idle connections per backend
    idle_timeout_seconds: 300 # Idle connection timeout (5 minutes)

health_checks:
  active:
    enabled: true
    interval: 5 # Interval in seconds
    timeout: 3 # Timeout in seconds
    path: "/health"
  passive:
    enabled: true
    unhealthy_threshold: 10 # Number of failures before marking as unhealthy
    unhealthy_timeout: 15 # Time in seconds to keep backend unhealthy

rate_limit:
  enabled: true
  max_tokens: 100 # Maximum tokens in bucket
  refill_rate_seconds: 1 # Refill rate in seconds

circuit_breaker:
  enabled: true
  max_requests: 100 # Max requests in half-open state
  interval_seconds: 30 # Time window for failure counting
  timeout_seconds: 15 # Time to wait before moving from open to half-open
  failure_threshold: 50 # Number of failures to open circuit
  success_threshold: 10 # Number of successes to close circuit

admin_api:
  enabled: true
  port: 9091 # Port for admin API server
  auth_token: "change-me" # JWT token for authentication (change in production)

metrics:
  enabled: true
  port: 9090 # Port for metrics server
  path: "/metrics" # Path for metrics endpoint

logging:
  level: "info" # Log level: debug, info, warn, error
  format: "text" # Log format: text (console) or json (machine-readable)
  include_caller: true # Include file and line number in logs
  request_id:
    enabled: true # Auto-generate and propagate request IDs
    header: "X-Request-ID" # Header name for request ID
  trace:
    enabled: true # Enable distributed tracing
    header: "X-Trace-ID" # Header name for trace ID

plugins:
  enabled: true
  chain:
    - name: logging
    - name: size_limit
      config:
        max_request_body: 10485760 # 10MB in bytes
        max_response_body: 52428800 # 50MB in bytes
    - name: gzip
      config:
        level: 5 # Compression level (1-9, default: 5)
        min_size: 1024 # Minimum response size to compress (bytes, default: 1024)
        content_types:
          - "text/html"
          - "text/css"
          - "text/plain"
          - "application/json"
          - "application/javascript"
          - "application/xml"
    - name: headers
      config:
        set:
          X-App: Helios
        request_set:
          X-From: LB
```

## Quick Start

### TLS/SSL Configuration

Helios supports TLS termination for secure HTTPS connections. The repository includes sample certificates for testing purposes only.

**WARNING: DO NOT use the included certificates in production. They are publicly available and insecure.**

#### Generating Your Own Certificates

For production use, generate your own TLS certificates:

**Self-signed certificate (for testing):**
```bash
openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes -subj "/CN=localhost"
```

**Using Let's Encrypt (for production):**
```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate
sudo certbot certonly --standalone -d yourdomain.com

# Copy certificates to Helios directory
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem certs/cert.pem
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem certs/key.pem
```

**Enable TLS in helios.yaml:**
```yaml
server:
  port: 8080
  tls:
    enabled: true
    certFile: "certs/cert.pem"
    keyFile: "certs/key.pem"
```

### Timeout Configuration

Helios provides comprehensive timeout controls to protect against various attack vectors and ensure reliable service:

**Server-side timeouts (protects Helios from malicious clients):**
- `read` - Maximum duration for reading the entire request (default: 15s)
  - Protects against slow-read attacks
- `write` - Maximum duration before timing out writes of the response (default: 15s)
  - Prevents slow writes from holding connections
- `idle` - Maximum duration to wait for the next request when keep-alives are enabled (default: 60s)
  - Manages connection lifecycle
- `handler` - End-to-end timeout for the entire request handler (default: 30s)
  - Ensures requests don't hang indefinitely
- `shutdown` - Maximum duration for graceful shutdown (default: 30s)
  - Allows in-flight requests to complete

**Backend timeouts (controls Helios → backend communication):**
- `backend_dial` - Maximum time to establish connection to backend (default: 10s)
  - Fails fast if backend is unreachable
- `backend_read` - Maximum time to wait for response from backend (default: 30s)
  - Prevents hanging on slow backends
- `backend_idle` - Maximum idle time for keep-alive connections to backends (default: 90s)
  - Connection pooling optimization

**Example configuration:**
```yaml
server:
  port: 8080
  timeouts:
    read: 15
    write: 15
    idle: 60
    handler: 30
    shutdown: 30
    backend_dial: 10
    backend_read: 30
    backend_idle: 90
```

**Production recommendations:**
- Keep `handler` timeout lower than backend services' timeouts
- Set `backend_read` based on your slowest acceptable backend response time
- Adjust `backend_dial` based on network latency to your backends
- Use shorter timeouts for public-facing services to prevent resource exhaustion

### Logging Configuration

Helios emits structured logs using [zerolog](https://github.com/rs/zerolog) for efficient structured logging. Configure verbosity, output format, and observability headers via the `logging` block:

- **level** – Supported values: `debug`, `info`, `warn`, `error` (default `info`).
- **format** – `text` (default) for console readability or `json` for machine-friendly ingestion.
- **include_caller** – When `true`, adds caller information to log entries.
- **request_id** – Enables automatic generation and propagation of a request identifier. The value is attached to responses and forwarded to backends using the configured header (default `X-Request-ID`).
- **trace** – Propagates distributed trace identifiers (default header `X-Trace-ID`) and includes them in every log entry.

Each HTTP request is logged with latency, status code, backend target, and associated request/trace identifiers, simplifying correlation across services.

#### Verifying request & trace propagation

1. Start one or more sample backends:
   ```bash
   go run ./cmd/backend --port=8081 --id=server1
   ```
2. In a second terminal, run the load balancer (it picks up `helios.yaml`):
   ```bash
   go run ./cmd/helios
   ```
3. Issue a request and optionally provide your own trace identifier:
   ```bash
   curl -H "X-Trace-ID: trace_demo" http://localhost:8080/api/users
   ```
4. Observe the terminal output. With the default `text` format you will see entries similar to:
   ```text
   time=2025-10-02T10:30:00Z level=info request_id=req_abc123 trace_id=trace_demo method=GET path=/api/users status=200 latency_ms=45 backend=server1 message="request completed"
   ```
5. To compare against JSON logs, change `logging.format` to `json`, restart Helios, and repeat step 3—the output will be emitted as structured JSON for side-by-side comparison.

### Build and Run

```bash
git clone https://github.com/0xReLogic/Helios.git
cd Helios
go build -o helios ./cmd/helios
./helios
```

### Basic Configuration (helios.yaml)

```yaml
server:
  port: 8080
  timeouts:
    read: 15
    write: 15
    idle: 60
    handler: 30

backends:
  - name: "server1"
    address: "http://localhost:8081"
    weight: 5
  - name: "server2"
    address: "http://localhost:8082"
    weight: 2
  - name: "server3"
    address: "http://localhost:8083"
    weight: 1

load_balancer:
  strategy: "round_robin" # round_robin, least_connections, weighted_round_robin, ip_hash
  websocket_pool:
    enabled: true
    max_idle_per_backend: 10
    idle_timeout_seconds: 300

health_checks:
  active:
    enabled: true
    interval: 5
    timeout: 3
    path: "/health"

circuit_breaker:
  enabled: true
  failure_threshold: 50

metrics:
  enabled: true
  port: 9090
```

### Test Backends

```bash
go build -o backend ./cmd/backend
./backend --port=8081 --id=1 &
./backend --port=8082 --id=2 &
./backend --port=8083 --id=3 &
```

## Monitoring & Management

### Metrics Endpoint

Access real-time metrics at `http://localhost:9090/metrics` (Prometheus format)

### Admin API

- Runtime backend management
- Strategy switching
- Health status monitoring
- JWT-protected endpoints

### Health Checks

- Active: Periodic backend health verification
- Passive: Request-based health tracking
- Circuit breaker: Automatic failure isolation

## Documentation

- [Plugin Development Guide](docs/plugin-development.md) - Learn how to create custom plugins
  - Example plugins: `internal/plugins/examples`

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Local Development with Docker Compose

Helios can be run locally with three backend servers using Docker Compose. This setup validates plugin behavior, health checks, and load balancing in a containerized environment.

#### Prerequisites

- Docker Engine 20.10 or higher
- Docker Compose V2 or higher

#### Services

| Service    | Port(s) | Description                              |
| ---------- | ------- | ---------------------------------------- |
| `helios`   | 8080    | Load balancer entrypoint                 |
|            | 9090    | Metrics endpoint (`/metrics`, `/health`) |
|            | 9091    | Admin API (token-protected)              |
| `backend1` | 8081    | Backend server instance 1                |
| `backend2` | 8082    | Backend server instance 2                |
| `backend3` | 8083    | Backend server instance 3                |

#### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/0xReLogic/Helios.git
   cd Helios
   ```

2. **Build and start all services**
   ```bash
   docker-compose up --build
   ```

   Or run in detached mode:
   ```bash
   docker-compose up -d --build
   ```

3. **Test the load balancer**
   ```bash
   # Send request through load balancer
   curl http://localhost:8080
   
   # Check metrics
   curl http://localhost:9090/metrics
   
   # Check health status
   curl http://localhost:9090/health
   ```

4. **Stop all services**
   ```bash
   docker-compose down
   ```

#### Configuration

The Docker Compose setup uses `helios.docker.yaml` which differs from the standard `helios.yaml`:

- **Backend addresses**: Uses Docker service names (`backend1:8080`) instead of localhost
- **Health check path**: Configured for `/` instead of `/health`
- **Timeouts**: Includes comprehensive timeout configuration for production readiness

To modify the configuration, edit `helios.docker.yaml` and restart the services:

```bash
docker-compose restart helios
```

#### Health Check Behavior

Helios performs active health checks on `/` for each backend every 10 seconds with a 7-second timeout. Backends respond with a success status when healthy.

If a backend fails to respond, it is marked unhealthy for 30 seconds before retry.

#### Viewing Logs

```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f helios
docker-compose logs -f backend1
```

#### Admin API

The Admin API is exposed on port 9091 and requires a token (configured in `helios.docker.yaml`). 

**Default token**: `change-me` (⚠️ Change this in production!)

Example requests:

```bash
# Get backend status
curl -H "Authorization: Bearer change-me" http://localhost:9091/api/backends

# Get circuit breaker metrics
curl -H "Authorization: Bearer change-me" http://localhost:9091/api/metrics
```

#### Metrics

Prometheus-compatible metrics are available at:

- **Metrics endpoint**: http://localhost:9090/metrics
- **Health endpoint**: http://localhost:9090/health

#### Plugin Chain

The default plugin chain includes:

- **logging**: Request logs with trace IDs and latency metrics
- **size_limit**: Enforces payload size limits (10MB request, 50MB response)
- **gzip**: Response compression for configured content types
- **headers**: Injects custom headers (`X-App: Helios`, `X-From: LB`)

#### Troubleshooting

**Port conflicts**:
```bash
# Check if ports are already in use
lsof -i :8080
lsof -i :9090
lsof -i :9091

# Modify ports in docker-compose.yaml if needed
```

**Rebuild after code changes**:
```bash
docker-compose down
docker-compose build --no-cache
docker-compose up
```

**Clean up everything**:
```bash
docker-compose down -v --remove-orphans
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Contributors

Thanks to all the amazing people who have contributed to Helios!

<a href="https://github.com/0xReLogic/Helios/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=0xReLogic/Helios" />
</a>
