package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/logging"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.WithContext(r.Context()).Error().Err(err).Msg("failed to upgrade to websocket")
		return
	}
	defer conn.Close()

	logging.WithContext(r.Context()).Info().Msg("websocket connection established")

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logging.WithContext(r.Context()).Error().Err(err).Msg("websocket read error")
			return
		}
		logging.WithContext(r.Context()).Info().Bytes("message", p).Msg("websocket message received")
		if err := conn.WriteMessage(messageType, p); err != nil {
			logging.WithContext(r.Context()).Error().Err(err).Msg("websocket write error")
			return
		}
	}
}

func main() {
	logging.Init(config.LoggingConfig{Format: "text"})
	logger := logging.L()

	// No need to initialize random seed in Go 1.20+
	// rand.Seed is deprecated since Go 1.20

	// Parse command line flags
	port := flag.Int("port", 8081, "Port to run the backend server on")
	serverID := flag.String("id", "1", "Server ID for identification")
	failRate := flag.Int("fail-rate", 0, "Percentage chance of simulating a failure (0-100)")
	flag.Parse()

	// Define HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestLogger := logging.WithContext(r.Context()).With().Str("backend", *serverID).Logger()
		requestLogger.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

		// Simulate random failures if fail-rate is set
		if *failRate > 0 && (rand.Intn(100) < *failRate) {
			requestLogger.Warn().Msg("simulating failure")
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
			logging.WithContext(r.Context()).Warn().Str("backend", *serverID).Msg("health check failing")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Health check failed from Server %s", *serverID)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK from Server %s", *serverID)
	})

	// Add a fail endpoint for testing health checks
	http.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		logging.WithContext(r.Context()).Warn().Str("backend", *serverID).Msg("manual failure triggered")

		// Return 500 error to trigger health check failure
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Manual failure triggered on Server %s", *serverID)
	})

	// Add a websocket echo endpoint
	http.HandleFunc("/ws", handleWebSocket)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	logger.Info().Str("backend", *serverID).Int("port", *port).Msg("backend server starting")
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatal().Err(err).Str("backend", *serverID).Msg("backend server failed")
	}
}
