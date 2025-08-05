package ratelimiter

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(clientIP string) bool
}

// TokenBucketRateLimiter implements a token bucket rate limiter
type TokenBucketRateLimiter struct {
	maxTokens   int           // Maximum number of tokens in the bucket
	refillRate  time.Duration // Rate at which tokens are refilled
	buckets     map[string]*bucket
	mutex       sync.RWMutex
	cleanupTick time.Duration
}

// bucket represents a token bucket for a specific client
type bucket struct {
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(maxTokens int, refillRate time.Duration) *TokenBucketRateLimiter {
	rl := &TokenBucketRateLimiter{
		maxTokens:   maxTokens,
		refillRate:  refillRate,
		buckets:     make(map[string]*bucket),
		cleanupTick: time.Minute * 10, // Clean up old buckets every 10 minutes
	}

	// Start cleanup routine
	go rl.cleanupRoutine()

	return rl
}

// Allow checks if a request from the given client IP is allowed
func (rl *TokenBucketRateLimiter) Allow(clientIP string) bool {
	rl.mutex.Lock()
	b, exists := rl.buckets[clientIP]
	if !exists {
		b = &bucket{
			tokens:     rl.maxTokens,
			lastRefill: time.Now(),
		}
		rl.buckets[clientIP] = b
	}
	rl.mutex.Unlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > rl.maxTokens {
			b.tokens = rl.maxTokens
		}
		b.lastRefill = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanupRoutine removes old buckets that haven't been used recently
func (rl *TokenBucketRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes buckets that haven't been used for more than 1 hour
func (rl *TokenBucketRateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Hour)

	for ip, b := range rl.buckets {
		b.mutex.Lock()
		if b.lastRefill.Before(cutoff) {
			delete(rl.buckets, ip)
		}
		b.mutex.Unlock()
	}
}

// RateLimitMiddleware wraps an http.Handler with rate limiting
func RateLimitMiddleware(rateLimiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := getClientIP(r)

			// Check rate limit
			if !rateLimiter.Allow(clientIP) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			return xff[:idx]
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
