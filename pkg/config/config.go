package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// ValidationResult represents a configuration validation result
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Validate validates the configuration
func (c *Config) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate controller config
	c.validateController(result)

	// Validate detection config
	c.validateDetection(result)

	// Validate remediation config
	c.validateRemediation(result)

	// Validate notification config
	c.validateNotification(result)

	// Validate namespace configs
	c.validateNamespaces(result)

	result.Valid = len(result.Errors) == 0
	return result
}

func (c *Config) validateController(result *ValidationResult) {
	if c.Controller.MetricsAddr == "" {
		result.Errors = append(result.Errors, "metrics address cannot be empty")
	}

	if c.Controller.ProbeAddr == "" {
		result.Errors = append(result.Errors, "probe address cannot be empty")
	}

	if c.Controller.MaxConcurrentReconciles < 1 {
		result.Errors = append(result.Errors, "max concurrent reconciles must be at least 1")
	}

	if c.Controller.SyncPeriod < time.Second {
		result.Warnings = append(result.Warnings, "sync period less than 1 second may cause high CPU usage")
	}
}

func (c *Config) validateDetection(result *ValidationResult) {
	if c.Detection.EvaluationInterval < time.Second {
		result.Errors = append(result.Errors, "evaluation interval must be at least 1 second")
	}

	if c.Detection.CrashLoopThreshold < 1 {
		result.Errors = append(result.Errors, "crash loop threshold must be at least 1")
	}

	if c.Detection.FailedDeploymentThreshold < 1 {
		result.Errors = append(result.Errors, "failed deployment threshold must be at least 1")
	}

	if c.Detection.CPUThresholdPercent < 0 || c.Detection.CPUThresholdPercent > 100 {
		result.Errors = append(result.Errors, "CPU threshold percent must be between 0 and 100")
	}

	if c.Detection.MemoryThresholdPercent < 0 || c.Detection.MemoryThresholdPercent > 100 {
		result.Errors = append(result.Errors, "memory threshold percent must be between 0 and 100")
	}

	if c.Detection.OOMKillThreshold < 1 {
		result.Errors = append(result.Errors, "OOM kill threshold must be at least 1")
	}
}

func (c *Config) validateRemediation(result *ValidationResult) {
	if c.Remediation.MaxRetries < 0 {
		result.Errors = append(result.Errors, "max retries cannot be negative")
	}

	if c.Remediation.MaxRetries > 10 {
		result.Errors = append(result.Errors, "max retries too high (security: potential resource exhaustion)")
	}

	if c.Remediation.RetryInterval < time.Second {
		result.Warnings = append(result.Warnings, "retry interval less than 1 second may cause excessive retries")
	}

	if c.Remediation.RetryInterval < 100*time.Millisecond {
		result.Errors = append(result.Errors, "retry interval too short (security: thundering herd risk)")
	}

	if c.Remediation.CooldownSeconds < 0 {
		result.Errors = append(result.Errors, "cooldown seconds cannot be negative")
	}

	if c.Remediation.CooldownSeconds == 0 {
		result.Errors = append(result.Errors, "cooldown disabled (security: potential for abuse)")
	}

	if c.Remediation.CooldownSeconds > 3600 {
		result.Warnings = append(result.Warnings, "cooldown period greater than 1 hour may be too long")
	}
}

func (c *Config) validateNotification(result *ValidationResult) {
	if c.Notification.Slack.Enabled {
		if c.Notification.Slack.Token == "" {
			result.Errors = append(result.Errors, "slack token is required when slack notifications are enabled")
		}

		if c.Notification.Slack.Channel == "" {
			result.Errors = append(result.Errors, "slack channel is required when slack notifications are enabled")
		}

		if c.Notification.Slack.Username == "" {
			result.Warnings = append(result.Warnings, "slack username is not set, using default")
		}

		// Validate channel format
		if c.Notification.Slack.Channel != "" {
			// Simple validation for Slack channel format
			if !isValidSlackChannel(c.Notification.Slack.Channel) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("slack channel name may be invalid: %s", c.Notification.Slack.Channel))
			}
		}
	}
}

