package bus

import (
	"context"
	"sync"
)

// EventBus manages event subscriptions and publishes events to subscribers.
type EventBus struct {
	subscribers map[EventType][]Subscriber
	mu          sync.RWMutex
	logger      Logger
	asyncBuffer chan asyncEvent
	wg          sync.WaitGroup
	closed      bool
	closeMu     sync.RWMutex
}

// asyncEvent wraps an event for async processing.
type asyncEvent struct {
	ctx   context.Context
	event Event
}

// Config holds configuration for the event bus.
type Config struct {
	AsyncBufferSize int
	WorkerPoolSize  int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		AsyncBufferSize: 1000,
		WorkerPoolSize:  10,
	}
}

// NewEventBus creates a new EventBus with the given logger.
func NewEventBus(logger Logger) *EventBus {
	return NewEventBusWithConfig(logger, DefaultConfig())
}

// NewEventBusWithConfig creates a new EventBus with the given logger and configuration.
func NewEventBusWithConfig(logger Logger, cfg Config) *EventBus {
	eb := &EventBus{
		subscribers: make(map[EventType][]Subscriber),
		logger:      logger,
		asyncBuffer: make(chan asyncEvent, cfg.AsyncBufferSize),
	}

	// Start worker pool for async events
	for i := 0; i < cfg.WorkerPoolSize; i++ {
		eb.wg.Add(1)
		go eb.worker()
	}

	return eb
}

// worker processes async events from the buffer.
func (eb *EventBus) worker() {
	defer eb.wg.Done()
	for ae := range eb.asyncBuffer {
		_ = eb.Publish(ae.ctx, ae.event)
	}
}

// Subscribe adds a subscriber for the given event type.
func (eb *EventBus) Subscribe(eventType EventType, subscriber Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], subscriber)
	if eb.logger != nil {
		eb.logger.Debug("subscriber added", "eventType", string(eventType))
	}
}

// Unsubscribe removes a subscriber for the given event type.
func (eb *EventBus) Unsubscribe(eventType EventType, subscriber Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscribers[eventType]
	for i, s := range subs {
		// Compare by pointer identity
		if &s == &subscriber {
			eb.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			if eb.logger != nil {
				eb.logger.Debug("subscriber removed", "eventType", string(eventType))
			}
			return
		}
	}
}

// Publish sends an event to all subscribers of the event type.
// Errors from individual subscribers are logged but don't affect other subscribers.
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	subs := make([]Subscriber, len(eb.subscribers[event.Type]))
	copy(subs, eb.subscribers[event.Type])
	eb.mu.RUnlock()

	if len(subs) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for event", "eventType", string(event.Type), "eventID", event.ID)
		}
		return nil
	}

	// Publish to all subscribers, isolating errors
	for _, sub := range subs {
		if err := eb.publishToSubscriber(ctx, sub, event); err != nil {
			if eb.logger != nil {
				eb.logger.Error("subscriber error",
					"eventType", string(event.Type),
					"eventID", event.ID,
					"error", err.Error(),
				)
			}
			// Continue to other subscribers despite error
		}
	}

	return nil
}

// publishToSubscriber safely calls a subscriber with panic recovery.
func (eb *EventBus) publishToSubscriber(ctx context.Context, sub Subscriber, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if eb.logger != nil {
				eb.logger.Error("subscriber panicked", "eventType", string(event.Type), "eventID", event.ID, "panic", r)
			}
		}
	}()
	return sub.Handle(ctx, event)
}

// PublishAsync queues an event for asynchronous processing.
// Returns immediately without waiting for subscribers.
func (eb *EventBus) PublishAsync(ctx context.Context, event Event) {
	eb.closeMu.RLock()
	defer eb.closeMu.RUnlock()

	if eb.closed {
		if eb.logger != nil {
			eb.logger.Warn("attempted to publish async to closed bus", "eventType", string(event.Type), "eventID", event.ID)
		}
		return
	}

	select {
	case eb.asyncBuffer <- asyncEvent{ctx: ctx, event: event}:
		// Event queued successfully
	default:
		// Buffer full, log warning
		if eb.logger != nil {
			eb.logger.Warn("async buffer full, dropping event", "eventType", string(event.Type), "eventID", event.ID)
		}
	}
}

// SubscriberCount returns the number of subscribers for an event type.
func (eb *EventBus) SubscriberCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers[eventType])
}

// Close gracefully shuts down the event bus, draining the async buffer.
func (eb *EventBus) Close() {
	eb.closeMu.Lock()
	eb.closed = true
	eb.closeMu.Unlock()

	close(eb.asyncBuffer)
	eb.wg.Wait()
}
