// Package subscriber provides event bus integration for webhooks.
package subscriber

import (
	"context"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/webhook/service"
)

// WebhookEventSubscriber listens to events and triggers webhook deliveries.
type WebhookEventSubscriber struct {
	webhookService *service.WebhookService
	eventTypes     []bus.EventType
	logger         Logger
}

// Logger defines the logging interface for the subscriber.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Option configures the WebhookEventSubscriber.
type Option func(*WebhookEventSubscriber)

// WithLogger sets the logger for the subscriber.
func WithLogger(logger Logger) Option {
	return func(s *WebhookEventSubscriber) {
		s.logger = logger
	}
}

// WithEventTypes sets the event types to subscribe to.
func WithEventTypes(types []bus.EventType) Option {
	return func(s *WebhookEventSubscriber) {
		s.eventTypes = types
	}
}

// NewWebhookEventSubscriber creates a new webhook event subscriber.
// By default, it subscribes to common workflow and job events.
func NewWebhookEventSubscriber(svc *service.WebhookService, opts ...Option) *WebhookEventSubscriber {
	s := &WebhookEventSubscriber{
		webhookService: svc,
		eventTypes: []bus.EventType{
			bus.EventWorkflowStarted,
			bus.EventWorkflowCompleted,
			bus.EventWorkflowFailed,
			bus.EventJobEnqueued,
			bus.EventJobStarted,
			bus.EventJobCompleted,
			bus.EventJobFailed,
			bus.EventAgentExecuted,
			bus.EventTestSuiteCompleted,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Handle processes an event and delivers it to subscribed webhooks.
func (s *WebhookEventSubscriber) Handle(ctx context.Context, event bus.Event) error {
	if s.logger != nil {
		s.logger.Debug("handling event for webhooks",
			"eventType", string(event.Type),
			"eventID", event.ID,
		)
	}

	if err := s.webhookService.DeliverWebhooksForEvent(ctx, event); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to deliver webhooks for event",
				"eventType", string(event.Type),
				"eventID", event.ID,
				"error", err.Error(),
			)
		}
		return err
	}

	return nil
}

// SubscribedEvents returns the event types this subscriber handles.
func (s *WebhookEventSubscriber) SubscribedEvents() []bus.EventType {
	return s.eventTypes
}

// RegisterWithBus subscribes this handler to all its event types on the bus.
func (s *WebhookEventSubscriber) RegisterWithBus(eventBus *bus.EventBus) {
	for _, eventType := range s.eventTypes {
		eventBus.Subscribe(eventType, s)
	}

	if s.logger != nil {
		s.logger.Info("webhook subscriber registered",
			"eventTypes", len(s.eventTypes),
		)
	}
}

// UnregisterFromBus removes this handler from all its event types on the bus.
func (s *WebhookEventSubscriber) UnregisterFromBus(eventBus *bus.EventBus) {
	for _, eventType := range s.eventTypes {
		eventBus.Unsubscribe(eventType, s)
	}

	if s.logger != nil {
		s.logger.Info("webhook subscriber unregistered",
			"eventTypes", len(s.eventTypes),
		)
	}
}