func (c *Config) validateNamespaces(result *ValidationResult) {
	for namespace, nsConfig := range c.Detection.Namespaces {
		// Validate namespace name
		if !isValidNamespaceName(namespace) {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid namespace name '%s'", namespace))
			continue
		}

		// Validate crash loop config
		if nsConfig.CrashLoop.RestartLimit < 1 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': crash loop restart limit must be at least 1", namespace))
		}

		if nsConfig.CrashLoop.CheckDuration < time.Second {
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace '%s': crash loop check duration less than 1 second may cause high CPU usage", namespace))
		}

		// Validate deployment config
		if nsConfig.Deployment.FailureThreshold < 1 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': deployment failure threshold must be at least 1", namespace))
		}

		if nsConfig.Deployment.CheckDuration < time.Second {
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace '%s': deployment check duration less than 1 second may cause high CPU usage", namespace))
		}

		// Validate CPU config
		if nsConfig.CPU.ThresholdPercent < 0 || nsConfig.CPU.ThresholdPercent > 100 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': CPU threshold percent must be between 0 and 100", namespace))
		}

		if nsConfig.CPU.CheckDuration < time.Second {
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace '%s': CPU check duration less than 1 second may cause high CPU usage", namespace))
		}

		// Validate memory config
		if nsConfig.Memory.ThresholdPercent < 0 || nsConfig.Memory.ThresholdPercent > 100 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': memory threshold percent must be between 0 and 100", namespace))
		}

		if nsConfig.Memory.OOMKillThreshold < 1 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': OOM kill threshold must be at least 1", namespace))
		}

		if nsConfig.Memory.CheckDuration < time.Second {
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace '%s': memory check duration less than 1 second may cause high CPU usage", namespace))
		}

		// Validate remediation config
		if nsConfig.Remediation.MaxRetries < 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': max retries cannot be negative", namespace))
		}

		if nsConfig.Remediation.RetryInterval < time.Second {
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace '%s': retry interval less than 1 second may cause excessive retries", namespace))
		}

		if nsConfig.Remediation.CooldownSeconds < 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("namespace '%s': cooldown seconds cannot be negative", namespace))
		}
	}
}

// isValidNamespaceName validates Kubernetes namespace name
func isValidNamespaceName(name string) bool {
	// Kubernetes namespace name regex
	namespaceRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	return namespaceRegex.MatchString(name) && len(name) <= 63
}

// isValidSlackChannel validates Slack channel name
func isValidSlackChannel(channel string) bool {
	// Slack channel names start with # and contain lowercase letters, numbers, hyphens, and underscores
	if len(channel) < 2 || channel[0] != '#' {
		return false
	}
	channelRegex := regexp.MustCompile(`^#[a-z0-9_-]+$`)
	return channelRegex.MatchString(channel) && len(channel) <= 22
}

// Config represents the main configuration for KubeGuardian
type Config struct {
	Controller   ControllerConfig   `yaml:"controller"`
	Detection    DetectionConfig    `yaml:"detection"`
	Remediation  RemediationConfig  `yaml:"remediation"`
	Notification NotificationConfig `yaml:"notification"`
}

// ControllerConfig contains controller-specific settings
type ControllerConfig struct {
	MetricsAddr             string        `yaml:"metricsAddr"`
	ProbeAddr               string        `yaml:"probeAddr"`
	LeaderElection          bool          `yaml:"leaderElection"`
	SyncPeriod              time.Duration `yaml:"syncPeriod"`
	MaxConcurrentReconciles int           `yaml:"maxConcurrentReconciles"`
}

// DetectionConfig contains detection engine settings
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

