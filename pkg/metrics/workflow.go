package metrics

import (
	"time"
)

// WorkflowMetrics provides methods to record workflow-related metrics.
type WorkflowMetrics struct {
	registry *Registry
}

// Workflow returns the workflow metrics interface for the registry.
func (r *Registry) Workflow() *WorkflowMetrics {
	return &WorkflowMetrics{registry: r}
}

// WorkflowStatus represents the outcome status of a workflow execution.
type WorkflowStatus string

const (
	WorkflowStatusSuccess   WorkflowStatus = "success"
	WorkflowStatusFailure   WorkflowStatus = "failure"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
	WorkflowStatusTimeout   WorkflowStatus = "timeout"
)

// RecordExecution records metrics for a completed workflow execution.
func (w *WorkflowMetrics) RecordExecution(workflowName string, status WorkflowStatus, duration time.Duration) {
	w.registry.workflowExecutionsTotal.WithLabelValues(
		workflowName,
		string(status),
	).Inc()

	w.registry.workflowExecutionDuration.WithLabelValues(workflowName).Observe(duration.Seconds())
}

// RecordStep records metrics for a completed workflow step.
func (w *WorkflowMetrics) RecordStep(workflowName, stepName string, duration time.Duration) {
	w.registry.workflowStepDuration.WithLabelValues(workflowName, stepName).Observe(duration.Seconds())
}

// IncActiveWorkflows increments the active workflow count.
func (w *WorkflowMetrics) IncActiveWorkflows(workflowName string) {
	w.registry.workflowActiveCount.WithLabelValues(workflowName).Inc()
}

// DecActiveWorkflows decrements the active workflow count.
func (w *WorkflowMetrics) DecActiveWorkflows(workflowName string) {
	w.registry.workflowActiveCount.WithLabelValues(workflowName).Dec()
}

// SetActiveWorkflows sets the active workflow count to a specific value.
func (w *WorkflowMetrics) SetActiveWorkflows(workflowName string, count int) {
	w.registry.workflowActiveCount.WithLabelValues(workflowName).Set(float64(count))
}

// WorkflowExecutionTimer provides a convenient way to time workflow executions.
type WorkflowExecutionTimer struct {
	metrics      *WorkflowMetrics
	workflowName string
	start        time.Time
}

// NewExecutionTimer creates a new workflow execution timer.
func (w *WorkflowMetrics) NewExecutionTimer(workflowName string) *WorkflowExecutionTimer {
	w.IncActiveWorkflows(workflowName)
	return &WorkflowExecutionTimer{
		metrics:      w,
		workflowName: workflowName,
		start:        time.Now(),
	}
}

// Done records the workflow execution duration and status.
func (t *WorkflowExecutionTimer) Done(status WorkflowStatus) {
	duration := time.Since(t.start)
	t.metrics.DecActiveWorkflows(t.workflowName)
	t.metrics.RecordExecution(t.workflowName, status, duration)
}

// Success records the workflow execution as successful.
func (t *WorkflowExecutionTimer) Success() {
	t.Done(WorkflowStatusSuccess)
}

// Failure records the workflow execution as failed.
func (t *WorkflowExecutionTimer) Failure() {
	t.Done(WorkflowStatusFailure)
}

// Cancelled records the workflow execution as cancelled.
func (t *WorkflowExecutionTimer) Cancelled() {
	t.Done(WorkflowStatusCancelled)
}

// Timeout records the workflow execution as timed out.
func (t *WorkflowExecutionTimer) Timeout() {
	t.Done(WorkflowStatusTimeout)
}

// WorkflowStepTimer provides a convenient way to time workflow steps.
type WorkflowStepTimer struct {
	metrics      *WorkflowMetrics
	workflowName string
	stepName     string
	start        time.Time
}

// NewStepTimer creates a new workflow step timer.
func (w *WorkflowMetrics) NewStepTimer(workflowName, stepName string) *WorkflowStepTimer {
	return &WorkflowStepTimer{
		metrics:      w,
		workflowName: workflowName,
		stepName:     stepName,
		start:        time.Now(),
	}
}

// Done records the step duration.
func (t *WorkflowStepTimer) Done() {
	duration := time.Since(t.start)
	t.metrics.RecordStep(t.workflowName, t.stepName, duration)
}

// ExecutionsTotal returns the counter for total workflow executions (for testing).
func (w *WorkflowMetrics) ExecutionsTotal() interface{} {
	return w.registry.workflowExecutionsTotal
}

// ActiveCount returns the gauge for active workflow count (for testing).
func (w *WorkflowMetrics) ActiveCount() interface{} {
	return w.registry.workflowActiveCount
}
