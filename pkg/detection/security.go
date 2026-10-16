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

// SecurityIssue represents a detected security policy violation
type SecurityIssue struct {
	ResourceKind string            `json:"resourceKind"`
	ResourceName string            `json:"resourceName"`
	Namespace    string            `json:"namespace"`
	RuleID       string            `json:"ruleID"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Severity     string            `json:"severity"`
	DetectedAt   time.Time         `json:"detectedAt"`
	Remediation  string            `json:"remediation"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// SecurityPolicy represents a single security check rule
type SecurityPolicy struct {
	ID          string `yaml:"id"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Severity    string `yaml:"severity"` // low, medium, high, critical
	Enabled     bool   `yaml:"enabled"`
	Remediation string `yaml:"remediation"`
}

// DefaultSecurityPolicies returns the default set of security policies
func DefaultSecurityPolicies() []SecurityPolicy {
	return []SecurityPolicy{
		{
			ID:          "privileged-container",
			Title:       "Privileged Container Detected",
			Description: "Container is running in privileged mode, which gives it full access to the host",
			Severity:    "critical",
			Enabled:     true,
			Remediation: "Remove securityContext.privileged=true from the container spec",
		},
		{
			ID:          "root-user",
			Title:       "Container Running as Root",
			Description: "Container is running as root user (runAsNonRoot not set or set to false)",
			Severity:    "high",
			Enabled:     true,
			Remediation: "Set securityContext.runAsNonRoot=true and securityContext.runAsUser to a non-zero value",
		},
		{
			ID:          "missing-security-context",
			Title:       "Missing Security Context",
			Description: "Container does not have a security context defined",
			Severity:    "medium",
			Enabled:     true,
			Remediation: "Add securityContext with runAsNonRoot, capabilities, and other restrictions",
		},
		{
			ID:          "host-network",
			Title:       "Host Network Enabled",
			Description: "Pod is using hostNetwork, giving it direct access to the host network namespace",
			Severity:    "high",
			Enabled:     true,
			Remediation: "Remove hostNetwork=true from the pod spec",
		},
		{
			ID:          "host-pid",
			Title:       "Host PID Enabled",
			Description: "Pod is using hostPID, giving it access to the host PID namespace",
			Severity:    "high",
			Enabled:     true,
			Remediation: "Remove hostPID=true from the pod spec",
		},
		{
			ID:          "host-ipc",
			Title:       "Host IPC Enabled",
			Description: "Pod is using hostIPC, giving it access to the host IPC namespace",
			Severity:    "high",
			Enabled:     true,
			Remediation: "Remove hostIPC=true from the pod spec",
		},
		{
			ID:          "allow-privilege-escalation",
			Title:       "Privilege Escalation Allowed",
			Description: "Container allows privilege escalation (allowPrivilegeEscalation not set to false)",
			Severity:    "high",
			Enabled:     true,
			Remediation: "Set securityContext.allowPrivilegeEscalation=false",
		},
		{
			ID:          "missing-readonly-rootfs",
			Title:       "Writable Root Filesystem",
			Description: "Container root filesystem is not read-only",
			Severity:    "medium",
			Enabled:     true,
			Remediation: "Set securityContext.readOnlyRootFilesystem=true",
		},
	}
}

// SecurityDetector scans pods and deployments for security policy violations
type SecurityDetector struct {
	client   kubernetes.Interface
	policies []SecurityPolicy
}

// NewSecurityDetector creates a new security detector
func NewSecurityDetector(client kubernetes.Interface) *SecurityDetector {
	return &SecurityDetector{
		client:   client,
		policies: DefaultSecurityPolicies(),
	}
}

// SetPolicies sets the security policies to enforce
func (d *SecurityDetector) SetPolicies(policies []SecurityPolicy) {
	d.policies = policies
}

// Detect scans all pods for security violations
func (d *SecurityDetector) Detect(ctx context.Context) ([]SecurityIssue, error) {
	logger := log.FromContext(ctx)
	var issues []SecurityIssue

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		podIssues := d.checkPodSecurity(pod)
		issues = append(issues, podIssues...)
	}

	if len(issues) > 0 {
		logger.Info("Security issues detected", "count", len(issues))
	}

	return issues, nil
}

// checkPodSecurity checks a single pod against all security policies
func (d *SecurityDetector) checkPodSecurity(pod corev1.Pod) []SecurityIssue {
	var issues []SecurityIssue

	// Check pod-level settings
	if pod.Spec.HostNetwork {
		issues = append(issues, SecurityIssue{
			ResourceKind: "Pod",
			ResourceName: pod.Name,
			Namespace:    pod.Namespace,
			RuleID:       "host-network",
			Title:        "Host Network Enabled",
			Description:  fmt.Sprintf("Pod %s is using hostNetwork", pod.Name),
			Severity:     "high",
			DetectedAt:   time.Now(),
			Remediation:  "Remove hostNetwork=true from the pod spec",
			Labels:       pod.Labels,
		})
	}

	if pod.Spec.HostPID {
		issues = append(issues, SecurityIssue{
			ResourceKind: "Pod",
			ResourceName: pod.Name,
			Namespace:    pod.Namespace,
			RuleID:       "host-pid",
			Title:        "Host PID Enabled",
			Description:  fmt.Sprintf("Pod %s is using hostPID", pod.Name),
			Severity:     "high",
			DetectedAt:   time.Now(),
			Remediation:  "Remove hostPID=true from the pod spec",
			Labels:       pod.Labels,
		})
	}

	if pod.Spec.HostIPC {
		issues = append(issues, SecurityIssue{
			ResourceKind: "Pod",
			ResourceName: pod.Name,
			Namespace:    pod.Namespace,
			RuleID:       "host-ipc",
			Title:        "Host IPC Enabled",
			Description:  fmt.Sprintf("Pod %s is using hostIPC", pod.Name),
			Severity:     "high",
			DetectedAt:   time.Now(),
			Remediation:  "Remove hostIPC=true from the pod spec",
			Labels:       pod.Labels,
		})
	}

	// Check container-level settings
	for _, container := range pod.Spec.Containers {
		containerIssues := d.checkContainerSecurity(pod, container)
		issues = append(issues, containerIssues...)
	}

	// Check init containers
	for _, container := range pod.Spec.InitContainers {
		containerIssues := d.checkContainerSecurity(pod, container)
		issues = append(issues, containerIssues...)
	}

	return issues
}

// checkContainerSecurity checks a single container against security policies
func (d *SecurityDetector) checkContainerSecurity(pod corev1.Pod, container corev1.Container) []SecurityIssue {
	var issues []SecurityIssue

	sc := container.SecurityContext
	if sc == nil {
		// Missing security context
		if policyEnabled(d.policies, "missing-security-context") {
			issues = append(issues, SecurityIssue{
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				RuleID:       "missing-security-context",
				Title:        "Missing Security Context",
				Description:  fmt.Sprintf("Container %s in pod %s has no security context", container.Name, pod.Name),
				Severity:     "medium",
				DetectedAt:   time.Now(),
				Remediation:  "Add securityContext with runAsNonRoot, capabilities, and other restrictions",
				Labels:       pod.Labels,
			})
		}
		return issues // Can't check individual fields without a context
	}

	// Check privileged
	if sc.Privileged != nil && *sc.Privileged {
		if policyEnabled(d.policies, "privileged-container") {
			issues = append(issues, SecurityIssue{
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				RuleID:       "privileged-container",
				Title:        "Privileged Container Detected",
				Description:  fmt.Sprintf("Container %s in pod %s is running in privileged mode", container.Name, pod.Name),
				Severity:     "critical",
				DetectedAt:   time.Now(),
				Remediation:  "Remove securityContext.privileged=true from the container spec",
				Labels:       pod.Labels,
			})
		}
	}

	// Check runAsNonRoot
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		if policyEnabled(d.policies, "root-user") {
			issues = append(issues, SecurityIssue{
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				RuleID:       "root-user",
				Title:        "Container Running as Root",
				Description:  fmt.Sprintf("Container %s in pod %s may run as root user", container.Name, pod.Name),
				Severity:     "high",
				DetectedAt:   time.Now(),
				Remediation:  "Set securityContext.runAsNonRoot=true and securityContext.runAsUser to a non-zero value",
				Labels:       pod.Labels,
			})
		}
	}

	// Check allowPrivilegeEscalation
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		if policyEnabled(d.policies, "allow-privilege-escalation") {
			issues = append(issues, SecurityIssue{
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				RuleID:       "allow-privilege-escalation",
				Title:        "Privilege Escalation Allowed",
				Description:  fmt.Sprintf("Container %s in pod %s allows privilege escalation", container.Name, pod.Name),
				Severity:     "high",
				DetectedAt:   time.Now(),
				Remediation:  "Set securityContext.allowPrivilegeEscalation=false",
				Labels:       pod.Labels,
			})
		}
	}

	// Check readOnlyRootFilesystem
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		if policyEnabled(d.policies, "missing-readonly-rootfs") {
			issues = append(issues, SecurityIssue{
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Namespace:    pod.Namespace,
				RuleID:       "missing-readonly-rootfs",
				Title:        "Writable Root Filesystem",
				Description:  fmt.Sprintf("Container %s in pod %s has a writable root filesystem", container.Name, pod.Name),
				Severity:     "medium",
				DetectedAt:   time.Now(),
				Remediation:  "Set securityContext.readOnlyRootFilesystem=true",
				Labels:       pod.Labels,
			})
		}
	}

	return issues
}

// policyEnabled checks if a security policy is enabled
func policyEnabled(policies []SecurityPolicy, id string) bool {
	for _, p := range policies {
		if p.ID == id {
			return p.Enabled
		}
	}
	return false
}

// Ensure time import is used
var _ = time.Time{}
var _ = metav1.NewTime(time.Now())
