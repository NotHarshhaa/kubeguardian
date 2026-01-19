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
	MetricsAddr          string        `yaml:"metricsAddr"`
	ProbeAddr            string        `yaml:"probeAddr"`
	LeaderElection       bool          `yaml:"leaderElection"`
	SyncPeriod           time.Duration `yaml:"syncPeriod"`
	MaxConcurrentReconciles int        `yaml:"maxConcurrentReconciles"`
}

// DetectionConfig contains detection engine settings
type DetectionConfig struct {
	RulesFile            string        `yaml:"rulesFile"`
	EvaluationInterval   time.Duration `yaml:"evaluationInterval"`
	CrashLoopThreshold   int           `yaml:"crashLoopThreshold"`
	FailedDeploymentThreshold int      `yaml:"failedDeploymentThreshold"`
	CPUThresholdPercent   float64      `yaml:"cpuThresholdPercent"`
}

// RemediationConfig contains remediation engine settings
type RemediationConfig struct {
	Enabled              bool          `yaml:"enabled"`
	MaxRetries           int           `yaml:"maxRetries"`
	RetryInterval        time.Duration `yaml:"retryInterval"`
	DryRun               bool          `yaml:"dryRun"`
	AutoRollbackEnabled  bool          `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled     bool          `yaml:"autoScaleEnabled"`
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
			MetricsAddr:            ":8080",
			ProbeAddr:              ":8081",
			LeaderElection:         true,
			SyncPeriod:             30 * time.Second,
			MaxConcurrentReconciles: 1,
		},
		Detection: DetectionConfig{
			RulesFile:                "/etc/kubeguardian/rules.yaml",
			EvaluationInterval:       30 * time.Second,
			CrashLoopThreshold:       3,
			FailedDeploymentThreshold: 5,
			CPUThresholdPercent:      80.0,
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
