// Package dispatcher provides event dispatching with persistence.
package dispatcher

import (
	"context"
	"fmt"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/event/repository"
)

// EventDispatcher dispatches events to the bus and optionally persists them.
type EventDispatcher struct {
	bus        *bus.EventBus
	repository repository.EventRepository
	logger     bus.Logger
	persist    bool
}

// Option configures the EventDispatcher.
type Option func(*EventDispatcher)

// WithRepository enables event persistence.
func WithRepository(repo repository.EventRepository) Option {
	return func(d *EventDispatcher) {
		d.repository = repo
		d.persist = true
	}
}

// WithLogger sets the logger for the dispatcher.
func WithLogger(logger bus.Logger) Option {
	return func(d *EventDispatcher) {
		d.logger = logger
	}
}

// NewDispatcher creates a new EventDispatcher.
func NewDispatcher(b *bus.EventBus, opts ...Option) *EventDispatcher {
	d := &EventDispatcher{
		bus:     b,
		persist: false,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Dispatch publishes an event to subscribers and persists it if configured.
func (d *EventDispatcher) Dispatch(ctx context.Context, event bus.Event) error {
	// Persist event first if repository is configured
	if d.persist && d.repository != nil {
		if err := d.repository.SaveEvent(ctx, event); err != nil {
			if d.logger != nil {
				d.logger.Error("failed to persist event",
					"eventID", event.ID,
					"eventType", string(event.Type),
					"error", err.Error(),
				)
			}
			return fmt.Errorf("persisting event: %w", err)
		}
	}

	// Publish to bus
	if err := d.bus.Publish(ctx, event); err != nil {
		if d.logger != nil {
			d.logger.Error("failed to publish event",
				"eventID", event.ID,
				"eventType", string(event.Type),
				"error", err.Error(),
			)
		}
		return fmt.Errorf("publishing event: %w", err)
	}

	if d.logger != nil {
		d.logger.Debug("event dispatched",
			"eventID", event.ID,
			"eventType", string(event.Type),
		)
	}

	return nil
}

// DispatchAsync publishes an event asynchronously without blocking.
func (d *EventDispatcher) DispatchAsync(ctx context.Context, event bus.Event) {
	// For async dispatch, we still persist synchronously to ensure durability
	if d.persist && d.repository != nil {
		if err := d.repository.SaveEvent(ctx, event); err != nil {
			if d.logger != nil {
				d.logger.Error("failed to persist async event",
					"eventID", event.ID,
					"eventType", string(event.Type),
					"error", err.Error(),
				)
			}
			return
		}
	}

	// Publish asynchronously
	d.bus.PublishAsync(ctx, event)

	if d.logger != nil {
		d.logger.Debug("event dispatched async",
			"eventID", event.ID,
			"eventType", string(event.Type),
		)
	}
}

// Subscribe adds a subscriber for the given event type.
func (d *EventDispatcher) Subscribe(eventType bus.EventType, subscriber bus.Subscriber) {
	d.bus.Subscribe(eventType, subscriber)
}

// Unsubscribe removes a subscriber for the given event type.
func (d *EventDispatcher) Unsubscribe(eventType bus.EventType, subscriber bus.Subscriber) {
	d.bus.Unsubscribe(eventType, subscriber)
}

// Close gracefully shuts down the dispatcher.
func (d *EventDispatcher) Close() {
	d.bus.Close()
}
