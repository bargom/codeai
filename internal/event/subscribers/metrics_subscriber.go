package subscribers

import (
	"context"
	"sync"

	"github.com/bargom/codeai/internal/event/bus"
)

// MetricsSubscriber tracks event metrics.
type MetricsSubscriber struct {
	mu         sync.RWMutex
	counters   map[string]int64
	typeCount  map[bus.EventType]int64
	sourceCount map[string]int64
}

// NewMetricsSubscriber creates a new MetricsSubscriber.
func NewMetricsSubscriber() *MetricsSubscriber {
	return &MetricsSubscriber{
		counters:    make(map[string]int64),
		typeCount:   make(map[bus.EventType]int64),
		sourceCount: make(map[string]int64),
	}
}

// Handle increments counters for the received event.
func (s *MetricsSubscriber) Handle(ctx context.Context, event bus.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Increment total counter
	s.counters["total"]++

	// Increment type counter
	s.typeCount[event.Type]++

	// Increment source counter
	s.sourceCount[event.Source]++

	return nil
}

// GetTotalCount returns the total number of events processed.
func (s *MetricsSubscriber) GetTotalCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.counters["total"]
}

// GetTypeCount returns the count for a specific event type.
func (s *MetricsSubscriber) GetTypeCount(eventType bus.EventType) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.typeCount[eventType]
}

// GetSourceCount returns the count for a specific source.
func (s *MetricsSubscriber) GetSourceCount(source string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sourceCount[source]
}

// GetStats returns all metrics as a map.
func (s *MetricsSubscriber) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total"] = s.counters["total"]

	typeStats := make(map[string]int64)
	for t, c := range s.typeCount {
		typeStats[string(t)] = c
	}
	stats["by_type"] = typeStats

	sourceStats := make(map[string]int64)
	for src, c := range s.sourceCount {
		sourceStats[src] = c
	}
	stats["by_source"] = sourceStats

	return stats
}

// Reset clears all counters.
func (s *MetricsSubscriber) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters = make(map[string]int64)
	s.typeCount = make(map[bus.EventType]int64)
	s.sourceCount = make(map[string]int64)
}
