package detection

import (
	"fmt"
	"sync"
	"time"
)

// SilenceRule defines a rule for suppressing alerts
type SilenceRule struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	RuleName    string            `json:"ruleName" yaml:"ruleName"`       // empty means all rules
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"` // empty means all namespaces
	Resource    string            `json:"resource,omitempty" yaml:"resource,omitempty"`   // empty means all resources
	Severity    string            `json:"severity,omitempty" yaml:"severity,omitempty"`   // empty means all severities
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`       // label matchers
	StartTime   time.Time         `json:"startTime" yaml:"startTime"`
	EndTime     time.Time         `json:"endTime" yaml:"endTime"`
	Reason      string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	CreatedBy   string            `json:"createdBy,omitempty" yaml:"createdBy,omitempty"`
}

// SilenceManager manages alert silencing rules
type SilenceManager struct {
	mu    sync.RWMutex
	rules []SilenceRule
}

// NewSilenceManager creates a new silence manager
func NewSilenceManager() *SilenceManager {
	return &SilenceManager{
		rules: make([]SilenceRule, 0),
	}
}

// AddRule adds a silence rule
func (s *SilenceManager) AddRule(rule SilenceRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
}

// RemoveRule removes a silence rule by ID
func (s *SilenceManager) RemoveRule(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.rules {
		if r.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			break
		}
	}
}

// IsSilenced checks if an alert should be silenced at the given time
func (s *SilenceManager) IsSilenced(ruleName, namespace, resource, severity string, labels map[string]string, now time.Time) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rule := range s.rules {
		if !s.ruleMatches(rule, ruleName, namespace, resource, severity, labels) {
			continue
		}

		if !s.ruleActive(rule, now) {
			continue
		}

		return true, rule.Reason
	}

	return false, ""
}

// ruleMatches checks if the silence rule matches the alert
func (s *SilenceManager) ruleMatches(rule SilenceRule, ruleName, namespace, resource, severity string, labels map[string]string) bool {
	if rule.RuleName != "" && rule.RuleName != ruleName {
		return false
	}
	if rule.Namespace != "" && rule.Namespace != namespace {
		return false
	}
	if rule.Resource != "" && rule.Resource != resource {
		return false
	}
	if rule.Severity != "" && rule.Severity != severity {
		return false
	}

	// Check label matchers
	for k, v := range rule.Labels {
		if labels[k] != v {
			return false
		}
	}

	return true
}

// ruleActive checks if the silence rule is currently active
func (s *SilenceManager) ruleActive(rule SilenceRule, now time.Time) bool {
	return !now.Before(rule.StartTime) && !now.After(rule.EndTime)
}

// GetRules returns all silence rules
func (s *SilenceManager) GetRules() []SilenceRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SilenceRule, len(s.rules))
	copy(result, s.rules)
	return result
}

// GenerateSilenceID generates a unique ID for a silence rule
func GenerateSilenceID(name string) string {
	return fmt.Sprintf("sil-%s-%d", name, time.Now().UnixNano())
}
