package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
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

	// Setup HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: lb,
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
