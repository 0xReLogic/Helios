package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}
		log.Printf("Received message: %s", p)
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

func main() {
	// No need to initialize random seed in Go 1.20+
	// rand.Seed is deprecated since Go 1.20

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

	// Add a websocket echo endpoint
	http.HandleFunc("/ws", handleWebSocket)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Backend server %s starting on port %d", *serverID, *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Backend server %s failed: %v", *serverID, err)
	}
}
