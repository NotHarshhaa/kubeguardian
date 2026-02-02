package health

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

func TestHealthCheckInitialization(t *testing.T) {
	hc := NewHealthCheck("v1.0.0", nil)
	
	if hc == nil {
		t.Fatal("NewHealthCheck returned nil")
	}
	
	if hc.version != "v1.0.0" {
		t.Errorf("version = %s, want v1.0.0", hc.version)
	}
	
	if hc.startTime.IsZero() {
		t.Error("start time not set")
	}
}

func TestHealthCheckRegistration(t *testing.T) {
	hc := NewHealthCheck("v1.0.0", nil)
	
	// Register a custom check
	check := &MockCheck{name: "test-check"}
	hc.RegisterCheck(check)
	
	// Verify registration
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if len(hc.checks) != 4 { // 3 built-in + 1 custom
		t.Errorf("expected 4 checks, got %d", len(hc.checks))
	}
	
	if _, exists := hc.checks["test-check"]; !exists {
		t.Error("custom check not registered")
	}
}

func TestHealthCheckExecution(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	// Add a mock check that succeeds
	successCheck := &MockCheck{
		name: "success",
		checkFunc: func(ctx context.Context) error {
			return nil
		},
	}
	hc.RegisterCheck(successCheck)
	
	// Add a mock check that fails
	failCheck := &MockCheck{
		name: "failure",
		checkFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc.RegisterCheck(failCheck)
	
	// Run health check
	ctx := context.Background()
	hc.RunChecks(ctx)
	
	// Check results
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if len(hc.results) != 5 { // 3 built-in + 2 custom
		t.Errorf("expected 5 results, got %d", len(hc.results))
	}
	
	// Verify success check result
	if result, exists := hc.results["success"]; exists {
		if result.Status != StatusHealthy {
			t.Errorf("success check status = %s, want %s", result.Status, StatusHealthy)
		}
	} else {
		t.Error("success check result not found")
	}
	
	// Verify failure check result
	if result, exists := hc.results["failure"]; exists {
		if result.Status != StatusUnhealthy {
			t.Errorf("failure check status = %s, want %s", result.Status, StatusUnhealthy)
		}
	} else {
		t.Error("failure check result not found")
	}
}

func TestHealthCheckOverall(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	// All checks should pass initially
	ctx := context.Background()
	hc.RunChecks(ctx)
	
	health := hc.GetHealth()
	if health.Status != StatusHealthy {
		t.Errorf("overall health status = %s, want %s", health.Status, StatusHealthy)
	}
	
	// Add a failing check
	failCheck := &MockCheck{
		name: "failure",
		checkFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc.RegisterCheck(failCheck)
	
	hc.RunChecks(ctx)
	health = hc.GetHealth()
	if health.Status != StatusUnhealthy {
		t.Errorf("overall health status with failure = %s, want %s", health.Status, StatusUnhealthy)
	}
}

func TestHealthCheckTimeout(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	// Add a slow check
	slowCheck := &MockCheck{
		name: "slow",
		checkFunc: func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return nil
		},
	}
	hc.RegisterCheck(slowCheck)
	
	// Run with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	hc.RunChecks(ctx)
	
	// Check should be marked as unhealthy due to timeout
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if result, exists := hc.results["slow"]; exists {
		if result.Status != StatusUnhealthy {
			t.Errorf("slow check status = %s, want %s", result.Status, StatusUnhealthy)
		}
	}
}

func TestHealthCheckJSON(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	// Add a test check
	testCheck := &MockCheck{
		name: "test",
		checkFunc: func(ctx context.Context) error {
			return nil
		},
	}
	hc.RegisterCheck(testCheck)
	
	// Run checks
	hc.RunChecks(context.Background())
	
	// Get JSON response
	health := hc.GetHealth()
	
	if health.Status == "" {
		t.Error("health response is empty")
	}
	
	// Check if version is included
	if health.Version != "v1.0.0" {
		t.Error("version not included in health response")
	}
}

func TestLivenessHandler(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	// Liveness should always return OK
	handler := hc.LivenessHandler()
	if handler == nil {
		t.Fatal("LivenessHandler returned nil")
	}
	
	// Test would require HTTP testing framework
	// This is a basic existence test
}

func TestReadinessHandler(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	handler := hc.ReadinessHandler()
	if handler == nil {
		t.Fatal("ReadinessHandler returned nil")
	}
}

func TestHTTPHandler(t *testing.T) {
	// Create a mock client for testing
	client := fake.NewSimpleClientset()
	hc := NewHealthCheck("v1.0.0", client)
	
	handler := hc.HTTPHandler()
	if handler == nil {
		t.Fatal("HTTPHandler returned nil")
	}
}

// MockCheck implements the Checker interface for testing
type MockCheck struct {
	name      string
	checkFunc func(ctx context.Context) error
}

func (m *MockCheck) Name() string {
	return m.name
}

func (m *MockCheck) Check(ctx context.Context) error {
	if m.checkFunc != nil {
		return m.checkFunc(ctx)
	}
	return nil
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   (len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		   (len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
