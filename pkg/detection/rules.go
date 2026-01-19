package detection

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Rule represents a detection rule
type Rule struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Enabled     bool              `yaml:"enabled"`
	Conditions  []RuleCondition   `yaml:"conditions"`
	Actions     []string          `yaml:"actions"`
	Severity    string            `yaml:"severity"`
	Labels      map[string]string `yaml:"labels"`
}

// RuleCondition represents a condition in a rule
type RuleCondition struct {
	Resource  string                 `yaml:"resource"`
	Field     string                 `yaml:"field"`
	Operator  string                 `yaml:"operator"`
	Value     interface{}            `yaml:"value"`
	Duration  *metav1.Duration       `yaml:"duration"`
	MatchExpr map[string]interface{} `yaml:"matchExpr"`
}

// Issue represents a detected issue
type Issue struct {
	RuleName    string            `yaml:"ruleName"`
	Description string            `yaml:"description"`
	Severity    string            `yaml:"severity"`
	Resource    runtime.Object    `yaml:"resource"`
	Namespace   string            `yaml:"namespace"`
	Name        string            `yaml:"name"`
	Kind        string            `yaml:"kind"`
	Actions     []string          `yaml:"actions"`
	Labels      map[string]string `yaml:"labels"`
	DetectedAt  time.Time         `yaml:"detectedAt"`
}

// Detector represents the detection engine
type Detector struct {
	client kubernetes.Interface
	rules  []Rule
	config DetectionConfig
}

// DetectionConfig contains detection configuration
type DetectionConfig struct {
	RulesFile                 string                     `yaml:"rulesFile"`
	EvaluationInterval        time.Duration              `yaml:"evaluationInterval"`
	CrashLoopThreshold        int                        `yaml:"crashLoopThreshold"`
	FailedDeploymentThreshold int                        `yaml:"failedDeploymentThreshold"`
	CPUThresholdPercent       float64                    `yaml:"cpuThresholdPercent"`
	MemoryThresholdPercent    float64                    `yaml:"memoryThresholdPercent"`
	OOMKillThreshold          int                        `yaml:"oomKillThreshold"`
	Namespaces                map[string]NamespaceConfig `yaml:"namespaces"`
}

// NamespaceConfig contains namespace-specific detection settings
type NamespaceConfig struct {
	CrashLoop  CrashLoopConfig  `yaml:"crashloop"`
	Deployment DeploymentConfig `yaml:"deployment"`
	CPU        CPUConfig        `yaml:"cpu"`
	Memory     MemoryConfig     `yaml:"memory"`
}

// CrashLoopConfig contains crash loop detection settings for a namespace
type CrashLoopConfig struct {
	RestartLimit  int           `yaml:"restartLimit"`
	CheckDuration time.Duration `yaml:"checkDuration"`
	Enabled       bool          `yaml:"enabled"`
}

// DeploymentConfig contains deployment detection settings for a namespace
type DeploymentConfig struct {
	FailureThreshold int           `yaml:"failureThreshold"`
	CheckDuration    time.Duration `yaml:"checkDuration"`
	Enabled          bool          `yaml:"enabled"`
}

// CPUConfig contains CPU monitoring settings for a namespace
type CPUConfig struct {
	ThresholdPercent float64       `yaml:"thresholdPercent"`
	CheckDuration    time.Duration `yaml:"checkDuration"`
	Enabled          bool          `yaml:"enabled"`
}

// MemoryConfig contains memory monitoring settings for a namespace
type MemoryConfig struct {
	ThresholdPercent float64       `yaml:"thresholdPercent"`
	CheckDuration    time.Duration `yaml:"checkDuration"`
	OOMKillThreshold int           `yaml:"oomKillThreshold"`
	Enabled          bool          `yaml:"enabled"`
}

// NewDetector creates a new detector instance
func NewDetector(client kubernetes.Interface, config DetectionConfig) *Detector {
	return &Detector{
		client: client,
		config: config,
		rules:  []Rule{},
	}
}

