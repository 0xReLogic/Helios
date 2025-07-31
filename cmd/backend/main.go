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
	flag.Parse()

	// Define HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Backend received request: %s %s", r.Method, r.URL.Path)
		fmt.Fprintf(w, "Hello from Backend Server! You requested: %s\n", r.URL.Path)
	})

	// Add a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Backend server starting on port %d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Backend server failed: %v", err)
	}
}
