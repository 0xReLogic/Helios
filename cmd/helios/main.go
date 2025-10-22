package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/0xReLogic/Helios/internal/adminapi"
	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/plugins"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("helios.yaml")
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

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

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

	// Start the server with or without TLS based on configuration
	if cfg.Server.TLS.Enabled {
		logger.Info().Msg("tls enabled")

		// Validate that cert and key files are specified
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			logger.Fatal().Msg("tls enabled but certificate or key not configured")
		}

		// Check if the certificate and key files exist
		if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
			logger.Fatal().Str("cert_file", cfg.Server.TLS.CertFile).Msg("tls certificate file not found")
		}
		if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
			logger.Fatal().Str("key_file", cfg.Server.TLS.KeyFile).Msg("tls key file not found")
		}

		logger.Info().Int("port", cfg.Server.Port).Msg("listening for https")
		if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
			logger.Fatal().Err(err).Msg("failed to start tls server")
		}
	} else {
		logger.Info().Int("port", cfg.Server.Port).Msg("listening for http")
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal().Err(err).Msg("failed to start http server")
		}
	}
}
