package controller

import (
	"context"
	"testing"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// MockKubernetesClient extends fake client with additional mock capabilities
type MockKubernetesClient struct {
	*fake.Clientset
}

func NewMockKubernetesClient(objects ...runtime.Object) *MockKubernetesClient {
	return &MockKubernetesClient{
		Clientset: fake.NewSimpleClientset(objects...),
	}
}

func TestNewController(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 30 * time.Second,
			CPUThresholdPercent: 80.0,
			MemoryThresholdPercent: 85.0,
		},
		Remediation: config.RemediationConfig{
			Enabled:    true,
			MaxRetries: 3,
			DryRun:     false,
		},
		Notification: config.NotificationConfig{
			Slack: config.SlackConfig{
				Enabled: false,
			},
		},
	}

	// Create metrics collector
	metricsCollector := metrics.NewMetrics()

	// Test controller creation
	ctrl, err := NewController(cfg, metricsCollector)
	
	assert.NoError(t, err)
	assert.NotNil(t, ctrl)
	assert.NotNil(t, ctrl.client)
	assert.NotNil(t, ctrl.config)
	assert.NotNil(t, ctrl.metrics)
}

func TestControllerRun(t *testing.T) {
	// Create test objects
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	// Create mock client with test objects
	_ = NewMockKubernetesClient(pod, deployment)

	// Create configuration
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 100 * time.Millisecond, // Short for testing
			RulesFile:         "testdata/rules.yaml",
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true, // Use dry run for safety
		},
		Notification: config.NotificationConfig{
			Slack: config.SlackConfig{
				Enabled: false,
			},
		},
	}

	// Create controller
	metricsCollector := metrics.NewMetrics()
	ctrl, err := NewController(cfg, metricsCollector)
	assert.NoError(t, err)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run controller (this will execute a few detection cycles)
	err = ctrl.Run(ctx)
	assert.NoError(t, err)
}

func TestControllerProcessIssue(t *testing.T) {
	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	_ = NewMockKubernetesClient(pod)

	cfg := &config.Config{
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
		Notification: config.NotificationConfig{
			Slack: config.SlackConfig{
				Enabled: false,
			},
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := NewController(cfg, metricsCollector)
	assert.NoError(t, err)

	// Create a test issue
	issue := detection.Issue{
		RuleName:    "test-rule",
		Description: "Test issue",
		Severity:    "high",
		Resource:    pod,
		Namespace:   "default",
		Name:        "test-pod",
		Kind:        "Pod",
		Actions:     []string{"restart-pod"},
		DetectedAt:  time.Now(),
	}

	// Process the issue
	err = ctrl.processIssue(context.Background(), issue)
	assert.NoError(t, err)
}

func TestControllerGetClient(t *testing.T) {
	client := NewMockKubernetesClient()
	
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 30 * time.Second,
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := NewController(cfg, metricsCollector)
	assert.NoError(t, err)

	// Test GetClient method
	retrievedClient := ctrl.GetClient()
	assert.NotNil(t, retrievedClient)
	assert.Equal(t, client, retrievedClient)
}

func TestControllerConfigurationConversion(t *testing.T) {
	// Test namespace configuration conversion
	configNamespaces := map[string]config.NamespaceConfig{
		"default": {
			CrashLoop: config.CrashLoopConfig{
				RestartLimit:  5,
				CheckDuration: 30 * time.Second,
				Enabled:       true,
			},
			CPU: config.CPUConfig{
				ThresholdPercent: 75.0,
				CheckDuration:    60 * time.Second,
				Enabled:          true,
			},
		},
	}

	detectionNamespaces := convertConfigNamespaces(configNamespaces)
	
	assert.Equal(t, 1, len(detectionNamespaces))
	
	nsConfig, exists := detectionNamespaces["default"]
	assert.True(t, exists)
	assert.Equal(t, 5, nsConfig.CrashLoop.RestartLimit)
	assert.Equal(t, 75.0, nsConfig.CPU.ThresholdPercent)

	// Test remediation configuration conversion
	remediationNamespaces := map[string]config.NamespaceRemediationConfig{
		"default": {
			Enabled:             true,
			AutoRollbackEnabled: true,
			MaxRetries:          3,
			CooldownSeconds:     300,
		},
	}

	convertedRemediation := convertRemediationNamespaces(remediationNamespaces)
	
	assert.Equal(t, 1, len(convertedRemediation))
	
	remConfig, exists := convertedRemediation["default"]
	assert.True(t, exists)
	assert.True(t, remConfig.AutoRollbackEnabled)
	assert.Equal(t, 3, remConfig.MaxRetries)
}

func TestControllerErrorHandling(t *testing.T) {
	// Test with invalid configuration
	invalidCfg := &config.Config{
		Detection: config.DetectionConfig{
			RulesFile: "nonexistent-file.yaml",
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := NewController(invalidCfg, metricsCollector)
	
	// Should fail due to missing rules file
	assert.Error(t, err)
	assert.Nil(t, ctrl)
}

func TestControllerMetricsIntegration(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	_ = NewMockKubernetesClient(pod)

	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 50 * time.Millisecond,
			RulesFile:         "testdata/rules.yaml",
		},
		Remediation: config.RemediationConfig{
			Enabled: true,
			DryRun:  true,
		},
		Notification: config.NotificationConfig{
			Slack: config.SlackConfig{
				Enabled: false,
			},
		},
	}

	metricsCollector := metrics.NewMetrics()
	ctrl, err := NewController(cfg, metricsCollector)
	assert.NoError(t, err)

	// Run controller for a short time to generate metrics
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = ctrl.Run(ctx)
	assert.NoError(t, err)

	// Verify metrics were updated (implementation specific)
	// This would require accessing the metrics registry
}

func TestControllerGracefulShutdown(t *testing.T) {
	_ = NewMockKubernetesClient()

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
	ctrl, err := NewController(cfg, metricsCollector)
	assert.NoError(t, err)

	// Test graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start controller in goroutine
	done := make(chan error, 1)
	go func() {
		done <- ctrl.Run(ctx)
	}()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)
	
	// Cancel context to trigger shutdown
	cancel()

	// Wait for shutdown
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Controller did not shutdown gracefully")
	}
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
