package utils

import (
	"net/http"
	"testing"
)

// TestGetClientIP tests IP extraction from various HTTP headers
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For with single IP",
			xff:        "203.0.113.195",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For with multiple IPs",
			xff:        "203.0.113.195, 70.41.3.18, 150.172.238.178",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For with spaces",
			xff:        "  203.0.113.195  ,  70.41.3.18  ",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Real-IP when no XFF",
			xri:        "203.0.113.195",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "XFF takes precedence over X-Real-IP",
			xff:        "203.0.113.195",
			xri:        "70.41.3.18",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "RemoteAddr fallback with port",
			remoteAddr: "203.0.113.195:56789",
			expected:   "203.0.113.195",
		},
		{
			name:       "RemoteAddr fallback without port",
			remoteAddr: "203.0.113.195",
			expected:   "203.0.113.195",
		},
		{
			name:       "IPv6 address",
			remoteAddr: "[2001:db8::1]:8080",
			expected:   "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			got := GetClientIP(req)
			if got != tt.expected {
				t.Errorf("GetClientIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}
