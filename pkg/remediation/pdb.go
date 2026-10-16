package remediation

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PDBChecker checks Pod Disruption Budgets before disruptive remediation actions
type PDBChecker struct {
	client kubernetes.Interface
}

// NewPDBChecker creates a new PDB checker
func NewPDBChecker(client kubernetes.Interface) *PDBChecker {
	return &PDBChecker{
		client: client,
	}
}

// CanDisrupt checks if a pod can be safely disrupted based on PDBs
func (p *PDBChecker) CanDisrupt(ctx context.Context, namespace, podName string) (bool, string, error) {
	logger := log.FromContext(ctx)

	// Get the pod
	pod, err := p.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return false, "", fmt.Errorf("failed to get pod %s/%s: %w", namespace, podName, err)
	}

	// Get all PDBs in the namespace
	pdbs, err := p.client.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, "", fmt.Errorf("failed to list PDBs: %w", err)
	}

	// Check each PDB
	for _, pdb := range pdbs.Items {
		allowed, reason, err := p.checkPDB(ctx, pdb, pod, namespace)
		if err != nil {
			logger.Error(err, "Failed to check PDB", "pdb", pdb.Name, "namespace", namespace)
			continue
		}
		if !allowed {
			return false, reason, nil
		}
	}

	return true, "No PDB prevents disruption", nil
}

// checkPDB checks a single PDB against a pod
func (p *PDBChecker) checkPDB(ctx context.Context, pdb policyv1.PodDisruptionBudget, pod *corev1.Pod, namespace string) (bool, string, error) {
	// Check if the pod matches the PDB selector
	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		return false, "", fmt.Errorf("invalid PDB selector: %w", err)
	}

	if !selector.Matches(labels.Set(pod.Labels)) {
		return true, "", nil // PDB doesn't apply to this pod
	}

	// Get current healthy replicas
	pods, err := p.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to list pods for PDB check: %w", err)
	}

	healthyCount := int32(0)
	for _, p := range pods.Items {
		if isPodReady(&p) {
			healthyCount++
		}
	}

	// Check against minAvailable
	if pdb.Spec.MinAvailable != nil {
		minAvailable := pdb.Spec.MinAvailable.IntValue()
		if healthyCount-1 < int32(minAvailable) {
			return false, fmt.Sprintf("PDB %s would be violated: minAvailable=%d, would have %d healthy",
				pdb.Name, minAvailable, healthyCount-1), nil
		}
	}

	// Check against maxUnavailable
	if pdb.Spec.MaxUnavailable != nil {
		maxUnavailable := pdb.Spec.MaxUnavailable.IntValue()
		unavailableCount := int32(len(pods.Items)) - healthyCount + 1
		if unavailableCount > int32(maxUnavailable) {
			return false, fmt.Sprintf("PDB %s would be violated: maxUnavailable=%d, would have %d unavailable",
				pdb.Name, maxUnavailable, unavailableCount), nil
		}
	}

	return true, "", nil
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
