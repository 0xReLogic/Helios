# Stage 1: Build
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY go.mod go.sum ./
COPY cmd/helios ./cmd/helios
COPY internal/ ./internal
COPY helios.yaml .
COPY certs/ certs/

RUN go build -o helios ./cmd/helios

# Stage 2: Runtime
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/helios .
COPY helios.yaml .
COPY certs/ certs/
RUN addgroup -S helios && adduser -S helios -G helios
USER helios
EXPOSE 8080 9090 9091
CMD ["./helios", "--config", "helios.yaml"]
