// Package builder provides a fluent API for constructing events.
package builder

import (
	"time"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/google/uuid"
)

// EventBuilder provides a fluent interface for building events.
type EventBuilder struct {
	event bus.Event
}

// NewEvent creates a new EventBuilder with the given event type.
func NewEvent(eventType bus.EventType) *EventBuilder {
	return &EventBuilder{
		event: bus.Event{
			ID:        uuid.New().String(),
			Type:      eventType,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]string),
		},
	}
}

// WithID sets a custom ID for the event.
func (b *EventBuilder) WithID(id string) *EventBuilder {
	b.event.ID = id
	return b
}

// WithSource sets the source of the event.
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithTimestamp sets a custom timestamp for the event.
func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.event.Timestamp = t
	return b
}

// WithData adds a key-value pair to the event data.
func (b *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	b.event.Data[key] = value
	return b
}

// WithDataMap merges the given map into the event data.
func (b *EventBuilder) WithDataMap(data map[string]interface{}) *EventBuilder {
	for k, v := range data {
		b.event.Data[k] = v
	}
	return b
}

// WithMetadata adds a key-value pair to the event metadata.
func (b *EventBuilder) WithMetadata(key, value string) *EventBuilder {
	b.event.Metadata[key] = value
	return b
}

// WithMetadataMap merges the given map into the event metadata.
func (b *EventBuilder) WithMetadataMap(metadata map[string]string) *EventBuilder {
	for k, v := range metadata {
		b.event.Metadata[k] = v
	}
	return b
}

// Build returns the constructed event.
func (b *EventBuilder) Build() bus.Event {
	return b.event
}

// WorkflowStarted creates a builder for a workflow started event.
func WorkflowStarted(workflowID string) *EventBuilder {
	return NewEvent(bus.EventWorkflowStarted).
		WithSource("workflow-engine").
		WithData("workflowID", workflowID)
}

// WorkflowCompleted creates a builder for a workflow completed event.
func WorkflowCompleted(workflowID string, duration float64) *EventBuilder {
	return NewEvent(bus.EventWorkflowCompleted).
		WithSource("workflow-engine").
		WithData("workflowID", workflowID).
		WithData("duration", duration)
}

// WorkflowFailed creates a builder for a workflow failed event.
func WorkflowFailed(workflowID string, err error) *EventBuilder {
	return NewEvent(bus.EventWorkflowFailed).
		WithSource("workflow-engine").
		WithData("workflowID", workflowID).
		WithData("error", err.Error())
}

// JobEnqueued creates a builder for a job enqueued event.
func JobEnqueued(jobID, jobType string) *EventBuilder {
	return NewEvent(bus.EventJobEnqueued).
		WithSource("job-queue").
		WithData("jobID", jobID).
		WithData("jobType", jobType)
}

// JobStarted creates a builder for a job started event.
func JobStarted(jobID, workerID string) *EventBuilder {
	return NewEvent(bus.EventJobStarted).
		WithSource("job-worker").
		WithData("jobID", jobID).
		WithData("workerID", workerID)
}

// JobCompleted creates a builder for a job completed event.
func JobCompleted(jobID string, duration float64) *EventBuilder {
	return NewEvent(bus.EventJobCompleted).
		WithSource("job-worker").
		WithData("jobID", jobID).
		WithData("duration", duration)
}

// JobFailed creates a builder for a job failed event.
func JobFailed(jobID string, err error, attempts int) *EventBuilder {
	return NewEvent(bus.EventJobFailed).
		WithSource("job-worker").
		WithData("jobID", jobID).
		WithData("error", err.Error()).
		WithData("attempts", attempts)
}

// AgentExecuted creates a builder for an agent executed event.
func AgentExecuted(agentID, action string) *EventBuilder {
	return NewEvent(bus.EventAgentExecuted).
		WithSource("agent-runtime").
		WithData("agentID", agentID).
		WithData("action", action)
}

// TestSuiteCompleted creates a builder for a test suite completed event.
func TestSuiteCompleted(suiteID string, passed, failed, skipped int) *EventBuilder {
	return NewEvent(bus.EventTestSuiteCompleted).
		WithSource("test-runner").
		WithData("suiteID", suiteID).
		WithData("passed", passed).
		WithData("failed", failed).
		WithData("skipped", skipped)
}

// WebhookTriggered creates a builder for a webhook triggered event.
func WebhookTriggered(webhookID, url string) *EventBuilder {
	return NewEvent(bus.EventWebhookTriggered).
		WithSource("webhook-service").
		WithData("webhookID", webhookID).
		WithData("url", url)
}

// EmailSent creates a builder for an email sent event.
func EmailSent(emailID, recipient string) *EventBuilder {
	return NewEvent(bus.EventEmailSent).
		WithSource("email-service").
		WithData("emailID", emailID).
		WithData("recipient", recipient)
}
