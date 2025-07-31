# Helios

🚦 A high-performance, layer-7 HTTP reverse proxy and load balancer built with Go, designed for scalability and fault tolerance.

## Current Status

This project is in active development. Currently implementing Phase 1: Core Reverse Proxy functionality.

## Features (Planned)

- HTTP reverse proxy
- Load balancing with multiple strategies
- Health checks (passive and active)
- Configuration via YAML
- High performance and low resource usage

## Getting Started

### Prerequisites

- Go 1.18 or higher

### Running the Reverse Proxy

1. Clone the repository:
   ```
   git clone https://github.com/0xReLogic/Helios.git
   cd Helios
   ```

2. Build the project:
   ```
   go build -o helios ./cmd/helios
   ```

3. Run the proxy:
   ```
   ./helios
   ```

### Running the Test Backend

For testing purposes, a simple backend server is included:

```
go run ./cmd/backend/main.go
```

## Configuration

Configuration is done via `helios.yaml`:

```yaml
server:
  port: 8080

backend:
  address: "http://localhost:8081"
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
