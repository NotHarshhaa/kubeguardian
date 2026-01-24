package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	tokens   int
	capacity int
	refillRate int // tokens per second
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity, refillRate int) *RateLimiter {
	return &RateLimiter{
		tokens:    capacity,
		capacity:  capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.refillRate))
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.capacity {
			rl.tokens = rl.capacity
		}
		rl.lastRefill = now
	}
}

// ActionRateLimiter manages rate limiting for different actions
type ActionRateLimiter struct {
	mu          sync.RWMutex
	limiter     map[string]*RateLimiter
	defaultRate int
	defaultCap  int
}

// NewActionRateLimiter creates a new action rate limiter
func NewActionRateLimiter(defaultRate, defaultCap int) *ActionRateLimiter {
	return &ActionRateLimiter{
		limiter:     make(map[string]*RateLimiter),
		defaultRate: defaultRate,
		defaultCap:  defaultCap,
	}
}

// Allow checks if an action is allowed
func (arl *ActionRateLimiter) Allow(action string) bool {
	arl.mu.RLock()
	limiter, exists := arl.limiter[action]
	arl.mu.RUnlock()

	if !exists {
		arl.mu.Lock()
		// Double-check after acquiring write lock
		if limiter, exists = arl.limiter[action]; !exists {
			limiter = NewRateLimiter(arl.defaultCap, arl.defaultRate)
			arl.limiter[action] = limiter
		}
		arl.mu.Unlock()
	}

	return limiter.Allow()
}

// SetRate sets a custom rate for an action
func (arl *ActionRateLimiter) SetRate(action string, rate, capacity int) {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	
	arl.limiter[action] = NewRateLimiter(capacity, rate)
}

// GetStats returns current stats for an action
func (arl *ActionRateLimiter) GetStats(action string) (tokens, capacity int) {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	
	if limiter, exists := arl.limiter[action]; exists {
		limiter.mu.Lock()
		tokens = limiter.tokens
		capacity = limiter.capacity
		limiter.mu.Unlock()
		return
	}
	
	return arl.defaultCap, arl.defaultCap
}
