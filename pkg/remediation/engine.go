package remediation

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Engine represents the remediation engine
type Engine struct {
	client kubernetes.Interface
	config RemediationConfig
}

// RemediationConfig contains remediation configuration
type RemediationConfig struct {
	Enabled             bool          `yaml:"enabled"`
	MaxRetries          int           `yaml:"maxRetries"`
	RetryInterval       time.Duration `yaml:"retryInterval"`
	DryRun              bool          `yaml:"dryRun"`
	AutoRollbackEnabled bool          `yaml:"autoRollbackEnabled"`
	AutoScaleEnabled    bool          `yaml:"autoScaleEnabled"`
}

// Action represents a remediation action
type Action struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Type        string      `yaml:"type"`
	Parameters  interface{} `yaml:"parameters"`
}

// Result represents the result of a remediation action
type Result struct {
	Action     string    `yaml:"action"`
	Success    bool      `yaml:"success"`
	Message    string    `yaml:"message"`
	Resource   string    `yaml:"resource"`
	Namespace  string    `yaml:"namespace"`
	ExecutedAt time.Time `yaml:"executedAt"`
	Duration   time.Duration `yaml:"duration"`
}

// NewEngine creates a new remediation engine
func NewEngine(client kubernetes.Interface, config RemediationConfig) *Engine {
	return &Engine{
		client: client,
		config: config,
	}
}

