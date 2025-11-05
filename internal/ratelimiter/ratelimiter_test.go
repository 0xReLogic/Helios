package ratelimiter

import (
	"net/http"
	"testing"
	"time"
)

func TestTokenBucketRateLimiter(t *testing.T) {
	// Create a rate limiter with 5 tokens that refills every 100ms
	rl := NewTokenBucketRateLimiter(5, 100*time.Millisecond)
	clientIP := "192.168.1.100"

	// Should allow 5 requests initially
	for i := 0; i < 5; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if rl.Allow(clientIP) {
		t.Error("6th request should be denied")
	}

	// Wait for refill and try again
	time.Sleep(150 * time.Millisecond)
	if !rl.Allow(clientIP) {
		t.Error("Request should be allowed after refill")
	}
}

func TestTokenBucketRateLimiterDifferentClients(t *testing.T) {
	rl := NewTokenBucketRateLimiter(2, 100*time.Millisecond)

	client1 := "192.168.1.100"
	client2 := "192.168.1.101"

	// Both clients should be able to make 2 requests
	for i := 0; i < 2; i++ {
		if !rl.Allow(client1) {
			t.Errorf("Client1 request %d should be allowed", i+1)
		}
		if !rl.Allow(client2) {
			t.Errorf("Client2 request %d should be allowed", i+1)
		}
	}

	// 3rd request should be denied for both
	if rl.Allow(client1) {
		t.Error("Client1 3rd request should be denied")
	}
	if rl.Allow(client2) {
		t.Error("Client2 3rd request should be denied")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	rl := NewTokenBucketRateLimiter(1, 50*time.Millisecond)
	clientIP := "192.168.1.100"

	// Use up the initial token
	if !rl.Allow(clientIP) {
		t.Error("First request should be allowed")
	}

	// Should be denied immediately
	if rl.Allow(clientIP) {
		t.Error("Second request should be denied")
	}

	// Wait for refill
	time.Sleep(60 * time.Millisecond)

	// Should be allowed after refill
	if !rl.Allow(clientIP) {
		t.Error("Request should be allowed after refill")
	}
}

// TestGetClientIP tests IP extraction from various HTTP headers
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		xff            string // X-Forwarded-For header
		xri            string // X-Real-IP header
		remoteAddr     string
		expectedIP     string
		description    string
	}{
		{
			name:        "X-Forwarded-For with single IP",
			xff:         "203.0.113.195",
			remoteAddr:  "10.0.0.1:1234",
			expectedIP:  "203.0.113.195",
			description: "Should extract single IP from XFF",
		},
		{
			name:        "X-Forwarded-For with multiple IPs",
			xff:         "203.0.113.195, 70.41.3.18, 150.172.238.178",
			remoteAddr:  "10.0.0.1:1234",
			expectedIP:  "203.0.113.195",
			description: "Should extract FIRST IP from comma-separated XFF list (actual client)",
		},
		{
			name:        "X-Forwarded-For with spaces",
			xff:         "  203.0.113.195  ,  70.41.3.18  ",
			remoteAddr:  "10.0.0.1:1234",
			expectedIP:  "203.0.113.195",
			description: "Should trim whitespace from XFF",
		},
		{
			name:        "X-Real-IP when no XFF",
			xri:         "203.0.113.195",
			remoteAddr:  "10.0.0.1:1234",
			expectedIP:  "203.0.113.195",
			description: "Should use X-Real-IP when XFF is absent",
		},
		{
			name:        "X-Forwarded-For takes precedence over X-Real-IP",
			xff:         "203.0.113.195",
			xri:         "70.41.3.18",
			remoteAddr:  "10.0.0.1:1234",
			expectedIP:  "203.0.113.195",
			description: "XFF should be preferred over X-Real-IP",
		},
		{
			name:        "RemoteAddr fallback with port",
			remoteAddr:  "203.0.113.195:56789",
			expectedIP:  "203.0.113.195",
			description: "Should extract IP from RemoteAddr, stripping port",
		},
		{
			name:        "RemoteAddr fallback without port",
			remoteAddr:  "203.0.113.195",
			expectedIP:  "203.0.113.195",
			description: "Should handle RemoteAddr without port",
		},
		{
			name:        "IPv6 address",
			remoteAddr:  "[2001:db8::1]:8080",
			expectedIP:  "2001:db8::1",
			description: "Should handle IPv6 addresses with port",
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

			got := getClientIP(req)
			if got != tt.expectedIP {
				t.Errorf("%s: got %q, want %q", tt.description, got, tt.expectedIP)
			}
		})
	}
}

