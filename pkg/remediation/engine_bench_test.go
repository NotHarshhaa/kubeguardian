package remediation

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func BenchmarkRemediationEngine(b *testing.B) {
	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	client := fake.NewSimpleClientset(pod)
	
	config := RemediationConfig{
		Enabled: true,
		DryRun:  true, // Use dry run for benchmarking
	}

	engine := NewEngine(client, config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := engine.ExecuteAction(context.Background(), "restart-pod", pod, "default")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCooldownCheck(b *testing.B) {
	client := fake.NewSimpleClientset()
	config := RemediationConfig{Enabled: true}
	engine := NewEngine(client, config)

	// Add some cooldown entries
	engine.recordCooldown("default:test-pod:restart-pod")
	engine.recordCooldown("default:test-deployment:rollback-deployment")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.isInCooldown("default:test-pod:restart-pod", 300)
	}
}

func BenchmarkCooldownCleanup(b *testing.B) {
	client := fake.NewSimpleClientset()
	config := RemediationConfig{Enabled: true}
	engine := NewEngine(client, config)

	// Add many cooldown entries
	for i := 0; i < 1000; i++ {
		engine.recordCooldown("default:resource:action")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CleanupCooldowns()
	}
}

func BenchmarkRateLimiting(b *testing.B) {
	client := fake.NewSimpleClientset()
	config := RemediationConfig{Enabled: true}
	engine := NewEngine(client, config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine.rateLimiter.Allow("test-action")
		}
	})
}

func BenchmarkCircuitBreaker(b *testing.B) {
	client := fake.NewSimpleClientset()
	config := RemediationConfig{Enabled: true}
	engine := NewEngine(client, config)

	cb := engine.circuitBreaker["pods"]

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(context.Background(), func() error {
				return nil
			})
		}
	})
}
