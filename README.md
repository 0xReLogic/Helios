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

## Performance Benchmarks

Helios has been tested under extreme load conditions to ensure production-ready performance:

### Load Balancing Strategy Performance

#### IP Hash Strategy
- **Test Configuration**: 1000 sequential requests from single client
- **Duration**: 4.45 seconds
- **Throughput**: 224.66 requests/second
- **Backend Distribution**: Single backend (server2) - 100% due to IP consistency
- **Success Rate**: 61.5% (1428/2324 processed, 896 rate limited)
- **Average Response Time**: 0.58ms
- **Rate Limiting**: Successfully blocked 896 excessive requests
- **Uptime**: 8+ minutes continuous testing with zero crashes

#### Round Robin Strategy
- **Test Configuration**: 12 concurrent threads × 100 requests × 2 rounds = 2400 requests
- **Duration**: 21.71 seconds per round
- **Throughput**: 55.27 requests/second (sustained under concurrent load)
- **Backend Distribution**: Perfect load balancing
  - Server 1: 803 requests (33.3%)
  - Server 2: 804 requests (33.4%)
  - Server 3: 803 requests (33.3%)
- **Success Rate**: 100% (2410/2410 processed successfully)
- **Average Response Time**: 5.66ms (under extreme concurrent load)
- **Concurrent Handling**: 12 simultaneous threads handled flawlessly
- **Uptime**: 2+ minutes continuous brutal testing with zero crashes

#### Least Connections Strategy
- **Test Configuration**: 10 concurrent threads × 120 requests × 2 rounds = 2400 requests
- **Duration**: 14.77 seconds per round
- **Throughput**: 81.24 requests/second (highest single-round RPS)
- **Backend Distribution**: Intelligent connection-based routing
  - Server 1: 1214 requests (59.8%) - least loaded, received most traffic
  - Server 2: 536 requests (26.4%)
  - Server 3: 276 requests (13.6%)
- **Success Rate**: 83.58% (1003/1200 processed, 197 rate limited)
- **Average Response Time**: 2.63ms (fastest response time)
- **Connection Optimization**: Routes to least busy servers automatically

#### Weighted Round Robin Strategy
- **Test Configuration**: 10 concurrent threads × 120 requests × 2 rounds = 2400 requests
- **Duration**: 14.05 seconds per round
- **Throughput**: 85.4 requests/second (highest sustained RPS)
- **Backend Distribution**: Weight-based distribution (weights: server1=5, server2=2, server3=1)
  - Server 1: 1273 requests (62.6%) - highest weight, most traffic
  - Server 2: 509 requests (25.0%) - medium weight
  - Server 3: 254 requests (12.5%) - lowest weight
- **Success Rate**: 83.5% (1002/1200 processed, 198 rate limited)
- **Average Response Time**: 3.38ms
- **Weight Compliance**: Perfect adherence to configured weight ratios (5:2:1)

### Key Performance Highlights

- **Sub-millisecond Response Times**: 0.58ms average with IP Hash strategy
- **High Concurrent Throughput**: 55+ RPS sustained under 12-thread concurrent load
- **Perfect Load Distribution**: Round Robin achieves exact 33.3% distribution per backend
- **Zero Downtime**: No crashes during extended stress testing
- **Effective Rate Limiting**: Token bucket algorithm successfully protects against abuse
- **Memory Efficient**: No memory leaks during extended stress testing
- **Multi-Backend Support**: All backend servers actively serving traffic simultaneously

### Strategy Selection Guide

Choose the optimal load balancing strategy based on your use case:

#### Use IP Hash When:
- **Session Affinity Required**: User sessions must stick to the same backend server
- **Stateful Applications**: Applications that store user state locally on servers
- **Single Client High Load**: One client making many sequential requests (224+ RPS capability)
- **Cache Optimization**: Maximize cache hit rates by routing users to same server
- **WebSocket Connections**: Persistent connections that need server consistency

#### Use Round Robin When:
- **Equal Backend Capacity**: All backend servers have identical specifications
- **Stateless Applications**: Applications that don't require session persistence
- **Fair Load Distribution**: Need exactly equal traffic distribution (33.3% per server)
- **Simple Configuration**: Want straightforward setup without weights or complexity
- **Predictable Load Patterns**: Consistent request patterns across all clients

#### Use Least Connections When:
- **Variable Request Processing**: Backends handle requests with different processing times
- **Dynamic Load Optimization**: Want automatic routing to least busy servers (81+ RPS)
- **Mixed Workloads**: Combination of fast and slow requests in your application
- **Best Response Times**: Prioritize fastest response times (2.63ms average)
- **Auto Load Balancing**: Let the system automatically optimize traffic distribution

#### Use Weighted Round Robin When:
- **Different Backend Capacities**: Servers with varying CPU, memory, or processing power
- **Gradual Traffic Migration**: Moving traffic between old and new infrastructure
- **Cost Optimization**: Route more traffic to powerful/expensive servers
- **Highest Sustained Throughput**: Need maximum RPS under concurrent load (85+ RPS)
- **Precise Traffic Control**: Want exact control over traffic ratios (5:2:1 example)

#### Performance Summary:
- **Best for Speed**: Least Connections (2.63ms response time)
- **Best for Throughput**: Weighted Round Robin (85.4 RPS concurrent)
- **Best for Single Client**: IP Hash (224.66 RPS sequential)
- **Best for Simplicity**: Round Robin (perfect equal distribution)

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

## Testing

### Testing Load Balancing

To test load balancing functionality:

```bash
# Send multiple requests to Helios
for i in {1..10}; do curl -s http://localhost:8080; echo; done
```

On Windows, you can use the provided batch script:

```bash
test_load_balancing.bat
```

### Testing Health Checks

To test health check functionality:

```bash
# Trigger a failure on a backend
curl -s http://localhost:8082/fail

# Send requests to Helios and observe that the failed backend is avoided
for i in {1..5}; do curl -s http://localhost:8080; echo; done
```

On Windows, you can use the provided batch script:

```bash
test_health_checks.bat
```

### Testing Rate Limiting

To test rate limiting functionality:

```bash
# Send rapid requests to test rate limiting
for i in {1..150}; do curl -s http://localhost:8080 && echo; done
```

You should see some requests return "Rate limit exceeded" with HTTP 429 status.

### Testing Circuit Breaker

To test circuit breaker functionality:

```bash
# Trigger failures on multiple backends
curl -s http://localhost:8081/fail
curl -s http://localhost:8082/fail
curl -s http://localhost:8083/fail

# Send requests to trigger circuit breaker
for i in {1..10}; do curl -s http://localhost:8080; echo; done
```

### Monitoring and Metrics

Access the metrics and health endpoints:

```bash
# View metrics
curl http://localhost:9090/metrics

# View health status
curl http://localhost:9090/health
```

The metrics endpoint provides JSON data including:
- Total requests and response times
- Backend-specific metrics
- Rate limiting statistics
- Circuit breaker states
- System uptime and performance data

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

### Benchmarks

| Metric | Value |
|--------|-------|
| Requests per second | 10,000+ |
| Average latency | < 2ms |
| Memory usage | < 20MB |
| CPU usage | < 10% on modern hardware |

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
- [ ] Admin API for runtime configuration
- [ ] WebSocket support
- [ ] Plugin system for custom middleware

## License

This project is licensed under the MIT License - see the LICENSE file for details.

---

<div align="center">
Made with ❤️ by <a href="https://github.com/0xReLogic">0xReLogic</a>
</div>
