package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 5,
		Interval:    100 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
	})
	
	// Test initial state
	if cb.State() != StateClosed {
		t.Errorf("initial state = %v, want %v", cb.State(), StateClosed)
	}
	
	// Test successful call
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("successful call returned error: %v", err)
	}
	
	if cb.State() != StateClosed {
		t.Errorf("state after success = %v, want %v", cb.State(), StateClosed)
	}
}

func TestCircuitBreakerTripping(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 1,
		Interval:    50 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
	
	// Fail consecutive calls to trigger circuit breaker
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
		
		if err == nil {
			t.Errorf("call %d should have failed", i)
		}
	}
	
	// Circuit should be open now
	if cb.State() != StateOpen {
		t.Errorf("state after failures = %v, want %v", cb.State(), StateOpen)
	}
	
	// Calls should fail immediately when circuit is open
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	
	if err == nil {
		t.Error("call should fail immediately when circuit is open")
	}
	
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 1,
		Interval:    50 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})
	
	// Trip the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
	}
	
	// Wait for timeout to enter half-open state
	time.Sleep(60 * time.Millisecond)
	
	// Should be in half-open state now
	if cb.State() != StateHalfOpen {
		t.Errorf("state after timeout = %v, want %v", cb.State(), StateHalfOpen)
	}
	
	// A successful call should close the circuit
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("successful call in half-open failed: %v", err)
	}
	
	if cb.State() != StateClosed {
		t.Errorf("state after successful half-open call = %v, want %v", cb.State(), StateClosed)
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 10,
		Interval:    100 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
	})
	
	// Test concurrent calls
	done := make(chan bool, 10)
	errorChan := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := cb.Execute(context.Background(), func() error {
				time.Sleep(time.Millisecond)
				return nil
			})
			errorChan <- err
			done <- true
		}(i)
	}
	
	// Wait for all calls to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Check that no unexpected errors occurred
	for i := 0; i < 10; i++ {
		err := <-errorChan
		if err != nil && !errors.Is(err, ErrCircuitBreakerOpen) {
			t.Errorf("unexpected error in concurrent call: %v", err)
		}
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 5,
		Interval:    100 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
	})
	
	// Make some calls to generate metrics
	for i := 0; i < 10; i++ {
		cb.Execute(context.Background(), func() error {
			if i%3 == 0 {
				return errors.New("test error")
			}
			return nil
		})
	}
	
	// Test that metrics are accessible (implementation specific)
	// This is a placeholder for metrics testing
	counts := cb.Counts()
	if counts.Requests == 0 {
		t.Error("metrics should show some requests")
	}
}
