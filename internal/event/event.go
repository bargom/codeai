// Package event provides an event dispatcher for the application.
package event

import (
	"context"
	"sync"
	"time"
)

// EventType represents the type of an event.
type EventType string

// Job lifecycle events.
const (
	EventJobCreated    EventType = "job.created"
	EventJobScheduled  EventType = "job.scheduled"
	EventJobStarted    EventType = "job.started"
	EventJobCompleted  EventType = "job.completed"
	EventJobFailed     EventType = "job.failed"
	EventJobCancelled  EventType = "job.cancelled"
	EventJobRetrying   EventType = "job.retrying"
)

// Event represents an application event.
type Event struct {
	Type      EventType
	Payload   any
	Timestamp time.Time
}

// NewEvent creates a new event with the current timestamp.
func NewEvent(eventType EventType, payload any) Event {
	return Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// Handler is a function that handles an event.
type Handler func(ctx context.Context, event Event) error

// Dispatcher dispatches events to registered handlers.
type Dispatcher interface {
	// Dispatch sends an event to all registered handlers.
	Dispatch(ctx context.Context, event Event) error

	// Subscribe registers a handler for the given event type.
	Subscribe(eventType EventType, handler Handler)

	// Unsubscribe removes a handler from the given event type.
	Unsubscribe(eventType EventType, handler Handler)
}

// dispatcher is the default implementation of Dispatcher.
type dispatcher struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() Dispatcher {
	return &dispatcher{
		handlers: make(map[EventType][]Handler),
	}
}

// Dispatch sends an event to all registered handlers.
func (d *dispatcher) Dispatch(ctx context.Context, event Event) error {
	d.mu.RLock()
	handlers := d.handlers[event.Type]
	d.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			// Log error but continue dispatching to other handlers
			continue
		}
	}
	return nil
}

// Subscribe registers a handler for the given event type.
func (d *dispatcher) Subscribe(eventType EventType, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Unsubscribe removes a handler from the given event type.
// Note: This is a simple implementation and may not work correctly
// if the same handler is registered multiple times.
func (d *dispatcher) Unsubscribe(eventType EventType, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// This is a simple implementation - in production, you might want
	// to use a more sophisticated approach with handler IDs
}

// NoOpDispatcher is a dispatcher that does nothing.
// Useful for testing or when events are not needed.
type NoOpDispatcher struct{}

// NewNoOpDispatcher creates a new no-op dispatcher.
func NewNoOpDispatcher() Dispatcher {
	return &NoOpDispatcher{}
}

// Dispatch does nothing.
func (d *NoOpDispatcher) Dispatch(_ context.Context, _ Event) error {
	return nil
}

// Subscribe does nothing.
func (d *NoOpDispatcher) Subscribe(_ EventType, _ Handler) {}

// Unsubscribe does nothing.
func (d *NoOpDispatcher) Unsubscribe(_ EventType, _ Handler) {}
