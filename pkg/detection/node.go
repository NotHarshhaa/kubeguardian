package detection

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NodeIssue represents a detected node issue
type NodeIssue struct {
	NodeName   string            `json:"nodeName"`
	Condition  string            `json:"condition"`
	Status     string            `json:"status"`
	Reason     string            `json:"reason"`
	Message    string            `json:"message"`
	Severity   string            `json:"severity"`
	DetectedAt time.Time         `json:"detectedAt"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// NodeDetector detects node health issues
type NodeDetector struct {
	client kubernetes.Interface
}

// NewNodeDetector creates a new node detector
func NewNodeDetector(client kubernetes.Interface) *NodeDetector {
	return &NodeDetector{
		client: client,
	}
}

// DetectNodeIssues checks all nodes for health issues
func (d *NodeDetector) DetectNodeIssues(ctx context.Context) ([]NodeIssue, error) {
	logger := log.FromContext(ctx)
	var issues []NodeIssue

	nodes, err := d.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		nodeIssues := d.checkNodeConditions(node)
		issues = append(issues, nodeIssues...)
	}

	if len(issues) > 0 {
		logger.Info("Node issues detected", "count", len(issues))
	}

	return issues, nil
}

// checkNodeConditions checks a single node for issues
func (d *NodeDetector) checkNodeConditions(node corev1.Node) []NodeIssue {
	var issues []NodeIssue

	for _, condition := range node.Status.Conditions {
		// We only care about conditions that are abnormal
		isAbnormal := false
		severity := "info"

		switch condition.Type {
		case corev1.NodeReady:
			if condition.Status == corev1.ConditionFalse {
				isAbnormal = true
				severity = "critical"
			} else if condition.Status == corev1.ConditionUnknown {
				isAbnormal = true
				severity = "high"
			}
		case corev1.NodeDiskPressure:
			if condition.Status == corev1.ConditionTrue {
				isAbnormal = true
				severity = "high"
			}
		case corev1.NodeMemoryPressure:
			if condition.Status == corev1.ConditionTrue {
				isAbnormal = true
				severity = "high"
			}
		case corev1.NodePIDPressure:
			if condition.Status == corev1.ConditionTrue {
				isAbnormal = true
				severity = "medium"
			}
		case corev1.NodeNetworkUnavailable:
			if condition.Status == corev1.ConditionTrue {
				isAbnormal = true
				severity = "high"
			}
		default:
			continue
		}

		if isAbnormal {
			issue := NodeIssue{
				NodeName:   node.Name,
				Condition:  string(condition.Type),
				Status:     string(condition.Status),
				Reason:     condition.Reason,
				Message:    condition.Message,
				Severity:   severity,
				DetectedAt: condition.LastTransitionTime.Time,
				Labels:     node.Labels,
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// NodeDetectorConfig contains configuration for node detection
type NodeDetectorConfig struct {
	Enabled          bool          `yaml:"enabled"`
	CheckInterval    time.Duration `yaml:"checkInterval"`
	AutoCordon       bool          `yaml:"autoCordon"`       // Automatically cordon unhealthy nodes
	AutoUncord       bool          `yaml:"autoUncord"`       // Automatically uncordon healthy nodes
	CordonDuration   time.Duration `yaml:"cordonDuration"`   // How long to cordon a node
	UncordonHealthy  bool          `yaml:"uncordonHealthy"`  // Uncordon nodes when they become healthy
}

// DefaultNodeDetectorConfig returns default node detection config
func DefaultNodeDetectorConfig() NodeDetectorConfig {
	return NodeDetectorConfig{
		Enabled:        true,
		CheckInterval:  1 * time.Minute,
		AutoCordon:     false,
		AutoUncord:     false,
		CordonDuration: 15 * time.Minute,
		UncordonHealthy: false,
	}
}
