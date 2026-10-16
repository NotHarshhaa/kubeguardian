package remediation

import (
	"fmt"
	"sync"
	"time"
)

// MaintenanceWindow defines a time window when remediation is allowed or blocked
type MaintenanceWindow struct {
	ID          string     `json:"id" yaml:"id"`
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	StartTime   time.Time  `json:"startTime" yaml:"startTime"`
	EndTime     time.Time  `json:"endTime" yaml:"endTime"`
	Recurring   bool       `json:"recurring" yaml:"recurring"`
	DaysOfWeek  []time.Weekday `json:"daysOfWeek,omitempty" yaml:"daysOfWeek,omitempty"`
	TimeZone    string     `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	Action      string     `json:"action" yaml:"action"` // "allow" or "block"
	Namespaces  []string   `json:"namespaces,omitempty" yaml:"namespaces,omitempty"` // empty means all
}

// MaintenanceManager manages maintenance windows for remediation gating
type MaintenanceManager struct {
	mu      sync.RWMutex
	windows []MaintenanceWindow
}

// NewMaintenanceManager creates a new maintenance manager
func NewMaintenanceManager() *MaintenanceManager {
	return &MaintenanceManager{
		windows: make([]MaintenanceWindow, 0),
	}
}

// AddWindow adds a maintenance window
func (m *MaintenanceManager) AddWindow(window MaintenanceWindow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.windows = append(m.windows, window)
}

// RemoveWindow removes a maintenance window by ID
func (m *MaintenanceManager) RemoveWindow(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, w := range m.windows {
		if w.ID == id {
			m.windows = append(m.windows[:i], m.windows[i+1:]...)
			break
		}
	}
}

// IsRemediationAllowed checks if remediation is allowed at the given time
func (m *MaintenanceManager) IsRemediationAllowed(namespace string, now time.Time) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blockedBy := ""
	allowedBy := ""

	for _, window := range m.windows {
		if !m.windowMatchesNamespace(window, namespace) {
			continue
		}

		if !m.windowActive(window, now) {
			continue
		}

		if window.Action == "block" {
			blockedBy = window.Name
		} else if window.Action == "allow" {
			allowedBy = window.Name
		}
	}

	// If any block window is active, remediation is blocked
	if blockedBy != "" {
		return false, fmt.Sprintf("Remediation blocked by maintenance window: %s", blockedBy)
	}

	// If there are allow windows and none are active, remediation is blocked
	hasAllowWindows := false
	for _, window := range m.windows {
		if window.Action == "allow" && m.windowMatchesNamespace(window, namespace) {
			hasAllowWindows = true
			break
		}
	}

	if hasAllowWindows && allowedBy == "" {
		return false, "Remediation outside allowed maintenance window"
	}

	return true, ""
}

// windowMatchesNamespace checks if the window applies to the namespace
func (m *MaintenanceManager) windowMatchesNamespace(window MaintenanceWindow, namespace string) bool {
	if len(window.Namespaces) == 0 {
		return true
	}
	for _, ns := range window.Namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

// windowActive checks if the maintenance window is currently active
func (m *MaintenanceManager) windowActive(window MaintenanceWindow, now time.Time) bool {
	if window.Recurring {
		// Check day of week
		if len(window.DaysOfWeek) > 0 {
			dayMatch := false
			for _, d := range window.DaysOfWeek {
				if now.Weekday() == d {
					dayMatch = true
					break
				}
			}
			if !dayMatch {
				return false
			}
		}

		// Check time of day (compare just the time portion)
		start := time.Date(now.Year(), now.Month(), now.Day(),
			window.StartTime.Hour(), window.StartTime.Minute(), window.StartTime.Second(), 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(),
			window.EndTime.Hour(), window.EndTime.Minute(), window.EndTime.Second(), 0, now.Location())

		// Handle overnight windows
		if end.Before(start) {
			end = end.Add(24 * time.Hour)
		}

		return !now.Before(start) && !now.After(end)
	}

	return !now.Before(window.StartTime) && !now.After(window.EndTime)
}

// GetWindows returns all maintenance windows
func (m *MaintenanceManager) GetWindows() []MaintenanceWindow {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MaintenanceWindow, len(m.windows))
	copy(result, m.windows)
	return result
}

// GenerateWindowID generates a unique ID for a maintenance window
func GenerateWindowID(name string) string {
	return fmt.Sprintf("mw-%s-%d", name, time.Now().UnixNano())
}
