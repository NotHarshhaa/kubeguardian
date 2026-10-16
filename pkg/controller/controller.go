package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	"github.com/NotHarshhaa/kubeguardian/pkg/notification"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
	"github.com/NotHarshhaa/kubeguardian/pkg/version"
)

// Controller represents the main KubeGuardian controller
type Controller struct {
	client         kubernetes.Interface
	config         *config.Config
	detector       *detection.Detector
	nodeDetector   *detection.NodeDetector
	remediator     *remediation.Engine
	notifier       *notification.Manager
	metrics        *metrics.Metrics
	configReloader *config.Reloader
}

// NewController creates a new controller instance
func NewController(cfg *config.Config, metricsCollector *metrics.Metrics) (*Controller, error) {
	// Create Kubernetes client
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
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
		Namespaces:                convertConfigNamespaces(cfg.Detection.Namespaces),
		NodeDetection: detection.NodeDetectorConfig{
			Enabled:        cfg.Detection.NodeDetection.Enabled,
			CheckInterval:  cfg.Detection.NodeDetection.CheckInterval,
			AutoCordon:     cfg.Detection.NodeDetection.AutoCordon,
			AutoUncord:     cfg.Detection.NodeDetection.AutoUncord,
			CordonDuration: cfg.Detection.NodeDetection.CordonDuration,
		},
	}
	detector := detection.NewDetector(client, detectionConfig)
	if err := detector.LoadRules(); err != nil {
		return nil, fmt.Errorf("failed to load detection rules: %w", err)
	}

	// Create node detector
	nodeDetector := detection.NewNodeDetector(client)

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

	// Create notification manager with all enabled channels
	notifier := notification.NewManager()

	// Slack notifier
	if cfg.Notification.Slack.Enabled {
		slackConfig := notification.SlackConfig{
			Token:     cfg.Notification.Slack.Token,
			Channel:   cfg.Notification.Slack.Channel,
			Username:  cfg.Notification.Slack.Username,
			IconEmoji: cfg.Notification.Slack.IconEmoji,
		}
		notifier.Register(notification.NewSlackNotifier(slackConfig))
	}

	// Email notifier
	if cfg.Notification.Email.Enabled {
		emailConfig := notification.EmailConfig{
			Enabled:         cfg.Notification.Email.Enabled,
			Host:            cfg.Notification.Email.Host,
			Port:            cfg.Notification.Email.Port,
			Username:        cfg.Notification.Email.Username,
			Password:        cfg.Notification.Email.Password,
			From:            cfg.Notification.Email.From,
			To:              cfg.Notification.Email.To,
			UseTLS:          cfg.Notification.Email.UseTLS,
			SkipTLSVerify:   cfg.Notification.Email.SkipTLSVerify,
		}
		notifier.Register(notification.NewEmailNotifier(emailConfig))
	}

	// Webhook notifier
	if cfg.Notification.Webhook.Enabled {
		webhookConfig := notification.WebhookConfig{
			Enabled:     cfg.Notification.Webhook.Enabled,
			URL:         cfg.Notification.Webhook.URL,
			Method:      cfg.Notification.Webhook.Method,
			Headers:     cfg.Notification.Webhook.Headers,
			Timeout:     cfg.Notification.Webhook.Timeout,
			Secret:      cfg.Notification.Webhook.Secret,
			PayloadType: cfg.Notification.Webhook.PayloadType,
		}
		notifier.Register(notification.NewWebhookNotifier(webhookConfig))
	}

	return &Controller{
		client:      client,
		config:      cfg,
		detector:    detector,
		nodeDetector: nodeDetector,
		remediator:  remediator,
		notifier:    notifier,
		metrics:     metricsCollector,
	}, nil
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// Test all notifier connections
	if err := c.notifier.TestConnection(ctx); err != nil {
		logger.Error(err, "Some notifier connection tests failed, continuing with available channels")
	}

	// Send startup notification via all enabled channels
	if err := c.notifier.SendStartupNotification(ctx, version.Version); err != nil {
		logger.Error(err, "Failed to send startup notification")
	}

	// Start the main detection loop
	ticker := time.NewTicker(c.config.Detection.EvaluationInterval)
	defer ticker.Stop()

	// Start node detection ticker
	nodeTicker := time.NewTicker(c.config.Detection.NodeDetection.CheckInterval)
	defer nodeTicker.Stop()

	// Start cooldown cleanup goroutine
	cleanupTicker := time.NewTicker(10 * time.Minute)
	defer cleanupTicker.Stop()

	logger.Info("KubeGuardian started",
		"evaluationInterval", c.config.Detection.EvaluationInterval,
		"nodeCheckInterval", c.config.Detection.NodeDetection.CheckInterval,
		"enabledChannels", c.notifier.Enabled(),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("KubeGuardian stopping")
			return nil
		case <-ticker.C:
			if err := c.runDetectionCycle(ctx); err != nil {
				logger.Error(err, "Detection cycle failed")
			}
		case <-nodeTicker.C:
			if c.config.Detection.NodeDetection.Enabled {
				if err := c.runNodeDetection(ctx); err != nil {
					logger.Error(err, "Node detection cycle failed")
				}
			}
		case <-cleanupTicker.C:
			c.remediator.CleanupCooldowns()
		}
	}
}