// ExecuteAction executes a remediation action
func (e *Engine) ExecuteAction(ctx context.Context, action string, resource interface{}, namespace string) (*Result, error) {
	logger := log.FromContext(ctx)
	
	if !e.config.Enabled {
		return &Result{
			Action:     action,
			Success:    false,
			Message:    "Remediation is disabled",
			ExecutedAt: time.Now(),
		}, nil
	}

	startTime := time.Now()
	
	switch action {
	case "restart-pod":
		return e.restartPod(ctx, resource, namespace)
	case "rollback-deployment":
		return e.rollbackDeployment(ctx, resource, namespace)
	case "scale-replicas":
		return e.scaleReplicas(ctx, resource, namespace)
	default:
		return &Result{
			Action:     action,
			Success:    false,
			Message:    fmt.Sprintf("Unknown action: %s", action),
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("unknown action: %s", action)
	}
}

// restartPod restarts a pod by deleting it
func (e *Engine) restartPod(ctx context.Context, resource interface{}, namespace string) (*Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()
	
	pod, ok := resource.(*corev1.Pod)
	if !ok {
		return &Result{
			Action:     "restart-pod",
			Success:    false,
			Message:    "Resource is not a Pod",
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("resource is not a Pod")
	}

	if e.config.DryRun {
		logger.Info("Dry run: would restart pod", "pod", pod.Name, "namespace", pod.Namespace)
		return &Result{
			Action:     "restart-pod",
			Success:    true,
			Message:    fmt.Sprintf("Dry run: would restart pod %s", pod.Name),
			Resource:   pod.Name,
			Namespace:  pod.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	err := e.client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return &Result{
			Action:     "restart-pod",
			Success:    false,
			Message:    fmt.Sprintf("Failed to restart pod: %v", err),
			Resource:   pod.Name,
			Namespace:  pod.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, err
	}

	logger.Info("Successfully restarted pod", "pod", pod.Name, "namespace", pod.Namespace)
	return &Result{
		Action:     "restart-pod",
		Success:    true,
		Message:    fmt.Sprintf("Successfully restarted pod %s", pod.Name),
		Resource:   pod.Name,
		Namespace:  pod.Namespace,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, nil
}

// rollbackDeployment rolls back a deployment to the previous revision
func (e *Engine) rollbackDeployment(ctx context.Context, resource interface{}, namespace string) (*Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()
	
	deployment, ok := resource.(*appsv1.Deployment)
	if !ok {
		return &Result{
			Action:     "rollback-deployment",
			Success:    false,
			Message:    "Resource is not a Deployment",
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("resource is not a Deployment")
	}

	if !e.config.AutoRollbackEnabled {
		return &Result{
			Action:     "rollback-deployment",
			Success:    false,
			Message:    "Auto rollback is disabled",
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	if e.config.DryRun {
		logger.Info("Dry run: would rollback deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
		return &Result{
			Action:     "rollback-deployment",
			Success:    true,
			Message:    fmt.Sprintf("Dry run: would rollback deployment %s", deployment.Name),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	// Get the deployment's rollout history to find the previous revision
	rolloutHistory, err := e.client.AppsV1().Deployments(deployment.Namespace).GetRolloutHistory(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil {
		return &Result{
			Action:     "rollback-deployment",
			Success:    false,
			Message:    fmt.Sprintf("Failed to get rollout history: %v", err),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, err
	}

	// Find the previous stable revision
	var previousRevision int64
	for _, revision := range rolloutHistory.Revisions {
		if revision.Revision < deployment.Status.Revision && revision.Revision > previousRevision {
			previousRevision = revision.Revision
		}
	}

	if previousRevision == 0 {
		return &Result{
			Action:     "rollback-deployment",
			Success:    false,
			Message:    "No previous revision found for rollback",
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("no previous revision found")
	}

	// Create a rollback annotation
	patch := fmt.Sprintf(`{"metadata":{"annotations":{"deployment.kubernetes.io/revision":"%d"}}}`, previousRevision)
	_, err = e.client.AppsV1().Deployments(deployment.Namespace).Patch(ctx, deployment.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return &Result{
			Action:     "rollback-deployment",
			Success:    false,
			Message:    fmt.Sprintf("Failed to rollback deployment: %v", err),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, err
	}

	logger.Info("Successfully rolled back deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "revision", previousRevision)
	return &Result{
		Action:     "rollback-deployment",
		Success:    true,
		Message:    fmt.Sprintf("Successfully rolled back deployment %s to revision %d", deployment.Name, previousRevision),
		Resource:   deployment.Name,
		Namespace:  deployment.Namespace,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, nil
}

// scaleReplicas scales up replicas for a deployment or replicaset
func (e *Engine) scaleReplicas(ctx context.Context, resource interface{}, namespace string) (*Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()
	
	if !e.config.AutoScaleEnabled {
		return &Result{
			Action:     "scale-replicas",
			Success:    false,
			Message:    "Auto scaling is disabled",
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	switch r := resource.(type) {
	case *corev1.Pod:
		return e.scalePodDeployment(ctx, r)
	case *appsv1.Deployment:
		return e.scaleDeployment(ctx, r)
	default:
		return &Result{
			Action:     "scale-replicas",
			Success:    false,
			Message:    "Resource type not supported for scaling",
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("resource type not supported for scaling")
	}
}

// scalePodDeployment scales the deployment that owns the pod
func (e *Engine) scalePodDeployment(ctx context.Context, pod *corev1.Pod) (*Result, error) {
	startTime := time.Now()
	
	// Find the deployment that owns this pod
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "ReplicaSet" {
			// Get the replicaset to find its owner deployment
			replicaSet, err := e.client.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				return &Result{
					Action:     "scale-replicas",
					Success:    false,
					Message:    fmt.Sprintf("Failed to get replicaset: %v", err),
					Resource:   pod.Name,
					Namespace:  pod.Namespace,
					ExecutedAt: time.Now(),
					Duration:   time.Since(startTime),
				}, err
			}

			for _, rsOwnerRef := range replicaSet.OwnerReferences {
				if rsOwnerRef.Kind == "Deployment" {
					return e.scaleDeployment(ctx, &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      rsOwnerRef.Name,
							Namespace: pod.Namespace,
						},
						Spec: appsv1.DeploymentSpec{
							Replicas: replicaSet.Spec.Replicas,
						},
					})
				}
			}
		}
	}

	return &Result{
		Action:     "scale-replicas",
		Success:    false,
		Message:    "Could not find owning deployment for pod",
		Resource:   pod.Name,
		Namespace:  pod.Namespace,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, fmt.Errorf("could not find owning deployment for pod")
}

// scaleDeployment scales a deployment by increasing replicas
func (e *Engine) scaleDeployment(ctx context.Context, deployment *appsv1.Deployment) (*Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()
	
	// Get the current deployment
	currentDeployment, err := e.client.AppsV1().Deployments(deployment.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil {
		return &Result{
			Action:     "scale-replicas",
			Success:    false,
			Message:    fmt.Sprintf("Failed to get deployment: %v", err),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, err
	}

	// Increase replicas by 50% or add 2, whichever is smaller
	currentReplicas := int32(1)
	if currentDeployment.Spec.Replicas != nil {
		currentReplicas = *currentDeployment.Spec.Replicas
	}
	
	increase := currentReplicas / 2
	if increase < 2 {
		increase = 2
	}
	
	newReplicas := currentReplicas + increase
	
	if e.config.DryRun {
		logger.Info("Dry run: would scale deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "from", currentReplicas, "to", newReplicas)
		return &Result{
			Action:     "scale-replicas",
			Success:    true,
			Message:    fmt.Sprintf("Dry run: would scale deployment %s from %d to %d replicas", deployment.Name, currentReplicas, newReplicas),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	// Scale the deployment
	patch := fmt.Sprintf(`{"spec":{"replicas":%d}}`, newReplicas)
	_, err = e.client.AppsV1().Deployments(deployment.Namespace).Patch(ctx, deployment.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return &Result{
			Action:     "scale-replicas",
			Success:    false,
			Message:    fmt.Sprintf("Failed to scale deployment: %v", err),
			Resource:   deployment.Name,
			Namespace:  deployment.Namespace,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, err
	}

	logger.Info("Successfully scaled deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "from", currentReplicas, "to", newReplicas)
	return &Result{
		Action:     "scale-replicas",
		Success:    true,
		Message:    fmt.Sprintf("Successfully scaled deployment %s from %d to %d replicas", deployment.Name, currentReplicas, newReplicas),
		Resource:   deployment.Name,
		Namespace:  deployment.Namespace,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, nil
}
