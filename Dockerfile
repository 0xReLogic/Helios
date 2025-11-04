# Stage 1: Build
FROM golang:1.20-alpine AS builder
WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files explicitly
COPY cmd/helios ./cmd/helios
COPY internal/ ./internal
COPY helios.docker.yaml .
COPY certs/ certs/

# Build the binary
RUN go build -o helios ./cmd/helios

# Stage 2: Runtime
FROM alpine:3.18
WORKDIR /app

# Copy built binary and config
COPY --from=builder /app/helios .
COPY helios.docker.yaml .
COPY certs/ certs/

# Create non-root user
RUN addgroup -S helios && adduser -S helios -G helios
USER helios

EXPOSE 8080 9090 9091
CMD ["./helios", "--config=helios.docker.yaml"]