// runNodeDetection runs node health detection and notifies on issues
func (c *Controller) runNodeDetection(ctx context.Context) error {
	logger := log.FromContext(ctx)

	issues, err := c.nodeDetector.DetectNodeIssues(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect node issues: %w", err)
	}

	if len(issues) == 0 {
		return nil
	}

	logger.Info("Node issues detected", "count", len(issues))

	for _, issue := range issues {
		logger.Info("Node issue", "node", issue.NodeName, "condition", issue.Condition, "severity", issue.Severity, "message", issue.Message)

		// Send notification for node issue
		nodeIssue := detection.Issue{
			RuleName:    "node-" + issue.Condition,
			Description: fmt.Sprintf("Node %s: %s - %s", issue.NodeName, issue.Condition, issue.Message),
			Severity:    issue.Severity,
			Namespace:   "",
			Name:        issue.NodeName,
			Kind:        "Node",
			Actions:     []string{"notify-only"},
			Labels:      issue.Labels,
			DetectedAt:  issue.DetectedAt,
		}

		if err := c.notifier.SendIssueNotification(ctx, nodeIssue); err != nil {
			logger.Error(err, "Failed to send node issue notification")
		}

		// Auto-cordon if enabled
		if c.config.Detection.NodeDetection.AutoCordon && issue.Severity == "critical" {
			c.cordonNode(ctx, issue.NodeName)
		}
	}

	return nil
}

// cordonNode cordons a node to prevent new pod scheduling
func (c *Controller) cordonNode(ctx context.Context, nodeName string) {
	// Get the current node
	node, err := c.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to get node for cordoning", "node", nodeName)
		return
	}

	// Check if already unschedulable
	if node.Spec.Unschedulable {
		log.FromContext(ctx).Info("Node already cordoned", "node", nodeName)
		return
	}

	// Patch the node to mark it unschedulable
	patch := []byte(`{"spec":{"unschedulable":true}}`)
	_, err = c.client.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to cordon node", "node", nodeName)
		return
	}

	log.FromContext(ctx).Info("Successfully cordoned node", "node", nodeName)
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

	// Send issue notification via all enabled channels
	if err := c.notifier.SendIssueNotification(ctx, issue); err != nil {
		logger.Error(err, "Failed to send issue notification")
		c.metrics.RecordNotification("issue", "failed")
	} else {
		c.metrics.RecordNotification("issue", "success")
	}

	// Execute remediation actions
	for _, action := range issue.Actions {
		if action == "notify-only" {
			logger.Info("Skipping remediation for notify-only action", "resource", issue.Name)
			continue
		}

		logger.Info("Executing remediation action", "action", action, "resource", issue.Name)
		start := time.Now()

		result, err := c.remediator.ExecuteAction(ctx, action, issue.Resource, issue.Namespace)
		if err != nil {
			logger.Error(err, "Failed to execute remediation action", "action", action)
			c.metrics.RecordRemediation(action, "error", issue.Namespace, time.Since(start))
			// Continue with other actions even if one fails
			continue
		}

		if result != nil {
			// Record remediation metrics
			status := "success"
			if !result.Success {
				status = "failed"
			}
			c.metrics.RecordRemediation(action, status, issue.Namespace, time.Since(start))

			// Send remediation notification via all enabled channels
			if err := c.notifier.SendRemediationNotification(ctx, issue, *result); err != nil {
				logger.Error(err, "Failed to send remediation notification")
				c.metrics.RecordNotification("remediation", "failed")
			} else {
				c.metrics.RecordNotification("remediation", "success")
			}

			logger.Info("Remediation action completed",
				"action", action,
				"success", result.Success,
				"message", result.Message,
			)
		}
	}

	return nil
}

// GetRemediationHistory returns remediation history entries
func (c *Controller) GetRemediationHistory(namespace, action string, limit int) []remediation.HistoryEntry {
	return c.remediator.History(namespace, action, limit)
}

// GetRemediationStats returns remediation statistics
func (c *Controller) GetRemediationStats() remediation.HistoryStats {
	return c.remediator.HistoryStats()
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
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	mgr, err := manager.New(restConfig, manager.Options{
		HealthProbeBindAddress: cfg.Controller.ProbeAddr,
		LeaderElection:         cfg.Controller.LeaderElection,
		LeaderElectionID:       "kubeguardian-leader-election",
		Cache: cache.Options{
			SyncPeriod: &cfg.Controller.SyncPeriod,
		},
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
