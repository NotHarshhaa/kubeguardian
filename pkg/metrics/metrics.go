package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	once sync.Once

	// Detection metrics
	issuesDetectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeguardian_issues_detected_total",
			Help: "Total number of issues detected by rule",
		},
		[]string{"rule", "severity", "namespace"},
	)

	detectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeguardian_detection_duration_seconds",
			Help:    "Time spent detecting issues",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"rule"},
	)

	// Remediation metrics
	remediationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeguardian_remediations_total",
			Help: "Total number of remediation actions executed",
		},
		[]string{"action", "result", "namespace"},
	)

	remediationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeguardian_remediation_duration_seconds",
			Help:    "Time spent executing remediation actions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"action"},
	)

	// Cooldown metrics
	cooldownActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeguardian_cooldown_active",
			Help: "Number of active cooldown entries",
		},
		[]string{"namespace"},
	)

	// API metrics
	apiCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeguardian_api_calls_total",
			Help: "Total number of Kubernetes API calls",
		},
		[]string{"method", "resource", "status"},
	)

	apiDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeguardian_api_duration_seconds",
			Help:    "Time spent on Kubernetes API calls",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "resource"},
	)

	// Notification metrics
	notificationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeguardian_notifications_total",
			Help: "Total number of notifications sent",
		},
		[]string{"type", "status"},
	)

	// System metrics
	lastDetectionTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubeguardian_last_detection_timestamp",
			Help: "Timestamp of the last detection cycle",
		},
	)

	uptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubeguardian_uptime_seconds",
			Help: "Uptime of KubeGuardian in seconds",
		},
	)
)

// Metrics holds all metrics
type Metrics struct {
	startTime time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	once.Do(func() {
		// Register metrics with the controller-runtime metrics registry
		metrics.Registry.MustRegister(
			issuesDetectedTotal,
			detectionDuration,
			remediationTotal,
			remediationDuration,
			cooldownActive,
			apiCallsTotal,
			apiDuration,
			notificationsTotal,
			lastDetectionTime,
			uptime,
		)
	})

	return &Metrics{
		startTime: time.Now(),
	}
}

// RecordIssueDetected records a detected issue
func (m *Metrics) RecordIssueDetected(rule, severity, namespace string) {
	issuesDetectedTotal.WithLabelValues(rule, severity, namespace).Inc()
}

// RecordDetectionDuration records detection duration
func (m *Metrics) RecordDetectionDuration(rule string, duration time.Duration) {
	detectionDuration.WithLabelValues(rule).Observe(duration.Seconds())
}

// RecordRemediation records a remediation action
func (m *Metrics) RecordRemediation(action, result, namespace string, duration time.Duration) {
	remediationTotal.WithLabelValues(action, result, namespace).Inc()
	remediationDuration.WithLabelValues(action).Observe(duration.Seconds())
}

// RecordCooldownActive records active cooldowns
func (m *Metrics) RecordCooldownActive(namespace string, count int) {
	cooldownActive.WithLabelValues(namespace).Set(float64(count))
}

// RecordAPICall records an API call
func (m *Metrics) RecordAPICall(method, resource, status string, duration time.Duration) {
	apiCallsTotal.WithLabelValues(method, resource, status).Inc()
	apiDuration.WithLabelValues(method, resource).Observe(duration.Seconds())
}

// RecordNotification records a notification
func (m *Metrics) RecordNotification(notificationType, status string) {
	notificationsTotal.WithLabelValues(notificationType, status).Inc()
}

// UpdateLastDetectionTime updates the last detection timestamp
func (m *Metrics) UpdateLastDetectionTime() {
	lastDetectionTime.SetToCurrentTime()
}

// UpdateUptime updates the uptime metric
func (m *Metrics) UpdateUptime() {
	uptime.Set(time.Since(m.startTime).Seconds())
}
