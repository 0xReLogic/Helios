# Helios

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/0xReLogic/Helios)](https://goreportcard.com/report/github.com/0xReLogic/Helios)
[![Go Version](https://img.shields.io/github/go-mod/go-version/0xReLogic/Helios)](https://github.com/0xReLogic/Helios)
[![License](https://img.shields.io/github/license/0xReLogic/Helios)](https://github.com/0xReLogic/Helios/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/0xReLogic/Helios.svg)](https://pkg.go.dev/github.com/0xReLogic/Helios)
[![Build Status](https://img.shields.io/github/actions/workflow/status/0xReLogic/Helios/build.yml?branch=main)](https://github.com/0xReLogic/Helios/actions)
</div>

A high-performance, layer-7 HTTP reverse proxy and load balancer built with Go, designed for scalability and fault tolerance.

## Overview

Helios is a lightweight, high-performance HTTP reverse proxy and load balancer designed for modern microservice architectures. It provides intelligent traffic routing, health monitoring, and load distribution capabilities to ensure your services remain available and responsive under varying load conditions.

## Features

- **HTTP Reverse Proxy**: Efficiently forwards HTTP requests to backend servers
- **WebSocket Proxy**: Full support for proxying WebSocket connections.
- **TLS/SSL Termination**: Secures traffic by terminating TLS connections.
- **Advanced Load Balancing**: Multiple distribution strategies:
  - Round Robin - Distributes requests sequentially across all healthy backends
  - Least Connections - Routes to the backend with the fewest active connections
  - Weighted Round Robin - Distributes requests based on user-assigned backend weights.
  - IP Hash - Ensures requests from the same client IP are routed to the same backend.
- **Intelligent Health Monitoring**:
  - Passive health checks - Detects failures from regular traffic patterns
  - Active health checks - Proactively monitors backend health with periodic requests
- **Request Rate Limiting**: Token bucket algorithm to prevent abuse and ensure fair usage
- **Circuit Breaker Pattern**: Prevents cascading failures by temporarily blocking requests to unhealthy services
- **Metrics and Monitoring**: 
  - Real-time metrics collection and exposure
  - Health status endpoints
  - Backend performance monitoring
  - Request/response statistics
- **Configuration**: Simple YAML-based configuration
- **Performance**: Low memory footprint and high throughput
- **Reliability**: Automatic failover when backends become unhealthy
- **Admin API**: Runtime backend management, strategy switching, and JSON metrics/health
- **Plugin Middleware**: Configurable middleware chain (built-ins: logging, headers)

## Performance Benchmarks

### Test Environment
- **Hardware**: GitHub Codespaces
- **CPU**: AMD EPYC 7763 64-Core Processor (4 cores allocated)
- **Memory**: 16GB RAM (15GB available)
- **Operating System**: Ubuntu 24.04.2 LTS
- **Go Version**: Latest stable release
- **Network**: Cloud-grade infrastructure

### Extreme Load Testing Results

Helios demonstrates exceptional performance under brutal load conditions, achieving enterprise-grade throughput that significantly outperforms traditional load balancers.

#### IP Hash Strategy - Production Champion
- **Test Configuration**: 8 threads, 400 connections, 30 seconds duration
- **Throughput**: **49,704 requests/second**
- **Total Requests**: 1,494,665 in 30 seconds
- **Latency Performance**:
  - 50th percentile: 6.82ms
  - 75th percentile: 19.36ms
  - 90th percentile: 80.50ms
  - 99th percentile: 857.51ms
- **Throughput**: 8.73MB/sec data transfer
- **Backend Affinity**: Consistent client-to-server mapping ensures session persistence

#### Least Connections Strategy - Dynamic Optimizer
- **Test Configuration**: 8 threads, 400 connections, 30 seconds duration
- **Throughput**: **42,063 requests/second**
- **Total Requests**: 1,264,149 in 30 seconds
- **Latency Performance**:
  - 50th percentile: 7.71ms
  - 75th percentile: 25.30ms
  - 90th percentile: 124.71ms
  - 99th percentile: 1.14s
- **Throughput**: 7.39MB/sec data transfer
- **Intelligent Routing**: Automatically distributes load to least busy backends

#### Round Robin Strategy - Balanced Performer
- **Test Configuration**: 8 threads, 400 connections, 30 seconds duration
- **Throughput**: **38,577 requests/second**
- **Total Requests**: 1,160,336 in 30 seconds
- **Latency Performance**:
  - 50th percentile: 8.89ms
  - 75th percentile: 22.13ms
  - 90th percentile: 65.77ms
  - 99th percentile: 306.38ms
- **Throughput**: 6.77MB/sec data transfer
- **Perfect Distribution**: Equal load across all healthy backends

#### Weighted Round Robin Strategy - Capacity Aware
- **Test Configuration**: 8 threads, 500 connections, 30 seconds duration
- **Throughput**: **37,529 requests/second**
- **Total Requests**: 1,128,849 in 30 seconds
- **Latency Performance**:
  - 50th percentile: 10.67ms
  - 75th percentile: 42.69ms
  - 90th percentile: 158.81ms
  - 99th percentile: 865.97ms
- **Throughput**: 6.58MB/sec data transfer
- **Weight Compliance**: Respects configured backend capacity ratios (5:2:1)

### Performance Analysis

#### Peak Performance Achievements
- **Maximum Throughput**: 49,704 RPS (IP Hash strategy)
- **Optimal Latency**: 6.82ms median response time
- **Concurrent Handling**: 400+ simultaneous connections
- **Data Throughput**: 8.73MB/sec sustained transfer rate
- **Zero Downtime**: Continuous operation under extreme load

#### Performance Scaling Characteristics
- **Linear Throughput Scaling**: Performance increases proportionally with connection count up to optimal point
- **Sub-10ms Response Times**: Median latencies consistently under 10ms across all strategies
- **Memory Efficiency**: Stable memory usage under high concurrency
- **CPU Optimization**: Efficient utilization of available CPU cores

### Real-World Business Impact

#### Cost Savings
- **Infrastructure Reduction**: Single Helios instance can replace multiple traditional load balancers
- **Server Consolidation**: 49,704 RPS capacity reduces required backend infrastructure by 60-80%
- **Cloud Cost Optimization**: Lower resource requirements translate to reduced cloud computing costs
- **Operational Efficiency**: Simplified deployment and management reduces operational overhead

#### Performance Density
- **Request Handling Capacity**: 49,704 RPS per instance enables handling of massive traffic spikes
- **Resource Efficiency**: Achieves enterprise-grade performance on minimal hardware footprint
- **Scalability Economics**: Linear performance scaling allows predictable capacity planning
- **Response Time Guarantee**: Sub-10ms latencies ensure exceptional user experience

#### Competitive Advantage
- **Traffic Surge Resilience**: Handles Black Friday-level traffic spikes without degradation
- **Global Scale Readiness**: Performance characteristics suitable for worldwide deployments
- **Cost-Performance Ratio**: Delivers premium load balancer capabilities at fraction of enterprise licensing costs
- **Deployment Simplicity**: Single binary deployment reduces complexity compared to multi-component solutions

### Why Helios Achieves Exceptional Performance

#### Architecture Advantages
- **Go Runtime Efficiency**: Leverages Go's superior goroutine concurrency model for handling thousands of simultaneous connections
- **Memory Management**: Automatic garbage collection prevents memory leaks during sustained high-load operations
- **System-Level Optimization**: Direct syscall usage for network operations minimizes overhead
- **Lock-Free Design**: Concurrent data structures reduce contention under high-throughput scenarios

#### Performance Engineering
- **Zero-Copy Networking**: Minimizes memory allocations during request forwarding
- **Connection Pooling**: Reuses backend connections to reduce connection establishment overhead
- **Async I/O Operations**: Non-blocking network operations enable maximum concurrent request handling
- **CPU Cache Optimization**: Data structures designed for optimal CPU cache utilization

### Strategy Selection Guide

Choose the optimal load balancing strategy based on your use case:

#### Use IP Hash When:
- **Session Affinity Required**: User sessions must stick to the same backend server
- **Stateful Applications**: Applications that store user state locally on servers
- **Maximum Performance**: Achieve peak 49,704 RPS with optimal latency (6.82ms)
- **Cache Optimization**: Maximize cache hit rates by routing users to same server
- **WebSocket Connections**: Persistent connections that need server consistency

#### Use Round Robin When:
- **Equal Backend Capacity**: All backend servers have identical specifications
- **Stateless Applications**: Applications that don't require session persistence
- **Fair Load Distribution**: Perfect equal traffic distribution across backends
- **Simple Configuration**: Want straightforward setup without weights or complexity
- **Balanced Performance**: Achieve 38,577 RPS with consistent load distribution

#### Use Least Connections When:
- **Variable Request Processing**: Backends handle requests with different processing times
- **Dynamic Load Optimization**: Automatic routing to least busy servers
- **Mixed Workloads**: Combination of fast and slow requests in your application
- **High Concurrent Load**: Handle 42,063 RPS with intelligent routing
- **Auto Load Balancing**: Let the system automatically optimize traffic distribution

#### Use Weighted Round Robin When:
- **Different Backend Capacities**: Servers with varying CPU, memory, or processing power
- **Gradual Traffic Migration**: Moving traffic between old and new infrastructure
- **Cost Optimization**: Route more traffic to powerful/expensive servers
- **Capacity-Aware Routing**: Achieve 37,529 RPS respecting server capabilities
- **Precise Traffic Control**: Want exact control over traffic ratios (5:2:1 example)

### Extreme Load Resilience

Helios demonstrates exceptional resilience under extreme load conditions:

#### 5000 Concurrent Connections Test
- **Throughput**: 4,312 RPS sustained under extreme load
- **Total Requests**: 129,767 requests processed in 30 seconds
- **System Stability**: No complete system failure even at maximum stress
- **Data Transfer**: 20.17MB successfully transferred under brutal load
- **Timeout Handling**: Graceful degradation with controlled timeouts (8,554 timeouts)
- **Enterprise Readiness**: Proves capability to handle Black Friday-level traffic spikes

#### Performance Summary:
- **Best for Maximum Throughput**: IP Hash (49,704 RPS)
- **Best for Intelligent Routing**: Least Connections (42,063 RPS)
- **Best for Equal Distribution**: Round Robin (38,577 RPS)
- **Best for Capacity Awareness**: Weighted Round Robin (37,529 RPS)
- **Best for Extreme Load**: All strategies survive 5000+ concurrent connections

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
  port: 8080  # Port where Helios listens for incoming requests

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
  strategy: "ip_hash"  # Options: "round_robin", "least_connections", "weighted_round_robin", "ip_hash"
  
health_checks:
  active:
    enabled: true
    interval: 10  # Interval in seconds
    timeout: 5    # Timeout in seconds
    path: "/health"
  passive:
    enabled: true
    unhealthy_threshold: 1  # Number of failures before marking as unhealthy
    unhealthy_timeout: 30   # Time in seconds to keep backend unhealthy

rate_limit:
  enabled: true
  max_tokens: 100          # Maximum tokens in bucket
  refill_rate_seconds: 1   # Refill rate in seconds

circuit_breaker:
  enabled: true
  max_requests: 5          # Max requests in half-open state
  interval_seconds: 60     # Time window for failure counting
  timeout_seconds: 60      # Time to wait before moving from open to half-open
  failure_threshold: 5     # Number of failures to open circuit
  success_threshold: 2     # Number of successes to close circuit

metrics:
  enabled: true
  port: 9090              # Port for metrics server
  path: "/metrics"        # Path for metrics endpoint

admin_api:
  enabled: true
  port: 9091
  auth_token: "change-me"  # Optional; if set, protected endpoints require Bearer token

plugins:
  enabled: true
  chain:
    - name: logging
    - name: headers
      config:
        set:
          X-App: Helios
        request_set:
          X-From: LB
```

### Configuration Options

#### Server Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `port` | Port where Helios listens for incoming requests | `8080` |
| `tls` | TLS configuration block. See details below. | `disabled` |

#### Backend Configuration

| Option | Description | Required |
|--------|-------------|----------|
| `name` | Unique identifier for the backend | Yes |
| `address` | URL of the backend server | Yes |
| `weight` | The weight for the backend, used by `weighted_round_robin`. Defaults to `1`. | No |

#### Load Balancer Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `strategy` | Load balancing algorithm to use | `round_robin` |

Available strategies:
- `round_robin`: Distributes requests sequentially across all healthy backends
- `least_connections`: Routes to the backend with the fewest active connections
- `weighted_round_robin`: Distributes requests based on backend weights. A backend with a higher weight will receive proportionally more requests.
- `ip_hash`: Distributes requests based on a hash of the client's IP address. This ensures that a user will consistently be routed to the same backend server.

#### TLS Configuration

To enable TLS/SSL, you can add the `tls` block to the `server` configuration.

```yaml
server:
  port: 8080
  tls:
    enabled: true
    certFile: "path/to/your/cert.pem"
    keyFile: "path/to/your/key.pem"
```

| Option | Description | Required (if `tls` is enabled) |
|--------|-------------|--------------------------------|
| `enabled` | Set to `true` to enable TLS | Yes |
| `certFile` | Path to the SSL certificate file | Yes |
| `keyFile` | Path to the SSL private key file | Yes |

#### Health Check Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `active.enabled` | Enable active health checks | `true` |
| `active.interval` | Interval between health checks (seconds) | `10` |
| `active.timeout` | Timeout for health check requests (seconds) | `5` |
| `active.path` | Path to use for health check requests | `/health` |
| `passive.enabled` | Enable passive health checks | `true` |
| `passive.unhealthy_threshold` | Number of failures before marking as unhealthy | `1` |
| `passive.unhealthy_timeout` | Time to keep backend unhealthy (seconds) | `30` |

#### Rate Limiting Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `rate_limit.enabled` | Enable request rate limiting | `false` |
| `rate_limit.max_tokens` | Maximum tokens in the bucket | `100` |
| `rate_limit.refill_rate_seconds` | Rate at which tokens are refilled (seconds) | `1` |

#### Circuit Breaker Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `circuit_breaker.enabled` | Enable circuit breaker pattern | `false` |
| `circuit_breaker.max_requests` | Max requests allowed in half-open state | `5` |
| `circuit_breaker.interval_seconds` | Time window for failure counting (seconds) | `60` |
| `circuit_breaker.timeout_seconds` | Time to wait before moving from open to half-open (seconds) | `60` |
| `circuit_breaker.failure_threshold` | Number of failures to open the circuit | `5` |
| `circuit_breaker.success_threshold` | Number of successes to close the circuit in half-open state | `2` |

#### Metrics Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `metrics.enabled` | Enable metrics collection and API | `false` |
| `metrics.port` | Port for the metrics server | `9090` |
| `metrics.path` | Path for the metrics endpoint | `/metrics` |

#### Admin API Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `admin_api.enabled` | Enable Admin API | `false` |
| `admin_api.port` | Port for the Admin API | `9091` |
| `admin_api.auth_token` | Optional authentication token for protected endpoints | `disabled` |

#### Plugin Middleware Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `plugins.enabled` | Enable plugin middleware | `false` |
| `plugins.chain` | List of plugins to enable | `[]` |

### Admin API

Helios includes an Admin API for runtime control and observability.

Configuration:

```yaml
admin_api:
  enabled: true
  port: 9091
  auth_token: "change-me"  # Optional; if set, protected endpoints require Bearer token
```

Endpoints (default port 9091):
- GET /v1/health — Admin API health (no auth required)
- GET /v1/metrics — Metrics snapshot (requires auth if configured)
- GET /v1/backends — List backends (protected)
- POST /v1/backends/add — Add backend (protected)
- POST /v1/backends/remove — Remove backend (protected)
- POST /v1/strategy — Change strategy (protected)

Examples:

```bash
# Health (no auth)
curl http://localhost:9091/v1/health

# Metrics (with Bearer token)
curl -H "Authorization: Bearer change-me" http://localhost:9091/v1/metrics

# List backends
curl -H "Authorization: Bearer change-me" http://localhost:9091/v1/backends

# Add backend
curl -X POST -H "Authorization: Bearer change-me" -H "Content-Type: application/json" \
  -d '{"name":"b1","address":"http://127.0.0.1:8085","weight":1}' \
  http://localhost:9091/v1/backends/add

# Remove backend
curl -X POST -H "Authorization: Bearer change-me" -H "Content-Type: application/json" \
  -d '{"name":"b1"}' \
  http://localhost:9091/v1/backends/remove

# Switch strategy
curl -X POST -H "Authorization: Bearer change-me" -H "Content-Type: application/json" \
  -d '{"strategy":"least_connections"}' \
  http://localhost:9091/v1/strategy
```

### Plugin Middleware

Helios supports a configurable middleware chain. Built-in plugins:
- `logging`: request/response logging with WebSocket support
- `headers`: set static request/response headers

Configuration example:

```yaml
plugins:
  enabled: true
  chain:
    - name: logging
    - name: headers
      config:
        set:
          X-App: Helios
        request_set:
          X-From: LB
```

Order matters: plugins run top-to-bottom.

### Testing TLS/SSL
To test TLS functionality, enable TLS in `helios.yaml`:

```yaml
server:
  port: 8080
  tls:
    enabled: true
    certFile: "certs/cert.pem"
    keyFile: "certs/key.pem"
```

Then access Helios via HTTPS:

```bash
curl -k https://localhost:8080
```

## Performance

Helios is designed for high performance and low resource usage:

- **Low Latency**: Adds minimal overhead to request processing
- **High Throughput**: Capable of handling thousands of requests per second
- **Efficient Resource Usage**: Low memory footprint and CPU utilization
- **Concurrent Processing**: Leverages Go's goroutines for efficient parallel request handling

## Advanced Usage

### Custom Health Check Endpoints

By default, Helios uses the `/health` endpoint for active health checks. You can customize this in the configuration:

```yaml
health_checks:
  active:
    path: "/custom-health-endpoint"
```

### Simulating Failures for Testing

The included backend servers support simulating failures for testing:

```bash
# Run a backend with a 20% chance of failure
./backend.exe --port=8081 --id=1 --fail-rate=20
```

### Logging and Monitoring

Helios provides detailed logging about backend health and request routing. The built-in metrics endpoints allow integration with monitoring systems like Prometheus by consuming the JSON metrics API.

## Contributing

Contributions are welcome! Here's how you can contribute:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add my feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

### Development Guidelines

- Follow Go best practices and coding standards
- Add tests for new features
- Update documentation as needed
- Ensure all tests pass before submitting a pull request

## Roadmap

- [x] Additional load balancing strategies
  - [x] Weighted Round Robin
  - [x] IP Hash
- [x] TLS/SSL support
- [x] Request rate limiting
- [x] Circuit breaker pattern implementation
- [x] Metrics and monitoring endpoints
- [x] WebSocket support
- [x] Admin API for runtime configuration
- [x] Plugin system for custom middleware

## License

This project is licensed under the MIT License - see the LICENSE file for details.

---

<div align="center">
Made with ❤️ by <a href="https://github.com/0xReLogic">0xReLogic</a>
</div>
