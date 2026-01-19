package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
)

// SlackNotifier handles Slack notifications
type SlackNotifier struct {
	client *slack.Client
	config SlackConfig
}

// SlackConfig contains Slack configuration
type SlackConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Token     string `yaml:"token"`
	Channel   string `yaml:"channel"`
	Username  string `yaml:"username"`
	IconEmoji string `yaml:"iconEmoji"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(config SlackConfig) *SlackNotifier {
	if !config.Enabled {
		return nil
	}

	client := slack.New(config.Token)
	return &SlackNotifier{
		client: client,
		config: config,
	}
}

// SendIssueNotification sends a notification about a detected issue
func (s *SlackNotifier) SendIssueNotification(ctx context.Context, issue detection.Issue) error {
	if s == nil || !s.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	// Create Slack attachment
	attachment := slack.Attachment{
		Color: s.getColorBySeverity(issue.Severity),
		Title: fmt.Sprintf("🚑 KubeGuardian Alert: %s", issue.RuleName),
		Text:  issue.Description,
		Fields: []slack.AttachmentField{
			{
				Title: "Resource",
				Value: fmt.Sprintf("%s/%s", issue.Kind, issue.Name),
				Short: true,
			},
			{
				Title: "Namespace",
				Value: issue.Namespace,
				Short: true,
			},
			{
				Title: "Severity",
				Value: strings.ToUpper(issue.Severity),
				Short: true,
			},
			{
				Title: "Detected At",
				Value: issue.DetectedAt.Format("2006-01-02 15:04:05"),
				Short: true,
			},
			{
				Title: "Actions",
				Value: strings.Join(issue.Actions, ", "),
				Short: false,
			},
		},
		Footer:     "KubeGuardian",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
		Ts:         json.Number(fmt.Sprintf("%d", issue.DetectedAt.Unix())),
	}

	// Send the message
	_, _, err := s.client.PostMessage(
		s.config.Channel,
		slack.MsgOptionText("Issue detected in Kubernetes cluster", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)

	if err != nil {
		logger.Error(err, "Failed to send Slack notification")
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}

	logger.Info("Successfully sent Slack notification for issue", "rule", issue.RuleName, "resource", issue.Name)
	return nil
}

// SendRemediationNotification sends a notification about a remediation action
func (s *SlackNotifier) SendRemediationNotification(ctx context.Context, issue detection.Issue, result remediation.Result) error {
	if s == nil || !s.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	// Create Slack attachment
	var color string
	if result.Success {
		color = "good"
	} else {
		color = "danger"
	}

	attachment := slack.Attachment{
		Color: color,
		Title: fmt.Sprintf("🔧 KubeGuardian Action: %s", result.Action),
		Text:  result.Message,
		Fields: []slack.AttachmentField{
			{
				Title: "Resource",
				Value: fmt.Sprintf("%s/%s", issue.Kind, issue.Name),
				Short: true,
			},
			{
				Title: "Namespace",
				Value: issue.Namespace,
				Short: true,
			},
			{
				Title: "Status",
				Value: func() string {
					if result.Success {
						return "✅ Success"
					}
					return "❌ Failed"
				}(),
				Short: true,
			},
			{
				Title: "Duration",
				Value: result.Duration.String(),
				Short: true,
			},
			{
				Title: "Issue",
				Value: issue.RuleName,
				Short: true,
			},
			{
				Title: "Executed At",
				Value: result.ExecutedAt.Format("2006-01-02 15:04:05"),
				Short: true,
			},
		},
		Footer:     "KubeGuardian",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
		Ts:         json.Number(fmt.Sprintf("%d", result.ExecutedAt.Unix())),
	}

	// Send the message
	_, _, err := s.client.PostMessage(
		s.config.Channel,
		slack.MsgOptionText("Remediation action executed", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)

	if err != nil {
		logger.Error(err, "Failed to send Slack remediation notification")
		return fmt.Errorf("failed to send Slack remediation notification: %w", err)
	}

	logger.Info("Successfully sent Slack remediation notification", "action", result.Action, "resource", result.Resource)
	return nil
}

// SendStartupNotification sends a notification when KubeGuardian starts
func (s *SlackNotifier) SendStartupNotification(ctx context.Context, version string) error {
	if s == nil || !s.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	attachment := slack.Attachment{
		Color: "good",
		Title: "🚀 KubeGuardian Started",
		Text:  fmt.Sprintf("KubeGuardian v%s is now monitoring your Kubernetes cluster", version),
		Fields: []slack.AttachmentField{
			{
				Title: "Version",
				Value: version,
				Short: true,
			},
			{
				Title: "Status",
				Value: "🟢 Active",
				Short: true,
			},
		},
		Footer:     "KubeGuardian",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
		Ts:         json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	_, _, err := s.client.PostMessage(
		s.config.Channel,
		slack.MsgOptionText("KubeGuardian started", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)

	if err != nil {
		logger.Error(err, "Failed to send Slack startup notification")
		return fmt.Errorf("failed to send Slack startup notification: %w", err)
	}

	logger.Info("Successfully sent Slack startup notification")
	return nil
}

// getColorBySeverity returns a color based on severity level
func (s *SlackNotifier) getColorBySeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "danger"
	case "high":
		return "warning"
	case "medium":
		return "#ff9900"
	case "low":
		return "good"
	default:
		return "#cccccc"
	}
}

// TestConnection tests the Slack connection
func (s *SlackNotifier) TestConnection(ctx context.Context) error {
	if s == nil || !s.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	// Test by getting auth info
	_, err := s.client.AuthTest()
	if err != nil {
		logger.Error(err, "Slack connection test failed")
		return fmt.Errorf("Slack connection test failed: %w", err)
	}

	logger.Info("Slack connection test successful")
	return nil
}
