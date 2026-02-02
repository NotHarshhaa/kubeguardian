package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	"github.com/NotHarshhaa/kubeguardian/pkg/notification"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
)

// Controller represents the main KubeGuardian controller
type Controller struct {
	client        kubernetes.Interface
	config        *config.Config
	detector      *detection.Detector
	remediator    *remediation.Engine
	slackNotifier *notification.SlackNotifier
	metrics       *metrics.Metrics
}

// NewController creates a new controller instance
func NewController(cfg *config.Config, metricsCollector *metrics.Metrics) (*Controller, error) {
	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create detector
	detectionConfig := detection.DetectionConfig{
		RulesFile:                 cfg.Detection.RulesFile,
		EvaluationInterval:        cfg.Detection.EvaluationInterval,
		CrashLoopThreshold:        cfg.Detection.CrashLoopThreshold,
		FailedDeploymentThreshold: cfg.Detection.FailedDeploymentThreshold,
		CPUThresholdPercent:       cfg.Detection.CPUThresholdPercent,
		MemoryThresholdPercent:    cfg.Detection.MemoryThresholdPercent,
		OOMKillThreshold:          cfg.Detection.OOMKillThreshold,
		Namespaces:                 convertConfigNamespaces(cfg.Detection.Namespaces),
	}
	detector := detection.NewDetector(client, detectionConfig)
	if err := detector.LoadRules(); err != nil {
		return nil, fmt.Errorf("failed to load detection rules: %w", err)
	}

	// Create remediation engine
	remediationConfig := remediation.RemediationConfig{
		Enabled:             cfg.Remediation.Enabled,
		MaxRetries:          cfg.Remediation.MaxRetries,
		RetryInterval:       cfg.Remediation.RetryInterval,
		DryRun:              cfg.Remediation.DryRun,
		AutoRollbackEnabled: cfg.Remediation.AutoRollbackEnabled,
		AutoScaleEnabled:    cfg.Remediation.AutoScaleEnabled,
		CooldownSeconds:     cfg.Remediation.CooldownSeconds,
		Namespaces:          convertRemediationNamespaces(cfg.Remediation.Namespaces),
	}
	remediator := remediation.NewEngine(client, remediationConfig)

	// Create Slack notifier if enabled
	var slackNotifier *notification.SlackNotifier
	if cfg.Notification.Slack.Enabled {
		slackConfig := notification.SlackConfig{
			Token:     cfg.Notification.Slack.Token,
			Channel:   cfg.Notification.Slack.Channel,
			Username:  cfg.Notification.Slack.Username,
			IconEmoji: cfg.Notification.Slack.IconEmoji,
		}
		slackNotifier = notification.NewSlackNotifier(slackConfig)
	}

	return &Controller{
		client:        client,
		config:        cfg,
		detector:      detector,
		remediator:    remediator,
		slackNotifier: slackNotifier,
		metrics:       metricsCollector,
	}, nil
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// Test Slack connection if enabled
	if c.slackNotifier != nil {
		if err := c.slackNotifier.TestConnection(ctx); err != nil {
			logger.Error(err, "Slack connection test failed, continuing without Slack notifications")
		} else {
			// Send startup notification
			if err := c.slackNotifier.SendStartupNotification(ctx, "v1.0.0"); err != nil {
				logger.Error(err, "Failed to send startup notification")
			}
		}
	}

	// Start the main detection loop
	ticker := time.NewTicker(c.config.Detection.EvaluationInterval)
	defer ticker.Stop()

	// Start cooldown cleanup goroutine
	cleanupTicker := time.NewTicker(10 * time.Minute)
	defer cleanupTicker.Stop()

	logger.Info("KubeGuardian started", "evaluationInterval", c.config.Detection.EvaluationInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("KubeGuardian stopping")
			return nil
		case <-ticker.C:
			if err := c.runDetectionCycle(ctx); err != nil {
				logger.Error(err, "Detection cycle failed")
			}
		case <-cleanupTicker.C:
			c.remediator.CleanupCooldowns()
		}
	}
}

// runDetectionCycle runs a single detection and remediation cycle
func (c *Controller) runDetectionCycle(ctx context.Context) error {
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Info("Starting detection cycle")

	// Detect issues
	issues, err := c.detector.DetectIssues(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect issues: %w", err)
	}

	// Record detection metrics
	c.metrics.UpdateLastDetectionTime()
	c.metrics.RecordDetectionDuration("detection_cycle", time.Since(start))

	if len(issues) == 0 {
		logger.Info("No issues detected")
		return nil
	}

	logger.Info("Issues detected", "count", len(issues))

	// Record metrics for each detected issue
	for _, issue := range issues {
		c.metrics.RecordIssueDetected(issue.RuleName, issue.Severity, issue.Namespace)
	}

	// Process each issue
	for _, issue := range issues {
		if err := c.processIssue(ctx, issue); err != nil {
			logger.Error(err, "Failed to process issue", "rule", issue.RuleName, "resource", issue.Name)
		}
	}

	return nil
}

