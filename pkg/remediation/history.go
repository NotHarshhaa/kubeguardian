package remediation

import (
	"fmt"
	"sync"
	"time"
)

// HistoryEntry represents a single remediation action record in the history
type HistoryEntry struct {
	ID             string        `json:"id"`
	Action         string        `json:"action"`
	Resource       string        `json:"resource"`
	Namespace      string        `json:"namespace"`
	Kind           string        `json:"kind"`
	Success        bool          `json:"success"`
	Message        string        `json:"message"`
	ExecutedAt     time.Time     `json:"executedAt"`
	Duration       time.Duration `json:"duration"`
	DryRun         bool          `json:"dryRun"`
	RuleName       string        `json:"ruleName,omitempty"`
	Severity       string        `json:"severity,omitempty"`
	IssueID        string        `json:"issueID,omitempty"`
	CooldownUsed   bool          `json:"cooldownUsed"`
}

// HistoryStore stores and manages remediation history entries
type HistoryStore struct {
	mu      sync.RWMutex
	entries []HistoryEntry
	maxSize int
}

// NewHistoryStore creates a new history store with a maximum size
func NewHistoryStore(maxSize int) *HistoryStore {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &HistoryStore{
		entries: make([]HistoryEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a new entry to the history store
func (h *HistoryStore) Add(entry HistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, entry)

	// Trim to max size, keeping most recent entries
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
	}
}

// Get returns all history entries, optionally filtered by namespace and action
func (h *HistoryStore) Get(namespace, action string, limit int) []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var filtered []HistoryEntry
	for _, entry := range h.entries {
		if namespace != "" && entry.Namespace != namespace {
			continue
		}
		if action != "" && entry.Action != action {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Return most recent first
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered
}

// GetByID returns a specific history entry by ID
func (h *HistoryStore) GetByID(id string) (HistoryEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return HistoryEntry{}, false
}

// GetStats returns statistics about remediation history
func (h *HistoryStore) GetStats() HistoryStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := HistoryStats{
		Total:       len(h.entries),
		ByAction:    make(map[string]int),
		ByNamespace: make(map[string]int),
		BySuccess:   make(map[bool]int),
		BySeverity:  make(map[string]int),
	}

	for _, entry := range h.entries {
		stats.ByAction[entry.Action]++
		stats.ByNamespace[entry.Namespace]++
		stats.BySuccess[entry.Success]++
		stats.BySeverity[entry.Severity]++
		if entry.Success {
			stats.Successful++
		} else {
			stats.Failed++
		}
	}

	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.Total) * 100
	}

	return stats
}

// HistoryStats contains aggregated statistics about remediation history
type HistoryStats struct {
	Total         int                    `json:"total"`
	Successful    int                    `json:"successful"`
	Failed        int                    `json:"failed"`
	SuccessRate   float64                `json:"successRate"`
	ByAction      map[string]int         `json:"byAction"`
	ByNamespace   map[string]int         `json:"byNamespace"`
	BySuccess     map[bool]int           `json:"bySuccess"`
	BySeverity    map[string]int         `json:"bySeverity"`
}

// Clear removes all history entries
func (h *HistoryStore) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]HistoryEntry, 0, h.maxSize)
}

// GenerateID generates a unique ID for a history entry
func GenerateID(action, namespace, resource string) string {
	return fmt.Sprintf("%s:%s:%s:%d", action, namespace, resource, time.Now().UnixNano())
}
