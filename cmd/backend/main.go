package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Parse command line flags
	port := flag.Int("port", 8081, "Port to run the backend server on")
	serverID := flag.String("id", "1", "Server ID for identification")
	failRate := flag.Int("fail-rate", 0, "Percentage chance of simulating a failure (0-100)")
	flag.Parse()

	// Define HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Backend %s received request: %s %s", *serverID, r.Method, r.URL.Path)

		// Simulate random failures if fail-rate is set
		if *failRate > 0 && (rand.Intn(100) < *failRate) {
			log.Printf("Backend %s: Simulating a failure (status 500)", *serverID)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Simulated failure from Backend Server %s!\n", *serverID)
			return
		}

		fmt.Fprintf(w, "Hello from Backend Server %s! You requested: %s\n", *serverID, r.URL.Path)
	})

	// Add a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Simulate random failures if fail-rate is set
		if *failRate > 0 && (rand.Intn(100) < *failRate) {
			log.Printf("Backend %s: Health check failing (status 500)", *serverID)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Health check failed from Server %s", *serverID)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK from Server %s", *serverID)
	})

	// Add a fail endpoint for testing health checks
	http.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Backend %s: Manually failing for testing", *serverID)

		// Return 500 error to trigger health check failure
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Manual failure triggered on Server %s", *serverID)
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Backend server %s starting on port %d", *serverID, *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Backend server %s failed: %v", *serverID, err)
	}
}
