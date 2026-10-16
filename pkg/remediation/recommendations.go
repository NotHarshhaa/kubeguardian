package remediation

import (
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ResourceRecommendation represents a VPA-like recommendation for a container
type ResourceRecommendation struct {
	ContainerName string            `json:"containerName"`
	Namespace     string            `json:"namespace"`
	PodName       string            `json:"podName"`
	CPURequest    string            `json:"cpuRequest"`
	CPULimit      string            `json:"cpuLimit"`
	MemoryRequest string            `json:"memoryRequest"`
	MemoryLimit   string            `json:"memoryLimit"`
	TargetCPUUtilization   float64           `json:"targetCPUUtilization"`
	TargetMemoryUtilization float64          `json:"targetMemoryUtilization"`
	Confidence    float64           `json:"confidence"` // 0-1
	LastUpdated   time.Time         `json:"lastUpdated"`
	Reason        string            `json:"reason"`
}

// ResourceRecommender generates resource recommendations based on historical usage
type ResourceRecommender struct {
	mu        sync.RWMutex
	recommendations map[string]*ResourceRecommendation // key: namespace/pod/container
	history     map[string][]MetricSample          // key: namespace/pod/container
}

// MetricSample represents a single metric sample
type MetricSample struct {
	Timestamp time.Time `json:"timestamp"`
	CPUUsage  float64   `json:"cpuUsage"`     // cores
	MemoryUsage float64  `json:"memoryUsage"`  // bytes
}

// NewResourceRecommender creates a new resource recommender
func NewResourceRecommender() *ResourceRecommender {
	return &ResourceRecommender{
		recommendations: make(map[string]*ResourceRecommendation),
		history:         make(map[string][]MetricSample),
	}
}

// AddSample adds a metric sample for a container
func (r *ResourceRecommender) AddSample(namespace, pod, container string, cpuUsage, memoryUsage float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
	sample := MetricSample{
		Timestamp:   time.Now(),
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
	}

	r.history[key] = append(r.history[key], sample)

	// Keep only last 1000 samples
	if len(r.history[key]) > 1000 {
		r.history[key] = r.history[key][len(r.history[key])-1000:]
	}
}

// GenerateRecommendation generates a resource recommendation for a container
func (r *ResourceRecommender) GenerateRecommendation(namespace, pod, container string, currentRequest, currentLimit corev1.ResourceList) *ResourceRecommendation {
	r.mu.RLock()
	samples, exists := r.history[fmt.Sprintf("%s/%s/%s", namespace, pod, container)]
	r.mu.RUnlock()

	if !exists || len(samples) == 0 {
		return nil
	}

	// Calculate p95 of CPU and memory usage
	cpuP95 := percentile(samples, func(s MetricSample) float64 { return s.CPUUsage }, 95)
	memP95 := percentile(samples, func(s MetricSample) float64 { return s.MemoryUsage }, 95)

	// Calculate recommendations with headroom
	headroomCPU := 1.25  // 25% headroom above p95
	headroomMem := 1.20  // 20% headroom above p95

	recCPU := cpuP95 * headroomCPU
	recMem := memP95 * headroomMem

	// Cap at current limit if set
	currentCPULimit := resourceQuantityToFloat(currentLimit, corev1.ResourceCPU)
	currentMemLimit := resourceQuantityToFloat(currentLimit, corev1.ResourceMemory)

	if currentCPULimit > 0 && recCPU > currentCPULimit {
		recCPU = currentCPULimit
	}
	if currentMemLimit > 0 && recMem > currentMemLimit {
		recMem = currentMemLimit
	}

	// Confidence based on sample count
	confidence := float64(len(samples)) / 100.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	reason := fmt.Sprintf("Based on %d samples over %s, p95 CPU=%.3f cores, p95 memory=%.0f MiB",
		len(samples), formatDuration(time.Since(samples[0].Timestamp)), cpuP95, memP95/1024/1024)

	return &ResourceRecommendation{
		ContainerName:         container,
		Namespace:             namespace,
		PodName:               pod,
		CPURequest:            formatCPU(recCPU),
		CPULimit:              formatCPU(recCPU * 1.5), // 1.5x burst
		MemoryRequest:         formatMemory(recMem),
		MemoryLimit:           formatMemory(recMem * 1.5),
		TargetCPUUtilization:  cpuP95 / recCPU * 100,
		TargetMemoryUtilization: memP95 / recMem * 100,
		Confidence:            confidence,
		LastUpdated:           time.Now(),
		Reason:                reason,
	}
}

// GetRecommendation returns a stored recommendation
func (r *ResourceRecommender) GetRecommendation(namespace, pod, container string) (*ResourceRecommendation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
	rec, exists := r.recommendations[key]
	return rec, exists
}

// StoreRecommendation stores a recommendation
func (r *ResourceRecommender) StoreRecommendation(rec *ResourceRecommendation) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", rec.Namespace, rec.PodName, rec.ContainerName)
	r.recommendations[key] = rec
}

// GetAllRecommendations returns all stored recommendations
func (r *ResourceRecommender) GetAllRecommendations() []*ResourceRecommendation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ResourceRecommendation, 0, len(r.recommendations))
	for _, rec := range r.recommendations {
		result = append(result, rec)
	}
	return result
}

// percentile calculates the p-th percentile of values extracted from samples
func percentile[T any](samples []T, extract func(T) float64, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	values := make([]float64, len(samples))
	for i, s := range samples {
		values[i] = extract(s)
	}

	// Sort
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	idx := int(float64(len(values)-1) * (p / 100.0))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}

	return values[idx]
}

// maxFloat returns the maximum value extracted from samples
func maxFloat[T any](samples []T, extract func(T) float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	max := extract(samples[0])
	for _, s := range samples[1:] {
		if v := extract(s); v > max {
			max = v
		}
	}
	return max
}

// resourceQuantityToFloat converts a resource quantity to float64
func resourceQuantityToFloat(res corev1.ResourceList, name corev1.ResourceName) float64 {
	q, ok := res[name]
	if !ok {
		return 0
	}
	// Convert to float64 using the quantity's value
	return float64(q.MilliValue()) / 1000.0
}

// formatCPU formats a CPU value in cores as millicores string
func formatCPU(cores float64) string {
	millicores := int64(cores * 1000)
	if millicores%1000 == 0 {
		return fmt.Sprintf("%d", millicores/1000)
	}
	return fmt.Sprintf("%dm", millicores)
}

// formatMemory formats a memory value in bytes as MiB string
func formatMemory(bytes float64) string {
	mib := int64(bytes / 1024 / 1024)
	return fmt.Sprintf("%dMi", mib)
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
