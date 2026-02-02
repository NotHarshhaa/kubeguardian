package config

import (
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		wantWarn bool
	}{
		{
			name: "valid config",
			config: &Config{
				Controller: ControllerConfig{
					MetricsAddr: ":8080",
					ProbeAddr:   ":8081",
					MaxConcurrentReconciles: 1,
					SyncPeriod: 30 * time.Second,
				},
				Detection: DetectionConfig{
					EvaluationInterval: 30 * time.Second,
					CPUThresholdPercent: 80.0,
					MemoryThresholdPercent: 85.0,
					CrashLoopThreshold:        3,
					FailedDeploymentThreshold: 5,
					OOMKillThreshold:          2,
				},
				Remediation: RemediationConfig{
					Enabled:     true,
					MaxRetries:  3,
					DryRun:      false,
					RetryInterval: 30 * time.Second,
					CooldownSeconds: 300,
				},
			},
			wantErr:  false,
			wantWarn: false,
		},
		{
			name: "invalid evaluation interval",
			config: &Config{
				Controller: ControllerConfig{
					MetricsAddr: ":8080",
					ProbeAddr:   ":8081",
					MaxConcurrentReconciles: 1,
					SyncPeriod: 30 * time.Second,
				},
				Detection: DetectionConfig{
					EvaluationInterval: 100 * time.Millisecond, // Too short
					CPUThresholdPercent: 80.0,
					MemoryThresholdPercent: 85.0,
					CrashLoopThreshold:        3,
					FailedDeploymentThreshold: 5,
					OOMKillThreshold:          2,
				},
				Remediation: RemediationConfig{
					Enabled:     true,
					MaxRetries:  3,
					DryRun:      false,
					RetryInterval: 30 * time.Second,
					CooldownSeconds: 300,
				},
			},
			wantErr:  true,
			wantWarn: false,
		},
		{
			name: "invalid CPU threshold",
			config: &Config{
				Controller: ControllerConfig{
					MetricsAddr: ":8080",
					ProbeAddr:   ":8081",
					MaxConcurrentReconciles: 1,
					SyncPeriod: 30 * time.Second,
				},
				Detection: DetectionConfig{
					EvaluationInterval: 30 * time.Second,
					CPUThresholdPercent: 150.0, // Invalid (> 100)
					MemoryThresholdPercent: 85.0,
					CrashLoopThreshold:        3,
					FailedDeploymentThreshold: 5,
					OOMKillThreshold:          2,
				},
				Remediation: RemediationConfig{
					Enabled:     true,
					MaxRetries:  3,
					DryRun:      false,
					RetryInterval: 30 * time.Second,
					CooldownSeconds: 300,
				},
			},
			wantErr:  true,
			wantWarn: false,
		},
		{
			name: "negative max retries",
			config: &Config{
				Controller: ControllerConfig{
					MetricsAddr: ":8080",
					ProbeAddr:   ":8081",
					MaxConcurrentReconciles: 1,
					SyncPeriod: 30 * time.Second,
				},
				Detection: DetectionConfig{
					EvaluationInterval: 30 * time.Second,
					CPUThresholdPercent: 80.0,
					MemoryThresholdPercent: 85.0,
					CrashLoopThreshold:        3,
					FailedDeploymentThreshold: 5,
					OOMKillThreshold:          2,
				},
				Remediation: RemediationConfig{
					MaxRetries: -1, // Invalid
					DryRun:      false,
					RetryInterval: 30 * time.Second,
					CooldownSeconds: 300,
				},
			},
			wantErr:  true,
			wantWarn: false,
		},
		{
			name: "config with warnings",
			config: &Config{
				Controller: ControllerConfig{
					MetricsAddr: ":8080",
					ProbeAddr:   ":8081",
					MaxConcurrentReconciles: 1,
					SyncPeriod: 500 * time.Millisecond, // Warning threshold
				},
				Detection: DetectionConfig{
					EvaluationInterval: 30 * time.Second,
					CPUThresholdPercent: 80.0,
					MemoryThresholdPercent: 85.0,
					CrashLoopThreshold:        3,
					FailedDeploymentThreshold: 5,
					OOMKillThreshold:          2,
				},
				Notification: NotificationConfig{
					Slack: SlackConfig{
						Enabled: true,
						Token:   "test-token",
						Channel: "#invalid-channel-with-uppercase",
					},
				},
				Remediation: RemediationConfig{
					Enabled:     true,
					MaxRetries:  3,
					DryRun:      false,
					RetryInterval: 30 * time.Second,
					CooldownSeconds: 300,
				},
			},
			wantErr:  false,
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			
			if tt.wantErr && len(result.Errors) == 0 {
				t.Errorf("expected validation errors but got none")
			}
			if !tt.wantErr && len(result.Errors) > 0 {
				t.Errorf("unexpected validation errors: %v", result.Errors)
			}
			if tt.wantWarn && len(result.Warnings) == 0 {
				t.Errorf("expected validation warnings but got none")
			}
			if !tt.wantWarn && len(result.Warnings) > 0 {
				t.Errorf("unexpected validation warnings: %v", result.Warnings)
			}
		})
	}
}

func TestNamespaceValidation(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		valid     bool
	}{
		{"valid namespace", "default", true},
		{"valid namespace with hyphens", "my-namespace", true},
		{"valid namespace with numbers", "namespace-123", true},
		{"invalid namespace - uppercase", "InvalidNamespace", false},
		{"invalid namespace - too long", "this-is-a-very-long-namespace-name-that-exceeds-the-kubernetes-limit", false},
		{"invalid namespace - special chars", "namespace$", false},
		{"invalid namespace - starts with hyphen", "-namespace", false},
		{"invalid namespace - ends with hyphen", "namespace-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidNamespaceName(tt.namespace)
			if result != tt.valid {
				t.Errorf("isValidNamespaceName(%q) = %v, want %v", tt.namespace, result, tt.valid)
			}
		})
	}
}

func TestSlackChannelValidation(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		valid   bool
	}{
		{"valid channel", "#general", true},
		{"valid channel with underscores", "#my_channel", true},
		{"valid channel with numbers", "#channel-123", true},
		{"invalid channel - no hash", "general", false},
		{"invalid channel - uppercase", "#General", false},
		{"invalid channel - special chars", "#channel$", false},
		{"invalid channel - too short", "#", false},
		{"invalid channel - too long", "#this-is-a-very-long-channel-name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSlackChannel(tt.channel)
			if result != tt.valid {
				t.Errorf("isValidSlackChannel(%q) = %v, want %v", tt.channel, result, tt.valid)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// Test that default config is valid
	result := config.Validate()
	if len(result.Errors) > 0 {
		t.Errorf("default config should be valid but has errors: %v", result.Errors)
	}
	
	// Test default values
	if config.Detection.EvaluationInterval != 30*time.Second {
		t.Errorf("default evaluation interval = %v, want %v", config.Detection.EvaluationInterval, 30*time.Second)
	}
	if config.Detection.CPUThresholdPercent != 80.0 {
		t.Errorf("default CPU threshold = %v, want %v", config.Detection.CPUThresholdPercent, 80.0)
	}
	if config.Remediation.MaxRetries != 3 {
		t.Errorf("default max retries = %v, want %v", config.Remediation.MaxRetries, 3)
	}
}
