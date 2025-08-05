package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxRequests      uint32        // Max requests allowed in half-open state
	interval         time.Duration // Time window for failure counting
	timeout          time.Duration // Time to wait before moving from open to half-open
	failureThreshold uint32        // Number of failures to open the circuit
	successThreshold uint32        // Number of successes to close the circuit in half-open state
	onStateChange    func(name string, from State, to State)

	mutex           sync.Mutex
	state           State
	failureCount    uint32
	successCount    uint32
	requestCount    uint32
	lastFailureTime time.Time
	lastSuccessTime time.Time
	nextAttempt     time.Time
}

var (
	// ErrCircuitBreakerOpen is returned when the circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when too many requests are made in half-open state
	ErrTooManyRequests = errors.New("too many requests")
)

// Settings holds the configuration for a circuit breaker
type Settings struct {
	Name             string
	MaxRequests      uint32
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold uint32
	SuccessThreshold uint32
	OnStateChange    func(name string, from State, to State)
}

// NewCircuitBreaker creates a new circuit breaker with the given settings
func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             settings.Name,
		maxRequests:      settings.MaxRequests,
		interval:         settings.Interval,
		timeout:          settings.Timeout,
		failureThreshold: settings.FailureThreshold,
		successThreshold: settings.SuccessThreshold,
		onStateChange:    settings.OnStateChange,
		state:            StateClosed,
	}

	// Set default values if not provided
	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval == 0 {
		cb.interval = time.Minute
	}
	if cb.timeout == 0 {
		cb.timeout = time.Minute
	}
	if cb.failureThreshold == 0 {
		cb.failureThreshold = 5
	}
	if cb.successThreshold == 0 {
		cb.successThreshold = 1
	}

	return cb
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	err := cb.beforeRequest()
	if err != nil {
		return err
	}

	// Increment request count for half-open state
	cb.mutex.Lock()
	if cb.state == StateHalfOpen {
		cb.requestCount++
	}
	cb.mutex.Unlock()

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(false)
			panic(r)
		}
	}()

	err = fn()
	cb.afterRequest(err == nil)
	return err
}

// Call is an alias for Execute for backward compatibility
func (cb *CircuitBreaker) Call(fn func() error) error {
	return cb.Execute(fn)
}

// beforeRequest checks if the request can proceed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Reset counts if interval has passed
		if cb.lastFailureTime.Add(cb.interval).Before(now) {
			cb.failureCount = 0
		}
		return nil

	case StateOpen:
		// Check if timeout has passed to move to half-open
		if cb.nextAttempt.Before(now) {
			cb.setState(StateHalfOpen)
			cb.requestCount = 0
			cb.successCount = 0
			return nil
		}
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		// Check if we've reached max requests in half-open state
		if cb.requestCount >= cb.maxRequests {
			return ErrTooManyRequests
		}
		// Don't increment here, we'll increment in Execute
		return nil

	default:
		return ErrCircuitBreakerOpen
	}
}

// afterRequest updates the circuit breaker state after a request
func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	if success {
		cb.lastSuccessTime = now
		switch cb.state {
		case StateClosed:
			// Stay closed
		case StateHalfOpen:
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				cb.setState(StateClosed)
				cb.failureCount = 0
			}
		}
	} else {
		cb.lastFailureTime = now
		cb.failureCount++

		switch cb.state {
		case StateClosed:
			if cb.failureCount >= cb.failureThreshold {
				cb.setState(StateOpen)
				cb.nextAttempt = now.Add(cb.timeout)
			}
		case StateHalfOpen:
			cb.setState(StateOpen)
			cb.nextAttempt = now.Add(cb.timeout)
		}
	}
}

// setState changes the circuit breaker state and calls the callback
func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Counts returns the current counts for the circuit breaker
func (cb *CircuitBreaker) Counts() (failureCount, successCount, requestCount uint32) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.failureCount, cb.successCount, cb.requestCount
}
