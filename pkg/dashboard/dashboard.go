package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
)

// Dashboard serves a simple web UI for KubeGuardian
type Dashboard struct {
	controller *controller.Controller
}

// NewDashboard creates a new dashboard handler
func NewDashboard(ctrl *controller.Controller) *Dashboard {
	return &Dashboard{
		controller: ctrl,
	}
}

// RegisterRoutes registers dashboard routes on the given mux
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/dashboard/summary", d.handleSummary)
	mux.HandleFunc("/api/v1/dashboard/history", d.handleHistory)
	mux.HandleFunc("/api/v1/dashboard/stats", d.handleStats)
	mux.HandleFunc("/api/v1/dashboard/recommendations", d.handleRecommendations)
	mux.HandleFunc("/api/v1/dashboard/silence", d.handleSilence)
	mux.HandleFunc("/api/v1/dashboard/maintenance", d.handleMaintenance)
	mux.HandleFunc("/", d.handleUI)
}

// handleSummary returns a summary of current state
func (d *Dashboard) handleSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "running",
		"version":   "v2.0.0",
		"features":  []string{"detection", "remediation", "notifications", "security", "events", "prometheus"},
	})
}

// handleHistory returns remediation history
func (d *Dashboard) handleHistory(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	action := r.URL.Query().Get("action")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := parseInt(limitStr); err == nil {
			limit = l
		}
	}

	history := d.controller.GetRemediationHistory(namespace, action, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// handleStats returns remediation statistics
func (d *Dashboard) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := d.controller.GetRemediationStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleRecommendations returns resource recommendations
func (d *Dashboard) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	recs := d.controller.GetResourceRecommendations()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recs,
		"count":           len(recs),
	})
}

// handleSilence handles silence rule operations
func (d *Dashboard) handleSilence(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules := d.controller.GetSilenceRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleMaintenance handles maintenance window operations
func (d *Dashboard) handleMaintenance(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		windows := d.controller.GetMaintenanceWindows()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"windows": windows,
			"count":   len(windows),
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleUI serves the dashboard HTML UI
func (d *Dashboard) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
