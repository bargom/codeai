// Package repository provides data access for events.
package repository

import (
	"context"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// EventFilter specifies criteria for filtering events.
type EventFilter struct {
	Types     []bus.EventType
	Sources   []string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// EventRepository defines the interface for event persistence.
type EventRepository interface {
	// SaveEvent persists an event to the database.
	SaveEvent(ctx context.Context, event bus.Event) error

	// GetEvent retrieves an event by its ID.
	GetEvent(ctx context.Context, eventID string) (*bus.Event, error)

	// ListEvents retrieves events matching the filter criteria.
	ListEvents(ctx context.Context, filter EventFilter) ([]bus.Event, error)

	// GetEventsByType retrieves events of a specific type.
	GetEventsByType(ctx context.Context, eventType bus.EventType, limit int) ([]bus.Event, error)

	// GetEventsBySource retrieves events from a specific source.
	GetEventsBySource(ctx context.Context, source string, limit int) ([]bus.Event, error)

	// CountEvents returns the total count of events matching the filter.
	CountEvents(ctx context.Context, filter EventFilter) (int64, error)

	// DeleteOldEvents removes events older than the specified time.
	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)
}
