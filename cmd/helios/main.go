package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/proxy"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("helios.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create reverse proxy
	reverseProxy, err := proxy.NewReverseProxy(cfg)
	if err != nil {
		log.Fatalf("Failed to create reverse proxy: %v", err)
	}

	// Setup HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: reverseProxy,
	}

	// Start server
	log.Printf("Helios reverse proxy starting on port %d", cfg.Server.Port)
	log.Printf("Proxying requests to backend: %s", cfg.Backend.Address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}
