package ratelimiter

import (
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
