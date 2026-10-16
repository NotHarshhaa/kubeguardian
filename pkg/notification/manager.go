package notification

import (
	"context"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/NotHarshhaa/kubeguardian/pkg/detection"
	"github.com/NotHarshhaa/kubeguardian/pkg/remediation"
)

// Notifier defines the interface for sending notifications
type Notifier interface {
	// Name returns the notifier name
	Name() string

	// SendIssueNotification sends a notification about a detected issue
	SendIssueNotification(ctx context.Context, issue detection.Issue) error

	// SendRemediationNotification sends a notification about a remediation action
	SendRemediationNotification(ctx context.Context, issue detection.Issue, result remediation.Result) error

	// SendStartupNotification sends a notification when KubeGuardian starts
	SendStartupNotification(ctx context.Context, version string) error

	// TestConnection tests the notification connection
	TestConnection(ctx context.Context) error

	// IsEnabled returns true if the notifier is enabled
	IsEnabled() bool
}

// Manager manages multiple notification channels
type Manager struct {
	mu          sync.RWMutex
	notifiers   map[string]Notifier
	enabled     []string
}

// NewManager creates a new notification manager
func NewManager() *Manager {
	return &Manager{
		notifiers: make(map[string]Notifier),
	}
}

// Register registers a notifier with the manager
func (m *Manager) Register(notifier Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if notifier == nil {
		return
	}

	name := notifier.Name()
	m.notifiers[name] = notifier

	if notifier.IsEnabled() {
		m.enabled = append(m.enabled, name)
	}
}

// Get returns a notifier by name
func (m *Manager) Get(name string) (Notifier, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notifier, exists := m.notifiers[name]
	return notifier, exists
}

// Enabled returns the list of enabled notifier names
func (m *Manager) Enabled() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.enabled))
	copy(result, m.enabled)
	return result
}

// SendIssueNotification sends an issue notification through all enabled notifiers
func (m *Manager) SendIssueNotification(ctx context.Context, issue detection.Issue) error {
	m.mu.RLock()
	notifiers := make([]Notifier, 0, len(m.enabled))
	for _, name := range m.enabled {
		if notifier, exists := m.notifiers[name]; exists {
			notifiers = append(notifiers, notifier)
		}
	}
	m.mu.RUnlock()

	var firstErr error
	for _, notifier := range notifiers {
		if err := notifier.SendIssueNotification(ctx, issue); err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "Failed to send issue notification", "notifier", notifier.Name())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// SendRemediationNotification sends a remediation notification through all enabled notifiers
func (m *Manager) SendRemediationNotification(ctx context.Context, issue detection.Issue, result remediation.Result) error {
	m.mu.RLock()
	notifiers := make([]Notifier, 0, len(m.enabled))
	for _, name := range m.enabled {
		if notifier, exists := m.notifiers[name]; exists {
			notifiers = append(notifiers, notifier)
		}
	}
	m.mu.RUnlock()

	var firstErr error
	for _, notifier := range notifiers {
		if err := notifier.SendRemediationNotification(ctx, issue, result); err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "Failed to send remediation notification", "notifier", notifier.Name())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// SendStartupNotification sends a startup notification through all enabled notifiers
func (m *Manager) SendStartupNotification(ctx context.Context, version string) error {
	m.mu.RLock()
	notifiers := make([]Notifier, 0, len(m.enabled))
	for _, name := range m.enabled {
		if notifier, exists := m.notifiers[name]; exists {
			notifiers = append(notifiers, notifier)
		}
	}
	m.mu.RUnlock()

	var firstErr error
	for _, notifier := range notifiers {
		if err := notifier.SendStartupNotification(ctx, version); err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "Failed to send startup notification", "notifier", notifier.Name())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// TestConnection tests all notifier connections
func (m *Manager) TestConnection(ctx context.Context) error {
	m.mu.RLock()
	notifiers := make([]Notifier, 0, len(m.notifiers))
	for _, notifier := range m.notifiers {
		notifiers = append(notifiers, notifier)
	}
	m.mu.RUnlock()

	var firstErr error
	for _, notifier := range notifiers {
		if err := notifier.TestConnection(ctx); err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "Notifier connection test failed", "notifier", notifier.Name())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}
