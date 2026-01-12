// Package bus provides the core event bus functionality for pub/sub messaging.
package bus

import (
	"context"
	"time"
)

// EventType represents the type of an event.
type EventType string

// Event types for workflow lifecycle.
const (
	EventWorkflowStarted   EventType = "workflow.started"
	EventWorkflowCompleted EventType = "workflow.completed"
	EventWorkflowFailed    EventType = "workflow.failed"
)

// Event types for job lifecycle.
const (
	EventJobEnqueued  EventType = "job.enqueued"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"
)

// Event types for other system events.
const (
	EventAgentExecuted      EventType = "agent.executed"
	EventTestSuiteCompleted EventType = "test.suite.completed"
	EventWebhookTriggered   EventType = "webhook.triggered"
	EventEmailSent          EventType = "email.sent"
)

// Event represents an event in the system.
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
}

// Subscriber is the interface that must be implemented by event subscribers.
type Subscriber interface {
	// Handle processes an event. It should return an error if the event
	// could not be processed successfully.
	Handle(ctx context.Context, event Event) error
}

// SubscriberFunc is a function type that implements the Subscriber interface.
type SubscriberFunc func(ctx context.Context, event Event) error

// Handle calls the function.
func (f SubscriberFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// Logger is the interface for logging within the event bus.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}
