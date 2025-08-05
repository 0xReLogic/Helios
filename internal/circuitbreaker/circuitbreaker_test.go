package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	// Circuit should start closed
	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", cb.State())
	}

	// Successful requests should keep it closed
	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error {
			return nil // Success
		})
		if err != nil {
			t.Errorf("Successful request %d failed: %v", i+1, err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED after successful requests, got %s", cb.State())
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	// Generate failures to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return errors.New("simulated failure")
		})
		if err == nil {
			t.Errorf("Request %d should have failed", i+1)
		}
	}

	// Circuit should now be open
	if cb.State() != StateOpen {
		t.Errorf("Expected state OPEN after failures, got %s", cb.State())
	}

	// Requests should be rejected immediately
	err := cb.Execute(func() error {
		return nil
	})
	if err != ErrCircuitBreakerOpen {
		t.Errorf("Expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name:             "test",
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      1,
	})

	// Generate failures to open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should move to half-open
	err := cb.Execute(func() error {
		return nil // Success
	})
	if err != nil {
		t.Errorf("First request after timeout should succeed, got %v", err)
	}

	// Should be closed again after successful request
	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED after successful half-open request, got %s", cb.State())
	}
}

func TestCircuitBreakerMaxRequests(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 3, // Increased so circuit stays half-open longer
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
	})

	// Open the circuit
	cb.Execute(func() error {
		return errors.New("failure")
	})

	// Wait for timeout to move to half-open
	time.Sleep(60 * time.Millisecond)

	// Should allow max_requests in half-open state
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Request %d in half-open should be allowed, got %v", i+1, err)
		}
	}

	// Additional request should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	if err != ErrTooManyRequests {
		t.Errorf("Expected ErrTooManyRequests, got %v", err)
	}
}
