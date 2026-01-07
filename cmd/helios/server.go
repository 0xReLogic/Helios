package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/0xReLogic/Helios/internal/adminapi"
	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/plugins"
)

// setupMetricsServer starts the metrics HTTP server if enabled in config
func setupMetricsServer(cfg *config.Config, lb *loadbalancer.LoadBalancer) {
	if !cfg.Metrics.Enabled {
		return
	}

	metricsPort := cfg.Metrics.Port
	if metricsPort == 0 {
		metricsPort = 9090 // Default metrics port
	}

	metricsPath := cfg.Metrics.Path
	if metricsPath == "" {
		metricsPath = "/metrics" // Default metrics path
	}

	metricsCollector := lb.GetMetricsCollector()
	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc(metricsPath, metricsCollector.MetricsHandler())
	metricsMux.HandleFunc("/health", metricsCollector.HealthHandler())

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", metricsPort),
		Handler:      metricsMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start metrics server in background
	go func() {
		logger := logging.L()
		logger.Info().Int("port", metricsPort).Str("path", metricsPath).Msg("metrics server starting")
		logger.Info().Int("port", metricsPort).Str("url", fmt.Sprintf("http://localhost:%d%s", metricsPort, metricsPath)).Msg("metrics endpoint")
		logger.Info().Int("port", metricsPort).Str("url", fmt.Sprintf("http://localhost:%d/health", metricsPort)).Msg("metrics health endpoint")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("metrics server error")
		}
	}()
}

// setupAdminAPIServer starts the Admin API HTTP server if enabled in config
func setupAdminAPIServer(cfg *config.Config, lb *loadbalancer.LoadBalancer) {
	if !cfg.AdminAPI.Enabled {
		return
	}

	adminPort := cfg.AdminAPI.Port
	if adminPort == 0 {
		adminPort = 9091 // Default admin port
	}

	mc := lb.GetMetricsCollector()
	adminHandler := adminapi.NewMux(lb, cfg, mc)
	adminServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", adminPort),
		Handler:      adminHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger := logging.L()
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

// buildHandler constructs the HTTP handler with plugins and middleware
func buildHandler(cfg *config.Config, lb *loadbalancer.LoadBalancer) (http.Handler, error) {
	var handler http.Handler = lb
	logger := logging.L()

	// Apply plugin chain if enabled
	if cfg.Plugins.Enabled && len(cfg.Plugins.Chain) > 0 {
		chained, err := plugins.BuildChain(cfg.Plugins, handler)
		if err != nil {
			return nil, fmt.Errorf("failed to build plugin chain: %w", err)
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

	// Add request context middleware
	handler = logging.RequestContextMiddleware(cfg.Logging)(handler)

	return handler, nil
}

// createHTTPServer creates and configures the main HTTP server
func createHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

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

	// Configure TLS if enabled
	if cfg.Server.TLS.Enabled {
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
	}

	return server
}

// validateTLSFiles checks if TLS certificate and key files exist
func validateTLSFiles(cfg *config.Config) error {
	if !cfg.Server.TLS.Enabled {
		return nil
	}

	if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
		return fmt.Errorf("tls enabled but certificate or key not configured")
	}

	if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("tls certificate file not found: %s", cfg.Server.TLS.CertFile)
	}

	if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("tls key file not found: %s", cfg.Server.TLS.KeyFile)
	}

	return nil
}

// startHTTPServer starts the HTTP/HTTPS server in a goroutine
func startHTTPServer(server *http.Server, cfg *config.Config, serverErrors chan<- error) {
	logger := logging.L()

	readTimeout := time.Duration(cfg.Server.Timeouts.Read) * time.Second
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := time.Duration(cfg.Server.Timeouts.Write) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 15 * time.Second
	}
	idleTimeout := time.Duration(cfg.Server.Timeouts.Idle) * time.Second
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}

	go func() {
		if cfg.Server.TLS.Enabled {
			logger.Info().Msg("tls enabled")
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
}

// logStartupInfo logs server startup information
func logStartupInfo(cfg *config.Config) {
	logger := logging.L()

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
}

// shutdownGracefully performs graceful shutdown of the server and load balancer
func shutdownGracefully(server *http.Server, lb *loadbalancer.LoadBalancer, shutdownTimeout time.Duration) {
	logger := logging.L()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info().Dur("timeout", shutdownTimeout).Msg("shutting down server gracefully")

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("error during server shutdown")
		if closeErr := server.Close(); closeErr != nil {
			logger.Error().Err(closeErr).Msg("error closing server")
		}
	}

	// Stop load balancer
	lb.Stop()

	logger.Info().Msg("server shutdown complete")
}
