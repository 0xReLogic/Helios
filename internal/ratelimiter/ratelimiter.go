package ratelimiter

import (
	"net/http"
	"sync"
	"time"

	"github.com/0xReLogic/Helios/internal/utils"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(clientIP string) bool
}

// TokenBucketRateLimiter implements a token bucket rate limiter with optimized concurrency
type TokenBucketRateLimiter struct {
	maxTokens   int           // Maximum number of tokens in the bucket
	refillRate  time.Duration // Rate at which tokens are refilled
	buckets     sync.Map      // Use sync.Map for better concurrent access
	cleanupTick time.Duration
}

// bucket represents a token bucket for a specific client with lock-free fast path
type bucket struct {
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex // Only lock when modifying tokens
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(maxTokens int, refillRate time.Duration) *TokenBucketRateLimiter {
	rl := &TokenBucketRateLimiter{
		maxTokens:   maxTokens,
		refillRate:  refillRate,
		buckets:     sync.Map{},       // sync.Map doesn't need initialization
		cleanupTick: time.Minute * 10, // Clean up old buckets every 10 minutes
	}

	// Start cleanup routine
	go rl.cleanupRoutine()

	return rl
}

// Allow checks if a request from the given client IP is allowed with optimized locking
func (rl *TokenBucketRateLimiter) Allow(clientIP string) bool {
	// Fast path: try to load existing bucket without locking
	value, exists := rl.buckets.Load(clientIP)
	var b *bucket

	if !exists {
		// Create new bucket
		b = &bucket{
			tokens:     rl.maxTokens,
			lastRefill: time.Now(),
		}
		// LoadOrStore ensures only one goroutine creates the bucket
		actual, loaded := rl.buckets.LoadOrStore(clientIP, b)
		if loaded {
			// Another goroutine created it, use that one
			b = actual.(*bucket)
		}
	} else {
		b = value.(*bucket)
	}

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
	now := time.Now()
	cutoff := now.Add(-time.Hour)

	// Use sync.Map's Range method for iteration
	rl.buckets.Range(func(key, value interface{}) bool {
		ip := key.(string)
		b := value.(*bucket)

		b.mutex.Lock()
		shouldDelete := b.lastRefill.Before(cutoff)
		b.mutex.Unlock()

		if shouldDelete {
			rl.buckets.Delete(ip)
		}
		return true // continue iteration
	})
}

// RateLimitMiddleware wraps an http.Handler with rate limiting
func RateLimitMiddleware(rateLimiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := utils.GetClientIP(r)

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
