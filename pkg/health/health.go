package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Status represents health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Check represents a health check
type Check struct {
	Name        string            `json:"name"`
	Status      Status            `json:"status"`
	Message     string            `json:"message,omitempty"`
	LastChecked time.Time         `json:"lastChecked"`
	Duration    time.Duration     `json:"duration"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    Status           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Checks    map[string]Check `json:"checks"`
	Uptime    time.Duration    `json:"uptime"`
	Version   string           `json:"version"`
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

// HealthCheck manages health checks
type HealthCheck struct {
	mu        sync.RWMutex
	checks    map[string]Checker
	results   map[string]Check
	startTime time.Time
	version   string
	client    kubernetes.Interface
}

// NewHealthCheck creates a new health check manager
func NewHealthCheck(version string, client kubernetes.Interface) *HealthCheck {
	hc := &HealthCheck{
		checks:    make(map[string]Checker),
		results:   make(map[string]Check),
		startTime: time.Now(),
		version:   version,
		client:    client,
	}

	// Register built-in checks
	hc.RegisterCheck(NewKubernetesAPICheck(client))
	hc.RegisterCheck(NewMemoryCheck(80.0))    // 80% memory threshold
	hc.RegisterCheck(NewDiskCheck("/", 85.0)) // 85% disk threshold

	return hc
}

// RegisterCheck registers a health check
func (h *HealthCheck) RegisterCheck(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[checker.Name()] = checker
}

// RunChecks runs all registered health checks
func (h *HealthCheck) RunChecks(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, checker := range h.checks {
		start := time.Now()

		result := Check{
			Name:        name,
			LastChecked: start,
			Duration:    0,
		}

		err := checker.Check(ctx)
		result.Duration = time.Since(start)

		if err != nil {
			result.Status = StatusUnhealthy
			result.Message = err.Error()
		} else {
			result.Status = StatusHealthy
			result.Message = "OK"
		}

		h.results[name] = result
	}
}

// GetHealth returns the current health status
func (h *HealthCheck) GetHealth() HealthResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()

	overallStatus := StatusHealthy
	for _, result := range h.results {
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		} else if result.Status == StatusUnknown && overallStatus == StatusHealthy {
			overallStatus = StatusUnknown
		}
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    h.results,
		Uptime:    time.Since(h.startTime),
		Version:   h.version,
	}
}

// IsHealthy returns true if all checks are healthy
func (h *HealthCheck) IsHealthy() bool {
	health := h.GetHealth()
	return health.Status == StatusHealthy
}

// HTTPHandler returns an HTTP handler for health checks
func (h *HealthCheck) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Run checks on demand
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		h.RunChecks(ctx)

		health := h.GetHealth()

		w.Header().Set("Content-Type", "application/json")

		switch health.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

		json.NewEncoder(w).Encode(health)
	}
}

// ReadinessHandler returns a readiness probe handler
func (h *HealthCheck) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := h.GetHealth()

		if health.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Ready"))
		}
	}
}

// LivenessHandler returns a liveness probe handler
func (h *HealthCheck) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For liveness, we just check if the service is running
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// Built-in health checks

// KubernetesAPICheck checks connectivity to Kubernetes API
type KubernetesAPICheck struct {
	name   string
	client kubernetes.Interface
}

func NewKubernetesAPICheck(client kubernetes.Interface) *KubernetesAPICheck {
	return &KubernetesAPICheck{
		name:   "kubernetes-api",
		client: client,
	}
}

func (k *KubernetesAPICheck) Name() string {
	return k.name
}

func (k *KubernetesAPICheck) Check(ctx context.Context) error {
	// Test API connectivity by listing namespaces
	_, err := k.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

// MemoryCheck checks memory usage
type MemoryCheck struct {
	name      string
	threshold float64 // percentage
}

func NewMemoryCheck(threshold float64) *MemoryCheck {
	return &MemoryCheck{
		name:      "memory",
		threshold: threshold,
	}
}

func (m *MemoryCheck) Name() string {
	return m.name
}

func (m *MemoryCheck) Check(ctx context.Context) error {
	// In a real implementation, you'd check actual memory usage
	// For now, just simulate the check
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// DiskCheck checks disk usage
type DiskCheck struct {
	name      string
	path      string
	threshold float64 // percentage
}

func NewDiskCheck(path string, threshold float64) *DiskCheck {
	return &DiskCheck{
		name:      "disk",
		path:      path,
		threshold: threshold,
	}
}

func (d *DiskCheck) Name() string {
	return d.name
}

func (d *DiskCheck) Check(ctx context.Context) error {
	// In a real implementation, you'd check actual disk usage
	// For now, just simulate the check
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