// GetNamespaceConfig returns the namespace-specific configuration, falling back to defaults
func (d *Detector) GetNamespaceConfig(namespace string) NamespaceConfig {
	if nsConfig, exists := d.config.Namespaces[namespace]; exists {
		return nsConfig
	}

	// Return default configuration if namespace not found
	return NamespaceConfig{
		CrashLoop: CrashLoopConfig{
			RestartLimit:  d.config.CrashLoopThreshold,
			CheckDuration: 5 * time.Minute,
			Enabled:       true,
		},
		Deployment: DeploymentConfig{
			FailureThreshold: d.config.FailedDeploymentThreshold,
			CheckDuration:    10 * time.Minute,
			Enabled:          true,
		},
		CPU: CPUConfig{
			ThresholdPercent: d.config.CPUThresholdPercent,
			CheckDuration:    5 * time.Minute,
			Enabled:          true,
		},
		Memory: MemoryConfig{
			ThresholdPercent: d.config.MemoryThresholdPercent,
			CheckDuration:    5 * time.Minute,
			OOMKillThreshold: d.config.OOMKillThreshold,
			Enabled:          true,
		},
	}
}

// LoadRules loads detection rules from configuration
func (d *Detector) LoadRules() error {
	// For now, use built-in rules. In a real implementation,
	// this would load from a YAML file
	d.rules = []Rule{
		{
			Name:        "crash-loop-backoff",
			Description: "Detect pods in CrashLoopBackOff state",
			Enabled:     true,
			Conditions: []RuleCondition{
				{
					Resource: "Pod",
					Field:    "status.phase",
					Operator: "equals",
					Value:    "Running",
				},
				{
					Resource: "Pod",
					Field:    "status.containerStatuses[*].state.waiting.reason",
					Operator: "equals",
					Value:    "CrashLoopBackOff",
					Duration: &metav1.Duration{Duration: 5 * time.Minute},
				},
			},
			Actions:  []string{"restart-pod"},
			Severity: "high",
		},
		{
			Name:        "failed-deployment",
			Description: "Detect failed deployments",
			Enabled:     true,
			Conditions: []RuleCondition{
				{
					Resource: "Deployment",
					Field:    "status.conditions[*].type",
					Operator: "equals",
					Value:    "Progressing",
				},
				{
					Resource: "Deployment",
					Field:    "status.conditions[*].status",
					Operator: "equals",
					Value:    "False",
					Duration: &metav1.Duration{Duration: 10 * time.Minute},
				},
			},
			Actions:  []string{"rollback-deployment"},
			Severity: "high",
		},
		{
			Name:        "high-cpu-usage",
			Description: "Detect high CPU usage",
			Enabled:     true,
			Conditions: []RuleCondition{
				{
					Resource: "Pod",
					Field:    "metrics.cpu.usage",
					Operator: "greater_than",
					Value:    d.config.CPUThresholdPercent,
					Duration: &metav1.Duration{Duration: 5 * time.Minute},
				},
			},
			Actions:  []string{"scale-replicas"},
			Severity: "medium",
		},
		{
			Name:        "high-memory-usage",
			Description: "Detect high memory usage",
			Enabled:     true,
			Conditions: []RuleCondition{
				{
					Resource: "Pod",
					Field:    "metrics.memory.usage",
					Operator: "greater_than",
					Value:    d.config.MemoryThresholdPercent,
					Duration: &metav1.Duration{Duration: 5 * time.Minute},
				},
			},
			Actions:  []string{"restart-pod"},
			Severity: "high",
		},
		{
			Name:        "oom-kill-detected",
			Description: "Detect OOMKilled pods",
			Enabled:     true,
			Conditions: []RuleCondition{
				{
					Resource: "Pod",
					Field:    "status.containerStatuses[*].state.terminated.reason",
					Operator: "equals",
					Value:    "OOMKilled",
				},
			},
			Actions:  []string{"restart-pod", "scale-replicas"},
			Severity: "critical",
		},
	}
	return nil
}

