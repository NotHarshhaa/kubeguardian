package metrics

import (
	"testing"
	"time"
)

func BenchmarkRecordIssueDetected(b *testing.B) {
	m := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordIssueDetected("test-rule", "medium", "test-namespace")
	}
}

func BenchmarkRecordRemediation(b *testing.B) {
	m := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordRemediation("restart-pod", "success", "test-namespace", time.Millisecond)
	}
}

func BenchmarkRecordAPICall(b *testing.B) {
	m := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordAPICall("GET", "pods", "success", 100*time.Millisecond)
	}
}

func BenchmarkConcurrentMetrics(b *testing.B) {
	m := NewMetrics()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.RecordIssueDetected("test-rule", "medium", "test-namespace")
			m.RecordRemediation("test-action", "success", "test-namespace", time.Millisecond)
			m.RecordAPICall("GET", "test", "success", time.Millisecond)
		}
	})
}
