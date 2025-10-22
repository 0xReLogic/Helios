package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/metrics"
)

// NewMux creates an HTTP handler for the Admin API
func NewMux(lb *loadbalancer.LoadBalancer, token string, mc *metrics.MetricsCollector) http.Handler {
	mux := http.NewServeMux()

	// Auth middleware
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			authz := r.Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") || strings.TrimPrefix(authz, "Bearer ") != token {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Health endpoint (no auth)
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Metrics endpoint (auth if token set)
	mux.Handle("/v1/metrics", auth(http.HandlerFunc(mc.MetricsHandler())))

	// List backends
	mux.Handle("/v1/backends", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		backends := lb.ListBackends()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backends)
	})))

	// Add backend
	mux.Handle("/v1/backends/add", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req config.BackendConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Address == "" {
			http.Error(w, "name and address are required", http.StatusBadRequest)
			return
		}
		if err := lb.AddBackend(req); err != nil {
			http.Error(w, fmt.Sprintf("failed to add backend: %v", err), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("added"))
	})))

	// Remove backend
	mux.Handle("/v1/backends/remove", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		type rmReq struct {
			Name string `json:"name"`
		}
		var req rmReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		lb.RemoveBackend(req.Name)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("removed"))
	})))

	// Change strategy
	mux.Handle("/v1/strategy", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		type setReq struct {
			Strategy string `json:"strategy"`
		}
		var req setReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
			return
		}
		if req.Strategy == "" {
			http.Error(w, "strategy is required", http.StatusBadRequest)
			return
		}
		if err := lb.SetStrategy(req.Strategy); err != nil {
			http.Error(w, fmt.Sprintf("failed to set strategy: %v", err), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("updated"))
	})))

	logging.L().Info().Msg("admin api mux initialized")
	return mux
}
