# Stage 1: Build
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o helios ./cmd/helios

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/helios .
COPY helios.yaml .
COPY certs/ certs/
EXPOSE 8080 9090 9091
CMD [ "./helios", "--config", "helios.yaml"]