// processIssue processes a single detected issue
func (c *Controller) processIssue(ctx context.Context, issue detection.Issue) error {
	logger := log.FromContext(ctx)
	logger.Info("Processing issue", "rule", issue.RuleName, "resource", issue.Name, "severity", issue.Severity)

	// Send issue notification
	if c.slackNotifier != nil {
		if err := c.slackNotifier.SendIssueNotification(ctx, issue); err != nil {
			logger.Error(err, "Failed to send issue notification")
			c.metrics.RecordNotification("issue", "failed")
		} else {
			c.metrics.RecordNotification("issue", "success")
		}
	}

	// Execute remediation actions
	for _, action := range issue.Actions {
		logger.Info("Executing remediation action", "action", action, "resource", issue.Name)
		start := time.Now()

		result, err := c.remediator.ExecuteAction(ctx, action, issue.Resource, issue.Namespace)
		if err != nil {
			logger.Error(err, "Failed to execute remediation action", "action", action)
			c.metrics.RecordRemediation(action, "error", issue.Namespace, time.Since(start))
			// Continue with other actions even if one fails
			continue
		}

		// Only send notification if result is not nil
		if result != nil {
			// Record remediation metrics
			status := "success"
			if !result.Success {
				status = "failed"
			}
			c.metrics.RecordRemediation(action, status, issue.Namespace, time.Since(start))

			// Send remediation notification
			if c.slackNotifier != nil {
				if err := c.slackNotifier.SendRemediationNotification(ctx, issue, *result); err != nil {
					logger.Error(err, "Failed to send remediation notification")
					c.metrics.RecordNotification("remediation", "failed")
				} else {
					c.metrics.RecordNotification("remediation", "success")
				}
			}

			logger.Info("Remediation action completed", "action", action, "success", result.Success, "message", result.Message)
		}
	}

	return nil
}

// convertConfigNamespaces converts config namespace configs to detection namespace configs
func convertConfigNamespaces(configNs map[string]config.NamespaceConfig) map[string]detection.NamespaceConfig {
	result := make(map[string]detection.NamespaceConfig)
	for name, ns := range configNs {
		result[name] = detection.NamespaceConfig{
			CrashLoop: detection.CrashLoopConfig{
				RestartLimit:  ns.CrashLoop.RestartLimit,
				CheckDuration: ns.CrashLoop.CheckDuration,
				Enabled:       ns.CrashLoop.Enabled,
			},
			Deployment: detection.DeploymentConfig{
				FailureThreshold: ns.Deployment.FailureThreshold,
				CheckDuration:    ns.Deployment.CheckDuration,
				Enabled:          ns.Deployment.Enabled,
			},
			CPU: detection.CPUConfig{
				ThresholdPercent: ns.CPU.ThresholdPercent,
				CheckDuration:    ns.CPU.CheckDuration,
				Enabled:          ns.CPU.Enabled,
			},
			Memory: detection.MemoryConfig{
				ThresholdPercent: ns.Memory.ThresholdPercent,
				OOMKillThreshold: ns.Memory.OOMKillThreshold,
				CheckDuration:    ns.Memory.CheckDuration,
				Enabled:          ns.Memory.Enabled,
			},
		}
	}
	return result
}

// convertRemediationNamespaces converts config namespace configs to remediation namespace configs
func convertRemediationNamespaces(configNs map[string]config.NamespaceRemediationConfig) map[string]remediation.NamespaceRemediationConfig {
	result := make(map[string]remediation.NamespaceRemediationConfig)
	for name, ns := range configNs {
		result[name] = remediation.NamespaceRemediationConfig{
			Enabled:             ns.Enabled,
			AutoRollbackEnabled: ns.AutoRollbackEnabled,
			AutoScaleEnabled:    ns.AutoScaleEnabled,
			MaxRetries:          ns.MaxRetries,
			RetryInterval:       ns.RetryInterval,
			CooldownSeconds:     ns.CooldownSeconds,
		}
	}
	return result
}

// GetClient returns the Kubernetes client
func (c *Controller) GetClient() kubernetes.Interface {
	return c.client
}

// SetupManager sets up the controller-runtime manager
func SetupManager(cfg *config.Config) (manager.Manager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	mgr, err := manager.New(config, manager.Options{
		HealthProbeBindAddress: cfg.Controller.ProbeAddr,
		LeaderElection:         cfg.Controller.LeaderElection,
		LeaderElectionID:       "kubeguardian-leader-election",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	return mgr, nil
}

// StartManager starts the controller-runtime manager
func StartManager(ctx context.Context, mgr manager.Manager) error {
	logger := log.FromContext(ctx)

	// Setup signals
	ctx = signals.SetupSignalHandler()

	// Start the manager
	logger.Info("Starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	return nil
}
