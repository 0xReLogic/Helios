package loadbalancer

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/gorilla/websocket"
)

// Helper to set up a backend echo server for websocket tests
func setupWebSocketTestBackend() *httptest.Server {
	upgrader := websocket.Upgrader{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Printf("ws upgrade error: %v", err)
				return
			}
			defer conn.Close()
			for {
				mt, message, err := conn.ReadMessage()
				if err != nil {
					break // Client closed connection
				}
				if err := conn.WriteMessage(mt, message); err != nil {
					break // Client closed connection
				}
			}
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})
	return httptest.NewServer(handler)
}

func TestWebSocketProxy(t *testing.T) {
	// 1. Setup backend server
	backendServer := setupWebSocketTestBackend()
	defer backendServer.Close()

	// 2. Setup Helios load balancer
	cfg := &config.Config{
		Backends: []config.BackendConfig{
			{Name: "ws-backend", Address: backendServer.URL},
		},
		LoadBalancer: config.LoadBalancerConfig{
			Strategy: "round_robin",
		},
		HealthChecks: config.HealthChecksConfig{
			Active: config.ActiveHealthCheckConfig{
				Enabled: false, // Disable active health checks for this test
			},
		},
	}

	lb, err := NewLoadBalancer(cfg)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	proxyServer := httptest.NewServer(lb)
	defer proxyServer.Close()

	// 3. Connect to the proxy via WebSocket
	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(proxyServer.URL, "http") + "/ws"

	// Dial the websocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer conn.Close()

	// 4. Send a message and check for echo
	testMessage := "Hello, WebSocket!"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(testMessage)); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// 5. Read the echoed message
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	_, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(p) != testMessage {
		t.Errorf("Expected message '%s', but got '%s'", testMessage, string(p))
	}

	log.Println("WebSocket test successful!")
}