// DetectIssues runs detection rules and returns detected issues
func (d *Detector) DetectIssues(ctx context.Context) ([]Issue, error) {
	logger := log.FromContext(ctx)
	var issues []Issue

	for _, rule := range d.rules {
		if !rule.Enabled {
			continue
		}

		logger.Info("Running detection rule", "rule", rule.Name)
		ruleIssues, err := d.evaluateRule(ctx, rule)
		if err != nil {
			logger.Error(err, "Failed to evaluate rule", "rule", rule.Name)
			continue
		}

		issues = append(issues, ruleIssues...)
	}

	return issues, nil
}

// evaluateRule evaluates a single rule
func (d *Detector) evaluateRule(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	switch rule.Name {
	case "crash-loop-backoff":
		return d.detectCrashLoopBackOff(ctx, rule)
	case "failed-deployment":
		return d.detectFailedDeployment(ctx, rule)
	case "high-cpu-usage":
		return d.detectHighCPUUsage(ctx, rule)
	case "high-memory-usage":
		return d.detectHighMemoryUsage(ctx, rule)
	case "oom-kill-detected":
		return d.detectOOMKilled(ctx, rule)
	default:
		return issues, fmt.Errorf("unknown rule: %s", rule.Name)
	}
}

// detectCrashLoopBackOff detects pods in CrashLoopBackOff state
func (d *Detector) detectCrashLoopBackOff(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Get namespace-specific configuration
		nsConfig := d.GetNamespaceConfig(pod.Namespace)

		// Skip if crash loop detection is disabled for this namespace
		if !nsConfig.CrashLoop.Enabled {
			continue
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil &&
				containerStatus.State.Waiting.Reason == "CrashLoopBackOff" {

				// Use namespace-specific restart limit
				if int(containerStatus.RestartCount) >= nsConfig.CrashLoop.RestartLimit {
					// Check if the condition has been met for the required duration
					if d.meetsDurationCondition(containerStatus.LastTerminationState.Terminated, &metav1.Duration{Duration: nsConfig.CrashLoop.CheckDuration}) {
						issue := Issue{
							RuleName:    rule.Name,
							Description: fmt.Sprintf("%s (restart limit: %d)", rule.Description, nsConfig.CrashLoop.RestartLimit),
							Severity:    rule.Severity,
							Resource:    pod.DeepCopyObject(),
							Namespace:   pod.Namespace,
							Name:        pod.Name,
							Kind:        "Pod",
							Actions:     rule.Actions,
							Labels:      rule.Labels,
							DetectedAt:  time.Now(),
						}
						issues = append(issues, issue)
					}
				}
			}
		}
	}

	return issues, nil
}

