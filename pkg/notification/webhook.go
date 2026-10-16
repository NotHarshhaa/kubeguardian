package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
)

// WebhookConfig contains webhook notification settings
type WebhookConfig struct {
	Enabled     bool              `yaml:"enabled"`
	URL         string            `yaml:"url"`
	Method      string            `yaml:"method"`
	Headers     map[string]string `yaml:"headers"`
	Timeout     time.Duration     `yaml:"timeout"`
	Secret      string            `yaml:"secret"`
	PayloadType string            `yaml:"payloadType"` // json or form
}

// WebhookNotifier sends notifications via HTTP webhook
type WebhookNotifier struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	method := config.Method
	if method == "" {
		method = "POST"
	}

	return &WebhookNotifier{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the notifier name
func (w *WebhookNotifier) Name() string {
	return "webhook"
}

// IsEnabled returns true if webhook notifications are enabled
func (w *WebhookNotifier) IsEnabled() bool {
	return w.config.Enabled
}

// SendIssueNotification sends an issue notification via webhook
func (w *WebhookNotifier) SendIssueNotification(ctx context.Context, issue detection.Issue) error {
	if !w.config.Enabled {
		return nil
	}

	payload := map[string]interface{}{
		"type":          "issue",
		"rule":          issue.RuleName,
		"description":   issue.Description,
		"severity":      issue.Severity,
		"resource":      issue.Kind,
		"name":          issue.Name,
		"namespace":     issue.Namespace,
		"detectedAt":    issue.DetectedAt,
		"actions":       issue.Actions,
		"labels":        issue.Labels,
		"source":        "kubeguardian",
	}

	return w.sendWebhook(ctx, payload, "issue notification")
}

// SendRemediationNotification sends a remediation notification via webhook
func (w *WebhookNotifier) SendRemediationNotification(ctx context.Context, issue detection.Issue, result remediation.Result) error {
	if !w.config.Enabled {
		return nil
	}

	status := "failed"
	if result.Success {
		status = "success"
	}

	payload := map[string]interface{}{
		"type":          "remediation",
		"action":        result.Action,
		"status":        status,
		"message":       result.Message,
		"resource":      issue.Kind,
		"name":          issue.Name,
		"namespace":     issue.Namespace,
		"issue":         issue.RuleName,
		"executedAt":    result.ExecutedAt,
		"duration":      result.Duration.String(),
		"source":        "kubeguardian",
	}

	return w.sendWebhook(ctx, payload, "remediation notification")
}

// SendStartupNotification sends a startup notification via webhook
func (w *WebhookNotifier) SendStartupNotification(ctx context.Context, version string) error {
	if !w.config.Enabled {
		return nil
	}

	payload := map[string]interface{}{
		"type":    "startup",
		"version": version,
		"status":  "active",
		"source":  "kubeguardian",
	}

	return w.sendWebhook(ctx, payload, "startup notification")
}

// TestConnection tests the webhook connection
func (w *WebhookNotifier) TestConnection(ctx context.Context) error {
	if !w.config.Enabled {
		return nil
	}

	payload := map[string]interface{}{
		"type":    "healthcheck",
		"source":  "kubeguardian",
		"message": "KubeGuardian webhook connection test",
	}

	if err := w.sendWebhook(ctx, payload, "connection test"); err != nil {
		return fmt.Errorf("webhook connection test failed: %w", err)
	}

	return nil
}

// sendWebhook sends a JSON payload to the configured webhook URL
func (w *WebhookNotifier) sendWebhook(ctx context.Context, payload interface{}, purpose string) error {
	if !w.config.Enabled {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, w.config.Method, w.config.URL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KubeGuardian")

	if w.config.Secret != "" {
		req.Header.Set("X-KubeGuardian-Secret", w.config.Secret)
	}

	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}
