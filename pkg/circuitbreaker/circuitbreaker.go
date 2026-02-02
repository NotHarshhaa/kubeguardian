package circuitbreaker

import (
	"context"
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

// Circuit breaker errors
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests")
)

// Config holds circuit breaker configuration
type Config struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(err error) bool
	Fallback      func(ctx context.Context, name string, err error) error
}

// Counts holds circuit breaker counts
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)
	isSuccessful  func(err error) bool
	fallback      func(ctx context.Context, name string, err error) error

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, cfg Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   cfg.MaxRequests,
		interval:      cfg.Interval,
		timeout:       cfg.Timeout,
		readyToTrip:   cfg.ReadyToTrip,
		onStateChange: cfg.OnStateChange,
		isSuccessful:  cfg.IsSuccessful,
		fallback:      cfg.Fallback,
	}

	if cfg.MaxRequests == 0 {
		cb.maxRequests = 1
	}

	if cfg.Interval <= 0 {
		cb.interval = 60 * time.Second
	}

	if cfg.Timeout <= 0 {
		cb.timeout = 60 * time.Second
	}

	if cfg.ReadyToTrip == nil {
		cb.readyToTrip = DefaultReadyToTrip
	}

	if cfg.IsSuccessful == nil {
		cb.isSuccessful = DefaultIsSuccessful
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// DefaultReadyToTrip is the default ready-to-trip function
func DefaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

// DefaultIsSuccessful is the default is-successful function
func DefaultIsSuccessful(err error) bool {
	return err == nil
}

// Execute executes the given operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		if cb.fallback != nil {
			return cb.fallback(ctx, cb.name, err)
		}
		return err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = operation()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return err
}

// beforeRequest is called before a request
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrCircuitBreakerOpen
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.Requests++
	return generation, nil
}

// afterRequest is called after a request
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
		cb.setState(StateClosed, now)
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

// currentState returns the current state and generation
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState sets the circuit breaker state
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration creates a new generation
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns the current circuit breaker counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}
