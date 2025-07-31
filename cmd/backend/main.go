package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8081, "Port to run the backend server on")
	serverID := flag.String("id", "1", "Server ID for identification")
	flag.Parse()

	// Define HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Backend %s received request: %s %s", *serverID, r.Method, r.URL.Path)
		fmt.Fprintf(w, "Hello from Backend Server %s! You requested: %s\n", *serverID, r.URL.Path)
	})

	// Add a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK from Server %s", *serverID)
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Backend server %s starting on port %d", *serverID, *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Backend server %s failed: %v", *serverID, err)
	}
}
