# Helios

🚦 A high-performance, layer-7 HTTP reverse proxy and load balancer built with Go, designed for scalability and fault tolerance.

## Current Status

This project is in active development. Currently implementing Phase 2: Load Balancing functionality.

## Features

- ✅ HTTP reverse proxy
- ✅ Load balancing with Round Robin strategy
- 🔄 Health checks (passive and active) - Coming soon
- ✅ Configuration via YAML
- ✅ High performance and low resource usage

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

### Running the Test Backends

For testing purposes, simple backend servers are included. You can run them with different ports and IDs:

```
# Run multiple backend servers
go run ./cmd/backend/main.go --port=8081 --id=1
go run ./cmd/backend/main.go --port=8082 --id=2
go run ./cmd/backend/main.go --port=8083 --id=3
```

Or use the provided batch script:

```
start_backends.bat
```

## Configuration

Configuration is done via `helios.yaml`:

```yaml
server:
  port: 8080

backends:
  - name: "server1"
    address: "http://localhost:8081"
  - name: "server2"
    address: "http://localhost:8082"
  - name: "server3"
    address: "http://localhost:8083"

load_balancer:
  strategy: "round_robin"  # Currently only round_robin is supported
```

## Testing Load Balancing

To test that load balancing is working correctly, you can use the provided batch script:

```
test_load_balancing.bat
```

Or manually send multiple requests to the proxy:

```
curl http://localhost:8080
```

You should see responses from different backend servers as the requests are distributed using the round-robin strategy.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
