package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/logging"
)

func main() {
	// Load configuration
	configPath := flag.String("config", "helios.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logging.L().Fatal().Err(err).Msg("failed to load configuration")
	}

	logging.Init(cfg.Logging)
	logger := logging.L()

	// Create load balancer
	lb, err := loadbalancer.NewLoadBalancer(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create load balancer")
	}

	// Setup ancillary servers
	setupMetricsServer(cfg, lb)
	setupAdminAPIServer(cfg, lb)

	// Build HTTP handler with plugins
	handler, err := buildHandler(cfg, lb)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build handler")
	}

	// Validate TLS configuration
	if err := validateTLSFiles(cfg); err != nil {
		logger.Fatal().Err(err).Msg("tls validation failed")
	}

	// Create and configure HTTP server
	server := createHTTPServer(cfg, handler)

	// Determine shutdown timeout
	shutdownTimeout := time.Duration(cfg.Server.Timeouts.Shutdown) * time.Second
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second // Default: 30s graceful shutdown
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server
	serverErrors := make(chan error, 1)
	startHTTPServer(server, cfg, serverErrors)

	// Log startup information
	logStartupInfo(cfg)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		logger.Fatal().Err(err).Msg("server failed to start")
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
		shutdownGracefully(server, lb, shutdownTimeout)
	}
}