// NamespaceConfig contains namespace-specific detection and remediation settings
type NamespaceConfig struct {
	CrashLoop   CrashLoopConfig            `yaml:"crashloop"`
	Deployment  DeploymentConfig           `yaml:"deployment"`
	CPU         CPUConfig                  `yaml:"cpu"`
	Memory      MemoryConfig               `yaml:"memory"`
	Remediation NamespaceRemediationConfig `yaml:"remediation"`
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

// NamespaceRemediationConfig contains namespace-specific remediation settings
type NamespaceRemediationConfig struct {
	Enabled             bool          `yaml:"enabled"`
	AutoRollbackEnabled bool          `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled    bool          `yaml:"autoScaleEnabled"`
	MaxRetries          int           `yaml:"maxRetries"`
	RetryInterval       time.Duration `yaml:"retryInterval"`
	CooldownSeconds     int           `yaml:"cooldownSeconds"`
}

// RemediationConfig contains remediation engine settings
type RemediationConfig struct {
	Enabled             bool                                  `yaml:"enabled"`
	MaxRetries          int                                   `yaml:"maxRetries"`
	RetryInterval       time.Duration                         `yaml:"retryInterval"`
	DryRun              bool                                  `yaml:"dryRun"`
	AutoRollbackEnabled bool                                  `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled    bool                                  `yaml:"autoScaleEnabled"`
	CooldownSeconds     int                                   `yaml:"cooldownSeconds"`
	Namespaces          map[string]NamespaceRemediationConfig `yaml:"namespaces"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Slack SlackConfig `yaml:"slack"`
}

// SlackConfig contains Slack-specific settings
type SlackConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Token     string `yaml:"token"`
	Channel   string `yaml:"channel"`
	Username  string `yaml:"username"`
	IconEmoji string `yaml:"iconEmoji"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Controller: ControllerConfig{
			MetricsAddr:             ":8080",
			ProbeAddr:               ":8081",
			LeaderElection:          true,
			SyncPeriod:              30 * time.Second,
			MaxConcurrentReconciles: 1,
		},
		Detection: DetectionConfig{
			RulesFile:                 "/etc/kubeguardian/rules.yaml",
			EvaluationInterval:        30 * time.Second,
			CrashLoopThreshold:        3,
			FailedDeploymentThreshold: 5,
			CPUThresholdPercent:       80.0,
			MemoryThresholdPercent:    85.0,
			OOMKillThreshold:          2,
			Namespaces: map[string]NamespaceConfig{
				"default": {
					CrashLoop: CrashLoopConfig{
						RestartLimit:  3,
						CheckDuration: 5 * time.Minute,
						Enabled:       true,
					},
					Deployment: DeploymentConfig{
						FailureThreshold: 5,
						CheckDuration:    10 * time.Minute,
						Enabled:          true,
					},
					CPU: CPUConfig{
						ThresholdPercent: 80.0,
						CheckDuration:    5 * time.Minute,
						Enabled:          true,
					},
					Memory: MemoryConfig{
						ThresholdPercent: 85.0,
						CheckDuration:    5 * time.Minute,
						OOMKillThreshold: 2,
						Enabled:          true,
					},
					Remediation: NamespaceRemediationConfig{
						Enabled:             true,
						AutoRollbackEnabled: true,
						AutoScaleEnabled:    true,
						MaxRetries:          3,
						RetryInterval:       10 * time.Second,
						CooldownSeconds:     300, // 5 minutes default cooldown
					},
				},
			},
		},
		Remediation: RemediationConfig{
			Enabled:             true,
			MaxRetries:          3,
			RetryInterval:       10 * time.Second,
			DryRun:              false,
			AutoRollbackEnabled: true,
			AutoScaleEnabled:    true,
			CooldownSeconds:     300, // 5 minutes default cooldown
		},
		Notification: NotificationConfig{
			Slack: SlackConfig{
				Enabled:   false,
				Token:     "",
				Channel:   "#kubeguardian",
				Username:  "KubeGuardian",
				IconEmoji: ":robot_face:",
			},
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the configuration
	validationResult := config.Validate()
	if !validationResult.Valid {
		return nil, fmt.Errorf("configuration validation failed: %v", validationResult.Errors)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			// In a real implementation, you'd use proper logging
			fmt.Printf("Configuration warning: %s\n", warning)
		}
	}

	return config, nil
}
