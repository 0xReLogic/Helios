package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0xReLogic/Helios/internal/adminapi"
	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/plugins"
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

	// Setup metrics server if enabled
	if cfg.Metrics.Enabled {
		metricsPort := cfg.Metrics.Port
		if metricsPort == 0 {
			metricsPort = 9090 // Default metrics port
		}

		metricsPath := cfg.Metrics.Path
		if metricsPath == "" {
			metricsPath = "/metrics" // Default metrics path
		}

		metricsCollector := lb.GetMetricsCollector()

		// Create metrics server
		metricsMux := http.NewServeMux()
		metricsMux.HandleFunc(metricsPath, metricsCollector.MetricsHandler())
		metricsMux.HandleFunc("/health", metricsCollector.HealthHandler())

		metricsServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", metricsPort),
			Handler: metricsMux,
		}

		// Start metrics server in background
		go func() {
			logger.Info().Int("port", metricsPort).Str("path", metricsPath).Msg("metrics server starting")
			logger.Info().Int("port", metricsPort).Str("url", fmt.Sprintf("http://localhost:%d%s", metricsPort, metricsPath)).Msg("metrics endpoint")
			logger.Info().Int("port", metricsPort).Str("url", fmt.Sprintf("http://localhost:%d/health", metricsPort)).Msg("metrics health endpoint")
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error().Err(err).Msg("metrics server error")
			}
		}()
	}

	// Setup Admin API server if enabled
	if cfg.AdminAPI.Enabled {
		adminPort := cfg.AdminAPI.Port
		if adminPort == 0 {
			adminPort = 9091 // Default admin port
		}

		mc := lb.GetMetricsCollector()
		adminHandler := adminapi.NewMux(lb, cfg.AdminAPI.AuthToken, mc)
		adminServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", adminPort),
			Handler: adminHandler,
		}

		go func() {
			logger.Info().Int("port", adminPort).Msg("admin api server starting")
			if cfg.AdminAPI.AuthToken != "" {
				logger.Info().Msg("admin api authentication enabled")
			} else {
				logger.Info().Msg("admin api authentication disabled")
			}
			if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error().Err(err).Msg("admin api server error")
			}
		}()
	}

	// Setup HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	// Base handler is the load balancer
	var handler http.Handler = lb
	// Apply plugin chain if enabled
	if cfg.Plugins.Enabled && len(cfg.Plugins.Chain) > 0 {
		chained, err := plugins.BuildChain(cfg.Plugins, handler)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to build plugin chain")
		}
		handler = chained

		// Log configured plugin names in order
		names := make([]string, 0, len(cfg.Plugins.Chain))
		for _, p := range cfg.Plugins.Chain {
			names = append(names, p.Name)
		}
		logger.Info().Strs("plugins", names).Msg("plugins enabled")
	} else {
		logger.Info().Msg("plugins disabled")
	}

	handler = logging.RequestContextMiddleware(cfg.Logging)(handler)

	// Apply timeout configurations with smart defaults
	readTimeout := time.Duration(cfg.Server.Timeouts.Read) * time.Second
	if readTimeout == 0 {
		readTimeout = 15 * time.Second // Default: protect against slow-read attacks
	}
	writeTimeout := time.Duration(cfg.Server.Timeouts.Write) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 15 * time.Second // Default: prevent slow writes
	}
	idleTimeout := time.Duration(cfg.Server.Timeouts.Idle) * time.Second
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second // Default: keep-alive timeout
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Setup graceful shutdown
	shutdownTimeout := time.Duration(cfg.Server.Timeouts.Shutdown) * time.Second
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second // Default: 30s graceful shutdown
	}

	// Channel for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		// Start the server with or without TLS based on configuration
		if cfg.Server.TLS.Enabled {
			logger.Info().Msg("tls enabled")

			// Validate that cert and key files are specified
			if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
				serverErrors <- fmt.Errorf("tls enabled but certificate or key not configured")
				return
			}

			// Check if the certificate and key files exist
			if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
				serverErrors <- fmt.Errorf("tls certificate file not found: %s", cfg.Server.TLS.CertFile)
				return
			}
			if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
				serverErrors <- fmt.Errorf("tls key file not found: %s", cfg.Server.TLS.KeyFile)
				return
			}

			// Configure TLS with minimum version 1.2 for security
			server.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				},
				PreferServerCipherSuites: true,
			}

			logger.Info().Int("port", cfg.Server.Port).Msg("listening for https")
			logger.Info().
				Str("min_tls_version", "1.2").
				Dur("read_timeout", readTimeout).
				Dur("write_timeout", writeTimeout).
				Dur("idle_timeout", idleTimeout).
				Msg("server timeouts configured")

			serverErrors <- server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			logger.Info().Int("port", cfg.Server.Port).Msg("listening for http")
			logger.Info().
				Dur("read_timeout", readTimeout).
				Dur("write_timeout", writeTimeout).
				Dur("idle_timeout", idleTimeout).
				Msg("server timeouts configured")

			serverErrors <- server.ListenAndServe()
		}
	}()

	// Start server
	logger.Info().Int("port", cfg.Server.Port).Msg("helios load balancer starting")
	logger.Info().Str("strategy", cfg.LoadBalancer.Strategy).Msg("load balancing strategy")
	logger.Info().Msg("configured backend servers")
	for _, backend := range cfg.Backends {
		logger.Info().Str("backend", backend.Name).Str("address", backend.Address).Msg("backend registered")
	}

	// Log health check configuration
	logger.Info().Msg("health check configuration")
	if cfg.HealthChecks.Active.Enabled {
		logger.Info().Int("interval_seconds", cfg.HealthChecks.Active.Interval).
			Int("timeout_seconds", cfg.HealthChecks.Active.Timeout).
			Str("path", cfg.HealthChecks.Active.Path).
			Msg("active health checks enabled")
	} else {
		logger.Info().Msg("active health checks disabled")
	}

	if cfg.HealthChecks.Passive.Enabled {
		logger.Info().Int("threshold", cfg.HealthChecks.Passive.UnhealthyThreshold).
			Int("timeout_seconds", cfg.HealthChecks.Passive.UnhealthyTimeout).
			Msg("passive health checks enabled")
	} else {
		logger.Info().Msg("passive health checks disabled")
	}

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		logger.Fatal().Err(err).Msg("server failed to start")
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		logger.Info().Dur("timeout", shutdownTimeout).Msg("shutting down server gracefully")

		// Shutdown HTTP server
		if err := server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("error during server shutdown")
			server.Close()
		}

		// Stop load balancer
		lb.Stop()

		logger.Info().Msg("server shutdown complete")
	}
}
