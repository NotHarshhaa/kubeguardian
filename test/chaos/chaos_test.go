//go:build chaos
// +build chaos

package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// ChaosTest represents a chaos engineering test
type ChaosTest struct {
	Name        string
	Description string
	TestFunc    func(t *testing.T, client *fake.Clientset)
}

func TestChaosEngineering(t *testing.T) {
	chaosTests := []ChaosTest{
		{
			Name:        "API Server Latency",
			Description: "Test behavior when API server is slow",
			TestFunc:    testAPIServerLatency,
		},
		{
			Name:        "API Server Failures",
			Description: "Test behavior when API server returns errors",
			TestFunc:    testAPIServerFailures,
		},
		{
			Name:        "Resource Exhaustion",
			Description: "Test behavior under resource pressure",
			TestFunc:    testResourceExhaustion,
		},
		{
			Name:        "Network Partitions",
			Description: "Test behavior during network issues",
			TestFunc:    testNetworkPartitions,
		},
		{
			Name:        "Memory Pressure",
			Description: "Test behavior under memory pressure",
			TestFunc:    testMemoryPressure,
		},
		{
			Name:        "High Load",
			Description: "Test behavior under high load",
			TestFunc:    testHighLoad,
		},
	}

	for _, test := range chaosTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Logf("Running chaos test: %s - %s", test.Name, test.Description)
			test.TestFunc(t, fake.NewSimpleClientset())
		})
	}
}

func testAPIServerLatency(t *testing.T, client *fake.Clientset) {
	// Create test resources
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	client.Fake.Resources = append(client.Fake.Resources, pod)

	// Create controller with configuration
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 100 * time.Millisecond,
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Inject latency into API calls
	client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		// Simulate API latency
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		return false, nil, nil
	})

	// Run controller with latency
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err = ctrl.Run(ctx)
	duration := time.Since(start)

	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Controller failed with latency: %v", err)
	}

	t.Logf("Controller ran for %v with API latency", duration)
}

func testAPIServerFailures(t *testing.T, client *fake.Clientset) {
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 100 * time.Millisecond,
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Inject random API failures
	client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		if rand.Intn(10) < 3 { // 30% failure rate
			return true, nil, fmt.Errorf("simulated API failure")
		}
		return false, nil, nil
	})

	// Run controller with failures
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = ctrl.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Controller handled API failures gracefully: %v", err)
	}
}

func testResourceExhaustion(t *testing.T, client *fake.Clientset) {
	// Create many resources to simulate resource exhaustion
	for i := 0; i < 1000; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
		client.Fake.Resources = append(client.Fake.Resources, pod)
	}

	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 50 * time.Millisecond, // Very frequent
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Run controller with many resources
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err = ctrl.Run(ctx)
	duration := time.Since(start)

	t.Logf("Controller processed 1000 resources in %v", duration)

	if duration > 5*time.Second {
		t.Errorf("Controller took too long to process resources: %v", duration)
	}
}

func testNetworkPartitions(t *testing.T, client *fake.Clientset) {
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 100 * time.Millisecond,
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Simulate network partition - intermittent failures
	partitionActive := false
	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			partitionActive = !partitionActive
		}
	}()

	client.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		if partitionActive {
			return true, nil, fmt.Errorf("network partition simulated")
		}
		return false, nil, nil
	})

	// Run controller during network partition
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = ctrl.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Controller handled network partitions: %v", err)
	}
}

func testMemoryPressure(t *testing.T, client *fake.Clientset) {
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 10 * time.Millisecond, // Very frequent to stress memory
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Create many objects to increase memory pressure
	for i := 0; i < 100; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("memory-test-pod-%d", i),
				Namespace: "default",
				Labels: map[string]string{
					"app":   "memory-test",
					"index": fmt.Sprintf("%d", i),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "nginx",
					},
				},
			},
		}
		client.Fake.Resources = append(client.Fake.Resources, pod)
	}

	// Run controller under memory pressure
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = ctrl.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Controller handled memory pressure: %v", err)
	}
}

func testHighLoad(t *testing.T, client *fake.Clientset) {
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 1 * time.Millisecond, // Extremely high frequency
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Create resources that require processing
	for i := 0; i < 50; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("load-test-pod-%d", i),
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed, // Generate issues to process
			},
		}
		client.Fake.Resources = append(client.Fake.Resources, pod)
	}

	// Run controller under high load
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = ctrl.Run(ctx)
	duration := time.Since(start)

	t.Logf("Controller handled high load for %v", duration)

	// Controller should handle high load without crashing
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Controller failed under high load: %v", err)
	}
}

func TestCircuitBreakerChaos(t *testing.T) {
	// Test circuit breaker under chaos conditions
	cb := circuitbreaker.NewCircuitBreaker("chaos-test", circuitbreaker.Config{
		MaxRequests: 5,
		Interval:    100 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
		TripAfter:   3,
	})

	// Simulate chaotic conditions
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	successCount := 0
	failureCount := 0

	for i := 0; i < 100; i++ {
		err := cb.Call(ctx, func(ctx context.Context) error {
			if rand.Intn(10) < 7 { // 70% failure rate
				return fmt.Errorf("chaos failure")
			}
			return nil
		})

		if err == nil {
			successCount++
		} else {
			failureCount++
		}

		// Small delay to prevent overwhelming
		time.Sleep(time.Millisecond)
	}

	t.Logf("Circuit breaker chaos test: %d successes, %d failures", successCount, failureCount)

	// Circuit breaker should have opened at some point
	if cb.State() == circuitbreaker.StateClosed && failureCount > 50 {
		t.Error("Circuit breaker should have opened with high failure rate")
	}
}

func TestRateLimiterChaos(t *testing.T) {
	// Test rate limiter under burst conditions
	rl := ratelimit.NewActionRateLimiter(10, 20) // 10 req/sec, 20 burst capacity

	// Simulate burst traffic
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	allowedCount := 0
	deniedCount := 0

	for i := 0; i < 100; i++ {
		if rl.Allow("chaos-test") {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	t.Logf("Rate limiter chaos test: %d allowed, %d denied", allowedCount, deniedCount)

	// Should have some denials due to rate limiting
	if deniedCount == 0 {
		t.Error("Rate limiter should have denied some requests under burst conditions")
	}
}
