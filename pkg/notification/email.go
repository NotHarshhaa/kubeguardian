package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
)

// EmailConfig contains email notification settings
type EmailConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	From            string `yaml:"from"`
	To              string `yaml:"to"`
	UseTLS          bool   `yaml:"useTLS"`
	SkipTLSVerify   bool   `yaml:"skipTLSVerify"`
}

// EmailNotifier sends notifications via email
type EmailNotifier struct {
	config EmailConfig
	mu     sync.RWMutex
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		config: config,
	}
}

// Name returns the notifier name
func (e *EmailNotifier) Name() string {
	return "email"
}

// IsEnabled returns true if email notifications are enabled
func (e *EmailNotifier) IsEnabled() bool {
	return e.config.Enabled
}

// SendIssueNotification sends an issue notification via email
func (e *EmailNotifier) SendIssueNotification(ctx context.Context, issue detection.Issue) error {
	if !e.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	subject := fmt.Sprintf("[KubeGuardian] %s - %s", strings.ToUpper(issue.Severity), issue.RuleName)
	body := e.buildIssueEmailBody(issue)

	if err := e.sendEmail(subject, body); err != nil {
		logger.Error(err, "Failed to send email notification for issue", "rule", issue.RuleName)
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	logger.Info("Successfully sent email notification for issue", "rule", issue.RuleName, "resource", issue.Name)
	return nil
}

// SendRemediationNotification sends a remediation notification via email
func (e *EmailNotifier) SendRemediationNotification(ctx context.Context, issue detection.Issue, result remediation.Result) error {
	if !e.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	status := "FAILED"
	if result.Success {
		status = "SUCCESS"
	}

	subject := fmt.Sprintf("[KubeGuardian] Remediation %s - %s", status, result.Action)
	body := e.buildRemediationEmailBody(issue, result, status)

	if err := e.sendEmail(subject, body); err != nil {
		logger.Error(err, "Failed to send email notification for remediation", "action", result.Action)
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	logger.Info("Successfully sent email notification for remediation", "action", result.Action, "status", status)
	return nil
}

// SendStartupNotification sends a startup notification via email
func (e *EmailNotifier) SendStartupNotification(ctx context.Context, version string) error {
	if !e.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	subject := "[KubeGuardian] Started"
	body := fmt.Sprintf("KubeGuardian v%s has started and is monitoring your Kubernetes cluster.", version)

	if err := e.sendEmail(subject, body); err != nil {
		logger.Error(err, "Failed to send startup email notification")
		return fmt.Errorf("failed to send startup email notification: %w", err)
	}

	logger.Info("Successfully sent startup email notification")
	return nil
}

// TestConnection tests the email connection
func (e *EmailNotifier) TestConnection(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	logger := log.FromContext(ctx)

	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	var tlsConfig *tls.Config
	if e.config.UseTLS {
		tlsConfig = &tls.Config{
			ServerName:         e.config.Host,
			InsecureSkipVerify: e.config.SkipTLSVerify,
		}
	}

	var conn *smtp.Client
	var err error
	if tlsConfig != nil {
		rawConn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			logger.Error(err, "Email connection test failed")
			return fmt.Errorf("email connection test failed: %w", err)
		}
		conn, err = smtp.NewClient(rawConn, e.config.Host)
	} else {
		conn, err = smtp.Dial(addr)
	}
	if err != nil {
		logger.Error(err, "Email connection test failed")
		return fmt.Errorf("email connection test failed: %w", err)
	}
	defer conn.Close()

	if e.config.Username != "" && e.config.Password != "" {
		auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)
		if err := conn.Auth(auth); err != nil {
			logger.Error(err, "Email authentication failed")
			return fmt.Errorf("email authentication failed: %w", err)
		}
	}

	logger.Info("Email connection test successful")
	return nil
}

// sendEmail sends an email using the configured SMTP server
func (e *EmailNotifier) sendEmail(subject, body string) error {
	e.mu.RLock()
	config := e.config
	e.mu.RUnlock()

	if !config.Enabled {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var tlsConfig *tls.Config
	if config.UseTLS {
		tlsConfig = &tls.Config{
			ServerName:         config.Host,
			InsecureSkipVerify: config.SkipTLSVerify,
		}
	}

	var conn *smtp.Client
	var err error
	if tlsConfig != nil {
		rawConn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		conn, err = smtp.NewClient(rawConn, config.Host)
	} else {
		conn, err = smtp.Dial(addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := conn.Auth(auth); err != nil {
			return err
		}
	}

	if err := conn.Mail(config.From); err != nil {
		return err
	}

	if err := conn.Rcpt(config.To); err != nil {
		return err
	}

	w, err := conn.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		config.From, config.To, subject, body)

	_, err = w.Write([]byte(message))
	return err
}

// buildIssueEmailBody builds the email body for an issue notification
func (e *EmailNotifier) buildIssueEmailBody(issue detection.Issue) string {
	return fmt.Sprintf(`
KubeGuardian Issue Alert
========================

Rule: %s
Description: %s
Severity: %s
Resource: %s/%s
Namespace: %s
Detected At: %s
Actions: %s

This is an automated message from KubeGuardian.
`, issue.RuleName, issue.Description, issue.Severity, issue.Kind, issue.Name, issue.Namespace, issue.DetectedAt.Format("2006-01-02 15:04:05"), strings.Join(issue.Actions, ", "))
}

// buildRemediationEmailBody builds the email body for a remediation notification
func (e *EmailNotifier) buildRemediationEmailBody(issue detection.Issue, result remediation.Result, status string) string {
	return fmt.Sprintf(`
KubeGuardian Remediation Alert
==============================

Action: %s
Status: %s
Message: %s
Resource: %s/%s
Namespace: %s
Issue: %s
Executed At: %s
Duration: %s

This is an automated message from KubeGuardian.
`, result.Action, status, result.Message, issue.Kind, issue.Name, issue.Namespace, issue.RuleName, result.ExecutedAt.Format("2006-01-02 15:04:05"), result.Duration.String())
}