// detectFailedDeployment detects failed deployments
func (d *Detector) detectFailedDeployment(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	deployments, err := d.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		// Get namespace-specific configuration
		nsConfig := d.GetNamespaceConfig(deployment.Namespace)

		// Skip if deployment failure detection is disabled for this namespace
		if !nsConfig.Deployment.Enabled {
			continue
		}

		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing &&
				condition.Status == corev1.ConditionFalse &&
				condition.Reason == "ProgressDeadlineExceeded" {

				// Check if the condition has been met for the required duration
				if d.meetsDurationCondition(nil, &metav1.Duration{Duration: nsConfig.Deployment.CheckDuration}) {
					issue := Issue{
						RuleName:    rule.Name,
						Description: fmt.Sprintf("%s (failure threshold: %d)", rule.Description, nsConfig.Deployment.FailureThreshold),
						Severity:    rule.Severity,
						Resource:    deployment.DeepCopyObject(),
						Namespace:   deployment.Namespace,
						Name:        deployment.Name,
						Kind:        "Deployment",
						Actions:     rule.Actions,
						Labels:      rule.Labels,
						DetectedAt:  time.Now(),
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, nil
}

// detectHighCPUUsage detects high CPU usage (simplified implementation)
func (d *Detector) detectHighCPUUsage(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	// This is a simplified implementation. In a real scenario,
	// you would use metrics server or Prometheus to get actual CPU metrics

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Get namespace-specific configuration
		nsConfig := d.GetNamespaceConfig(pod.Namespace)

		// Skip if CPU monitoring is disabled for this namespace
		if !nsConfig.CPU.Enabled {
			continue
		}

		// Simulate high CPU detection based on restart count
		// In reality, you'd query metrics server or Prometheus
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if int(containerStatus.RestartCount) > int(nsConfig.CPU.ThresholdPercent) {
				// Check if the condition has been met for the required duration
				if d.meetsDurationCondition(nil, &metav1.Duration{Duration: nsConfig.CPU.CheckDuration}) {
					issue := Issue{
						RuleName:    rule.Name,
						Description: fmt.Sprintf("%s (threshold: %.1f%%)", rule.Description, nsConfig.CPU.ThresholdPercent),
						Severity:    rule.Severity,
						Resource:    pod.DeepCopyObject(),
						Namespace:   pod.Namespace,
						Name:        pod.Name,
						Kind:        "Pod",
						Actions:     rule.Actions,
						Labels:      rule.Labels,
						DetectedAt:  time.Now(),
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, nil
}

// detectHighMemoryUsage detects high memory usage in pods
func (d *Detector) detectHighMemoryUsage(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Get namespace-specific configuration
		nsConfig := d.GetNamespaceConfig(pod.Namespace)

		// Skip if memory monitoring is disabled for this namespace
		if !nsConfig.Memory.Enabled {
			continue
		}

		// Simulate high memory detection based on restart count and container status
		// In reality, you'd query metrics server or Prometheus for actual memory usage
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// Check for memory pressure indicators
			if containerStatus.RestartCount > 3 ||
				(containerStatus.State.Waiting != nil &&
					(containerStatus.State.Waiting.Reason == "CrashLoopBackOff" ||
						containerStatus.State.Waiting.Reason == "ContainerCreating")) {

				// Check if the condition has been met for the required duration
				if d.meetsDurationCondition(containerStatus.LastTerminationState.Terminated, &metav1.Duration{Duration: nsConfig.Memory.CheckDuration}) {
					issue := Issue{
						RuleName:    rule.Name,
						Description: fmt.Sprintf("%s (threshold: %.1f%%)", rule.Description, nsConfig.Memory.ThresholdPercent),
						Severity:    rule.Severity,
						Resource:    pod.DeepCopyObject(),
						Namespace:   pod.Namespace,
						Name:        pod.Name,
						Kind:        "Pod",
						Actions:     rule.Actions,
						Labels:      rule.Labels,
						DetectedAt:  time.Now(),
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, nil
}

// detectOOMKilled detects pods that have been OOMKilled
func (d *Detector) detectOOMKilled(ctx context.Context, rule Rule) ([]Issue, error) {
	var issues []Issue

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Get namespace-specific configuration
		nsConfig := d.GetNamespaceConfig(pod.Namespace)

		// Skip if memory monitoring is disabled for this namespace
		if !nsConfig.Memory.Enabled {
			continue
		}

		// Check for OOMKilled containers
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Terminated != nil &&
				containerStatus.State.Terminated.Reason == "OOMKilled" {

				// Count OOMKills for this container
				oomKillCount := 0
				if containerStatus.RestartCount > 0 {
					oomKillCount = int(containerStatus.RestartCount)
				}

				// Check if OOMKill threshold is exceeded
				if oomKillCount >= nsConfig.Memory.OOMKillThreshold {
					issue := Issue{
						RuleName:    rule.Name,
						Description: fmt.Sprintf("%s (OOMKills: %d, threshold: %d)", rule.Description, oomKillCount, nsConfig.Memory.OOMKillThreshold),
						Severity:    rule.Severity,
						Resource:    pod.DeepCopyObject(),
						Namespace:   pod.Namespace,
						Name:        pod.Name,
						Kind:        "Pod",
						Actions:     rule.Actions,
						Labels:      rule.Labels,
						DetectedAt:  time.Now(),
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, nil
}

// meetsDurationCondition checks if a condition has been met for the required duration
func (d *Detector) meetsDurationCondition(terminated *corev1.ContainerStateTerminated, duration *metav1.Duration) bool {
	if terminated == nil || duration == nil {
		return false
	}

	return time.Since(terminated.FinishedAt.Time) >= duration.Duration
}
