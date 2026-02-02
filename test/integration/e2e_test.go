//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestEndToEndDetectionAndRemediation(t *testing.T) {
	// Skip if not running in cluster
	if os.Getenv("KUBECONFIG") == "" && os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		t.Skip("Skipping integration test: no Kubernetes configuration available")
	}

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatalf("Failed to get in-cluster config: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create test namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeguardian-test",
		},
	}

	_, err = client.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}
	defer client.CoreV1().Namespaces().Delete(context.Background(), "kubeguardian-test", metav1.DeleteOptions{})

	// Create test deployment with issues
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "kubeguardian-test",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = client.AppsV1().Deployments("kubeguardian-test").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test deployment: %v", err)
	}

	// Create KubeGuardian configuration
	cfg := &config.Config{
		Detection: config.DetectionConfig{
			EvaluationInterval: 5 * time.Second,
			CPUThresholdPercent: 80.0,
			MemoryThresholdPercent: 85.0,
			Namespaces: map[string]config.NamespaceConfig{
				"kubeguardian-test": {
					CrashLoop: config.CrashLoopConfig{
						RestartLimit:  3,
						CheckDuration: 30 * time.Second,
						Enabled:       true,
					},
					CPU: config.CPUConfig{
						ThresholdPercent: 80.0,
						CheckDuration:    30 * time.Second,
						Enabled:          true,
					},
				},
			},
		},
		Remediation: config.RemediationConfig{
			Enabled:     true,
			MaxRetries:  3,
			DryRun:      true, // Use dry run for safety
			CooldownSeconds: 60,
			Namespaces: map[string]config.NamespaceRemediationConfig{
				"kubeguardian-test": {
					Enabled:             true,
					AutoRollbackEnabled: true,
					AutoScaleEnabled:    true,
					MaxRetries:          3,
					CooldownSeconds:     60,
				},
			},
		},
		Notification: config.NotificationConfig{
			Slack: config.SlackConfig{
				Enabled: false, // Disable for testing
			},
		},
	}

	// Create controller
	metricsCollector := metrics.NewMetrics()
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	// Run controller for a short time
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = ctrl.Run(ctx)
	if err != nil {
		t.Errorf("Controller run failed: %v", err)
	}

	// Verify deployment still exists (dry run should not affect it)
	_, err = client.AppsV1().Deployments("kubeguardian-test").Get(context.Background(), "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Test deployment should still exist after dry run: %v", err)
	}
}

func TestEndToEndHealthChecks(t *testing.T) {
	// Skip if not running in cluster
	if os.Getenv("KUBECONFIG") == "" && os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		t.Skip("Skipping integration test: no Kubernetes configuration available")
	}

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatalf("Failed to get in-cluster config: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create health check manager
	healthChecker := health.NewHealthCheck("v1.5.0", client)

	// Run health checks
	ctx := context.Background()
	healthChecker.RunChecks(ctx)

	// Check overall health
	overall := healthChecker.Overall()
	if overall != health.StatusHealthy {
		t.Errorf("Overall health status should be healthy, got %s", overall)
	}

	// Check individual checks
	results := healthChecker.Results()
	if len(results) == 0 {
		t.Error("No health check results found")
	}

	// Verify Kubernetes API check
	apiCheck, exists := results["kubernetes-api"]
	if !exists {
		t.Error("Kubernetes API health check not found")
	} else if apiCheck.Status != health.StatusHealthy {
		t.Errorf("Kubernetes API health check failed: %s", apiCheck.Message)
	}
}

func TestEndToEndMetricsCollection(t *testing.T) {
	// Skip if not running in cluster
	if os.Getenv("KUBECONFIG") == "" && os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		t.Skip("Skipping integration test: no Kubernetes configuration available")
	}

	// Create metrics collector
	metricsCollector := metrics.NewMetrics()

	// Simulate some activity
	for i := 0; i < 10; i++ {
		metricsCollector.RecordIssueDetected("test-rule", "medium", "test-namespace")
		metricsCollector.RecordRemediation("restart-pod", "success", "test-namespace", time.Millisecond)
		metricsCollector.RecordAPICall("GET", "pods", "success", 100*time.Millisecond)
	}

	// Update metrics
	metricsCollector.UpdateUptime()
	metricsCollector.UpdateLastDetectionTime()
	metricsCollector.RecordDetectionDuration("test-cycle", 500*time.Millisecond)

	// Test that metrics are accessible (implementation specific)
	// This would typically involve checking the Prometheus registry
}

func TestEndToEndConfigurationValidation(t *testing.T) {
	// Test various configuration scenarios
	testCases := []struct {
		name     string
		config   *config.Config
		wantErr  bool
	}{
		{
			name: "valid production config",
			config: &config.Config{
				Detection: config.DetectionConfig{
					EvaluationInterval: 30 * time.Second,
					CPUThresholdPercent: 80.0,
					MemoryThresholdPercent: 85.0,
				},
				Remediation: config.RemediationConfig{
					Enabled:     true,
					MaxRetries:  3,
					DryRun:      false,
					CooldownSeconds: 300,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - too frequent",
			config: &config.Config{
				Detection: config.DetectionConfig{
					EvaluationInterval: 100 * time.Millisecond,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - excessive retries",
			config: &config.Config{
				Remediation: config.RemediationConfig{
					MaxRetries: 1000,
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.config.Validate()
			
			if tc.wantErr && len(result.Errors) == 0 {
				t.Errorf("Expected validation errors for config: %s", tc.name)
			}
			
			if !tc.wantErr && len(result.Errors) > 0 {
				t.Errorf("Unexpected validation errors for config %s: %v", tc.name, result.Errors)
			}
		})
	}
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
