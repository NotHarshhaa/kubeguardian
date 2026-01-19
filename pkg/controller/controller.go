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
}

// NewController creates a new controller instance
func NewController(cfg *config.Config) (*Controller, error) {
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
	}
	remediator := remediation.NewEngine(client, remediationConfig)

	// Create Slack notifier
	slackConfig := notification.SlackConfig{
		Enabled:   cfg.Notification.Slack.Enabled,
		Token:     cfg.Notification.Slack.Token,
		Channel:   cfg.Notification.Slack.Channel,
		Username:  cfg.Notification.Slack.Username,
		IconEmoji: cfg.Notification.Slack.IconEmoji,
	}
	slackNotifier := notification.NewSlackNotifier(slackConfig)

	return &Controller{
		client:        client,
		config:        cfg,
		detector:      detector,
		remediator:    remediator,
		slackNotifier: slackNotifier,
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
		}
	}
}

// runDetectionCycle runs a single detection and remediation cycle
func (c *Controller) runDetectionCycle(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting detection cycle")

	// Detect issues
	issues, err := c.detector.DetectIssues(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect issues: %w", err)
	}

	if len(issues) == 0 {
		logger.Info("No issues detected")
		return nil
	}

	logger.Info("Issues detected", "count", len(issues))

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
		}
	}

	// Execute remediation actions
	for _, action := range issue.Actions {
		logger.Info("Executing remediation action", "action", action, "resource", issue.Name)

		result, err := c.remediator.ExecuteAction(ctx, action, issue.Resource, issue.Namespace)
		if err != nil {
			logger.Error(err, "Failed to execute remediation action", "action", action)
			continue
		}

		// Send remediation notification
		if c.slackNotifier != nil {
			if err := c.slackNotifier.SendRemediationNotification(ctx, issue, *result); err != nil {
				logger.Error(err, "Failed to send remediation notification")
			}
		}

		logger.Info("Remediation action completed", "action", action, "success", result.Success, "message", result.Message)
	}

	return nil
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
