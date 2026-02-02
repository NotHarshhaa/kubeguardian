package ratelimit

import (
	"testing"
	"time"
)

func TestActionRateLimiter(t *testing.T) {
	// Create a rate limiter: 2 requests per second, bucket capacity 5
	rl := NewActionRateLimiter(2, 5)

	if rl == nil {
		t.Fatal("NewActionRateLimiter returned nil")
	}

	// Test initial capacity - should allow 5 requests immediately
	for i := 0; i < 5; i++ {
		if !rl.Allow("test-action") {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Next request should be denied (bucket empty)
	if rl.Allow("test-action") {
		t.Error("request should be denied when bucket is empty")
	}

	// Wait for token refill (1 token per 0.5 seconds for 2 req/sec)
	time.Sleep(600 * time.Millisecond)

	// Should allow one request now
	if !rl.Allow("test-action") {
		t.Error("request should be allowed after token refill")
	}

	// But not two requests
	if rl.Allow("test-action") {
		t.Error("second request should be denied")
	}
}

func TestMultipleActions(t *testing.T) {
	rl := NewActionRateLimiter(1, 3) // 1 req/sec, capacity 3

	// Different actions should have separate buckets
	for i := 0; i < 3; i++ {
		if !rl.Allow("action1") {
			t.Errorf("action1 request %d should be allowed", i)
		}
		if !rl.Allow("action2") {
			t.Errorf("action2 request %d should be allowed", i)
		}
	}

	// Both should be denied now
	if rl.Allow("action1") {
		t.Error("action1 should be denied")
	}
	if rl.Allow("action2") {
		t.Error("action2 should be denied")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewActionRateLimiter(10, 10) // 10 req/sec, capacity 10

	// Empty the bucket
	for i := 0; i < 10; i++ {
		rl.Allow("test")
	}

	// Should be empty
	if rl.Allow("test") {
		t.Error("should be denied when empty")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond) // Should refill ~1.5 tokens

	// Should allow at least one request
	if !rl.Allow("test") {
		t.Error("should allow request after refill")
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	rl := NewActionRateLimiter(100, 100) // High rate for concurrency test

	// Test concurrent access
	done := make(chan bool, 50)
	allowed := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func() {
			result := rl.Allow("concurrent-test")
			allowed <- result
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Count allowed requests
	allowedCount := 0
	for i := 0; i < 50; i++ {
		if <-allowed {
			allowedCount++
		}
	}

	// Should allow exactly bucket capacity (100) requests, but we only made 50
	if allowedCount != 50 {
		t.Errorf("expected 50 allowed requests, got %d", allowedCount)
	}
}

func TestRateLimiterZeroRate(t *testing.T) {
	rl := NewActionRateLimiter(0, 10) // Zero rate

	// Should allow initial capacity
	for i := 0; i < 10; i++ {
		if !rl.Allow("test") {
			t.Errorf("initial request %d should be allowed", i)
		}
	}

	// Should never allow more (no refill)
	time.Sleep(100 * time.Millisecond)
	if rl.Allow("test") {
		t.Error("should never allow requests with zero rate")
	}
}

func TestRateLimiterZeroCapacity(t *testing.T) {
	rl := NewActionRateLimiter(10, 0) // Zero capacity

	// Should never allow any requests
	if rl.Allow("test") {
		t.Error("should never allow requests with zero capacity")
	}

	time.Sleep(100 * time.Millisecond)
	if rl.Allow("test") {
		t.Error("should never allow requests even after time with zero capacity")
	}
}

func TestRateLimiterMetrics(t *testing.T) {
	rl := NewActionRateLimiter(5, 5)

	// Make some requests
	for i := 0; i < 8; i++ {
		rl.Allow("test")
	}

	// Test metrics (implementation specific)
	// This is a placeholder for metrics testing
	_, capacity := rl.GetStats("test")
	if capacity == 0 {
		t.Error("stats should not be zero")
	}
}
