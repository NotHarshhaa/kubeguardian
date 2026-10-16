package detection

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EventIssue represents a detected Kubernetes event issue
type EventIssue struct {
	InvolvedObjectKind string            `json:"involvedObjectKind"`
	InvolvedObjectName string            `json:"involvedObjectName"`
	InvolvedObjectNS   string            `json:"involvedObjectNamespace"`
	Reason             string            `json:"reason"`
	Message            string            `json:"message"`
	Type               string            `json:"type"`
	Severity           string            `json:"severity"`
	FirstTimestamp     time.Time         `json:"firstTimestamp"`
	LastTimestamp      time.Time         `json:"lastTimestamp"`
	Count              int32             `json:"count"`
	Source             string            `json:"source"`
	Labels             map[string]string `json:"labels,omitempty"`
}

// EventDetector watches Kubernetes events for warning-level issues
type EventDetector struct {
	client       kubernetes.Interface
	mu           sync.RWMutex
	seenEvents   map[string]time.Time // deduplication key -> first seen time
	dedupWindow  time.Duration
	eventChan    chan EventIssue
	stopCh       chan struct{}
	watching     bool
}

// NewEventDetector creates a new event detector
func NewEventDetector(client kubernetes.Interface) *EventDetector {
	return &EventDetector{
		client:       client,
		seenEvents:   make(map[string]time.Time),
		dedupWindow:  5 * time.Minute,
		eventChan:    make(chan EventIssue, 100),
		stopCh:       make(chan struct{}),
	}
}

// SetDedupWindow sets the time window for event deduplication
func (d *EventDetector) SetDedupWindow(window time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dedupWindow = window
}

// StartWatch begins watching for events in the background
func (d *EventDetector) StartWatch(ctx context.Context) error {
	d.mu.Lock()
	if d.watching {
		d.mu.Unlock()
		return fmt.Errorf("event watcher already running")
	}
	d.watching = true
	d.mu.Unlock()

	go d.watchLoop(ctx)
	return nil
}

// StopWatch stops the event watcher
func (d *EventDetector) StopWatch() {
	d.mu.Lock()
	d.watching = false
	close(d.stopCh)
	d.mu.Unlock()
}

// EventChannel returns the channel for consuming detected events
func (d *EventDetector) EventChannel() <-chan EventIssue {
	return d.eventChan
}

// DetectEvents polls for recent warning events (non-watch mode)
func (d *EventDetector) DetectEvents(ctx context.Context, namespaces []string, since time.Duration) ([]EventIssue, error) {
	logger := log.FromContext(ctx)
	var issues []EventIssue

	// If no namespaces specified, watch all namespaces
	if len(namespaces) == 0 {
		namespaces = append(namespaces, metav1.NamespaceAll)
	}

	sinceTime := metav1.NewTime(time.Now().Add(-since))

	for _, ns := range namespaces {
		events, err := d.client.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("type", "Warning").String(),
		})
		if err != nil {
			logger.Error(err, "Failed to list events", "namespace", ns)
			continue
		}

		for _, event := range events.Items {
			if event.LastTimestamp.Before(&sinceTime) {
				continue
			}

			eventIssue := d.eventToIssue(&event)
			issues = append(issues, eventIssue)
		}
	}

	return issues, nil
}

// watchLoop watches for new events in the background
func (d *EventDetector) watchLoop(ctx context.Context) {
	logger := log.FromContext(ctx)

	for {
		select {
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Watch events across all namespaces
		w, err := d.client.CoreV1().Events(metav1.NamespaceAll).Watch(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("type", "Warning").String(),
		})
		if err != nil {
			logger.Error(err, "Failed to watch events, retrying in 5s")
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		// Process watch events
		for {
			select {
			case <-d.stopCh:
				w.Stop()
				return
			case <-ctx.Done():
				w.Stop()
				return
			case watchEvent, ok := <-w.ResultChan():
				if !ok {
					// Watch connection closed, will reconnect
					break
				}

				event, ok := watchEvent.Object.(*corev1.Event)
				if !ok {
					continue
				}

				// Check deduplication
				key := fmt.Sprintf("%s:%s:%s:%s", event.InvolvedObject.Kind, event.InvolvedObject.Namespace, event.InvolvedObject.Name, event.Reason)
				d.mu.Lock()
				lastSeen, exists := d.seenEvents[key]
				if exists && time.Since(lastSeen) < d.dedupWindow {
					d.mu.Unlock()
					continue
				}
				d.seenEvents[key] = time.Now()
				d.mu.Unlock()

				eventIssue := d.eventToIssue(event)
				select {
				case d.eventChan <- eventIssue:
				default:
					// Channel full, skip to avoid blocking
				}
			}
		}
	}
}

// eventToIssue converts a Kubernetes Event to an EventIssue
func (d *EventDetector) eventToIssue(event *corev1.Event) EventIssue {
	severity := "medium"
	if event.Reason == "FailedScheduling" || event.Reason == "FailedCreatePod" || event.Reason == "FailedDelete" {
		severity = "high"
	}
	if event.Reason == "OutOfDisk" || event.Reason == "NodeNotReady" {
		severity = "critical"
	}

	labels := map[string]string{}
	if event.InvolvedObject.UID != "" {
		labels["uid"] = string(event.InvolvedObject.UID)
	}

	return EventIssue{
		InvolvedObjectKind: event.InvolvedObject.Kind,
		InvolvedObjectName: event.InvolvedObject.Name,
		InvolvedObjectNS:   event.InvolvedObject.Namespace,
		Reason:             event.Reason,
		Message:            event.Message,
		Type:               event.Type,
		Severity:           severity,
		FirstTimestamp:     event.FirstTimestamp.Time,
		LastTimestamp:      event.LastTimestamp.Time,
		Count:              event.Count,
		Source:             event.Source.Component,
		Labels:             labels,
	}
}

// GetIssues returns all pending events from the channel (non-blocking)
func (d *EventDetector) GetIssues() []EventIssue {
	var issues []EventIssue
	for {
		select {
		case issue := <-d.eventChan:
			issues = append(issues, issue)
		default:
			return issues
		}
	}
}
