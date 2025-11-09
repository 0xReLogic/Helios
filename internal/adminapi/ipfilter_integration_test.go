package adminapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/metrics"
)

func TestAdminAPI_WithIPFilter_AllowList(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()

	cfg := &config.Config{
		AdminAPI: config.AdminAPIConfig{
			Enabled:     true,
			Port:        9091,
			AuthToken:   "secret",
			IPAllowList: []string{"192.168.1.0/24", "127.0.0.1"},
			IPDenyList:  []string{},
		},
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: false},
		},
	}

	mux := NewMux(lb, cfg, mc)

	tests := []struct {
		name           string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "allowed IP in subnet",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "allowed localhost",
			clientIP:       "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocked IP not in allow list",
			clientIP:       "10.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "blocked external IP",
			clientIP:       "203.0.113.50",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAdminAPI_WithIPFilter_DenyList(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()

	cfg := &config.Config{
		AdminAPI: config.AdminAPIConfig{
			Enabled:     true,
			Port:        9091,
			AuthToken:   "secret",
			IPAllowList: []string{}, // Empty allow list = allow all except denied
			IPDenyList:  []string{"203.0.113.0/24", "10.0.0.1"},
		},
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: false},
		},
	}

	mux := NewMux(lb, cfg, mc)

	tests := []struct {
		name           string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "allowed IP not in deny list",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocked IP in deny subnet",
			clientIP:       "203.0.113.50",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "blocked specific IP",
			clientIP:       "10.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAdminAPI_WithIPFilter_DenyTakesPrecedence(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()

	cfg := &config.Config{
		AdminAPI: config.AdminAPIConfig{
			Enabled:     true,
			Port:        9091,
			AuthToken:   "secret",
			IPAllowList: []string{"192.168.1.0/24"}, // Allow entire subnet
			IPDenyList:  []string{"192.168.1.100"},  // But deny specific IP
		},
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: false},
		},
	}

	mux := NewMux(lb, cfg, mc)

	tests := []struct {
		name           string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "allowed IP in subnet",
			clientIP:       "192.168.1.50",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "denied IP despite being in allow list",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAdminAPI_WithoutIPFilter(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()

	cfg := &config.Config{
		AdminAPI: config.AdminAPIConfig{
			Enabled:     true,
			Port:        9091,
			AuthToken:   "secret",
			IPAllowList: []string{}, // No IP filtering
			IPDenyList:  []string{},
		},
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: false},
		},
	}

	mux := NewMux(lb, cfg, mc)

	// All IPs should be allowed when no filter is configured
	testIPs := []string{"192.168.1.100", "10.0.0.1", "203.0.113.50", "127.0.0.1"}

	for _, ip := range testIPs {
		t.Run("allow_"+ip, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
			req.RemoteAddr = ip + ":12345"

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200 for IP %s, got %d", ip, rec.Code)
			}
		})
	}
}
