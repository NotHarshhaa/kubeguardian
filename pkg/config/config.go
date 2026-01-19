package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

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
	Namespaces                map[string]NamespaceConfig `yaml:"namespaces"`
}

// NamespaceConfig contains namespace-specific detection and remediation settings
type NamespaceConfig struct {
	CrashLoop   CrashLoopConfig            `yaml:"crashloop"`
	Deployment  DeploymentConfig           `yaml:"deployment"`
	CPU         CPUConfig                  `yaml:"cpu"`
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

// NamespaceRemediationConfig contains namespace-specific remediation settings
type NamespaceRemediationConfig struct {
	Enabled             bool          `yaml:"enabled"`
	AutoRollbackEnabled bool          `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled    bool          `yaml:"autoScaleEnabled"`
	MaxRetries          int           `yaml:"maxRetries"`
	RetryInterval       time.Duration `yaml:"retryInterval"`
}

// RemediationConfig contains remediation engine settings
type RemediationConfig struct {
	Enabled             bool          `yaml:"enabled"`
	MaxRetries          int           `yaml:"maxRetries"`
	RetryInterval       time.Duration `yaml:"retryInterval"`
	DryRun              bool          `yaml:"dryRun"`
	AutoRollbackEnabled bool          `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled    bool          `yaml:"autoScaleEnabled"`
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
					Remediation: NamespaceRemediationConfig{
						Enabled:             true,
						AutoRollbackEnabled: true,
						AutoScaleEnabled:    true,
						MaxRetries:          3,
						RetryInterval:       10 * time.Second,
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

	return config, nil
}
