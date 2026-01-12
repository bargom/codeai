// Package subscribers provides built-in event subscribers.
package subscribers

import (
	"context"

	"github.com/bargom/codeai/internal/event/bus"
)

// LoggingSubscriber logs events as they are received.
type LoggingSubscriber struct {
	logger bus.Logger
}

// NewLoggingSubscriber creates a new LoggingSubscriber.
func NewLoggingSubscriber(logger bus.Logger) *LoggingSubscriber {
	return &LoggingSubscriber{logger: logger}
}

// Handle logs the received event.
func (s *LoggingSubscriber) Handle(ctx context.Context, event bus.Event) error {
	s.logger.Info("event received",
		"eventID", event.ID,
		"type", string(event.Type),
		"source", event.Source,
		"timestamp", event.Timestamp.String(),
	)
	return nil
}
