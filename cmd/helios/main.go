package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/adminapi"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/plugins"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("helios.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create load balancer
	lb, err := loadbalancer.NewLoadBalancer(cfg)
	if err != nil {
		log.Fatalf("Failed to create load balancer: %v", err)
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
			log.Printf("Metrics server starting on port %d", metricsPort)
			log.Printf("Metrics endpoint: http://localhost:%d%s", metricsPort, metricsPath)
			log.Printf("Health endpoint: http://localhost:%d/health", metricsPort)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Metrics server error: %v", err)
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
			log.Printf("Admin API server starting on port %d", adminPort)
			if cfg.AdminAPI.AuthToken != "" {
				log.Printf("Admin API authentication: Bearer token enabled")
			} else {
				log.Printf("Admin API authentication: disabled")
			}
			if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Admin API server error: %v", err)
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
			log.Fatalf("Failed to build plugin chain: %v", err)
		}
		handler = chained

		// Log configured plugin names in order
		names := make([]string, 0, len(cfg.Plugins.Chain))
		for _, p := range cfg.Plugins.Chain {
			names = append(names, p.Name)
		}
		log.Printf("Plugins enabled: %v", names)
	} else {
		log.Printf("Plugins disabled")
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server
	log.Printf("Helios load balancer starting on port %d", cfg.Server.Port)
	log.Printf("Load balancing strategy: %s", cfg.LoadBalancer.Strategy)
	log.Printf("Backend servers:")
	for _, backend := range cfg.Backends {
		log.Printf("  - %s (%s)", backend.Name, backend.Address)
	}

	// Log health check configuration
	log.Printf("Health check configuration:")
	if cfg.HealthChecks.Active.Enabled {
		log.Printf("  - Active health checks: Enabled (interval: %ds, timeout: %ds, path: %s)",
			cfg.HealthChecks.Active.Interval,
			cfg.HealthChecks.Active.Timeout,
			cfg.HealthChecks.Active.Path)
	} else {
		log.Printf("  - Active health checks: Disabled")
	}

	if cfg.HealthChecks.Passive.Enabled {
		log.Printf("  - Passive health checks: Enabled (threshold: %d, timeout: %ds)",
			cfg.HealthChecks.Passive.UnhealthyThreshold,
			cfg.HealthChecks.Passive.UnhealthyTimeout)
	} else {
		log.Printf("  - Passive health checks: Disabled")
	}

	// Start the server with or without TLS based on configuration
	if cfg.Server.TLS.Enabled {
		log.Println("TLS is enabled.")

		// Validate that cert and key files are specified
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			log.Fatal("TLS is enabled, but certFile or keyFile is not specified in the configuration.")
		}

		// Check if the certificate and key files exist
		if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
			log.Fatalf("TLS certificate file not found: %s", cfg.Server.TLS.CertFile)
		}
		if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
			log.Fatalf("TLS key file not found: %s", cfg.Server.TLS.KeyFile)
		}

		log.Printf("Listening for HTTPS on port %d", cfg.Server.Port)
		if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
			log.Fatalf("Failed to start TLS server: %v", err)
		}
	} else {
		log.Printf("Listening for HTTP on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}
