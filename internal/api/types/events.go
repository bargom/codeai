package types

import (
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// EventResponse represents an event in API responses.
type EventResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
}

// EventFromBus converts a bus.Event to an API response.
func EventFromBus(e bus.Event) *EventResponse {
	return &EventResponse{
		ID:        e.ID,
		Type:      string(e.Type),
		Source:    e.Source,
		Timestamp: e.Timestamp,
		Data:      e.Data,
		Metadata:  e.Metadata,
	}
}

// EventsFromBus converts a slice of bus.Events to API responses.
func EventsFromBus(events []bus.Event) []*EventResponse {
	responses := make([]*EventResponse, len(events))
	for i, e := range events {
		responses[i] = EventFromBus(e)
	}
	return responses
}

// EventTypeInfo describes an event type.
type EventTypeInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// EventStatsResponse represents event statistics.
type EventStatsResponse struct {
	TotalEvents int64            `json:"total_events"`
	ByType      map[string]int64 `json:"by_type"`
	BySource    map[string]int64 `json:"by_source"`
}

// ListEventsRequest represents a request to list events.
type ListEventsRequest struct {
	Types     []string `json:"types"`
	Sources   []string `json:"sources"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

// GetEventTypes returns all available event types with descriptions.
func GetEventTypes() []EventTypeInfo {
	return []EventTypeInfo{
		{Type: string(bus.EventWorkflowStarted), Description: "Workflow execution has started", Category: "workflow"},
		{Type: string(bus.EventWorkflowCompleted), Description: "Workflow execution completed successfully", Category: "workflow"},
		{Type: string(bus.EventWorkflowFailed), Description: "Workflow execution failed", Category: "workflow"},
		{Type: string(bus.EventJobEnqueued), Description: "Job has been added to the queue", Category: "job"},
		{Type: string(bus.EventJobStarted), Description: "Job processing has started", Category: "job"},
		{Type: string(bus.EventJobCompleted), Description: "Job completed successfully", Category: "job"},
		{Type: string(bus.EventJobFailed), Description: "Job processing failed", Category: "job"},
		{Type: string(bus.EventAgentExecuted), Description: "Agent has executed an action", Category: "agent"},
		{Type: string(bus.EventTestSuiteCompleted), Description: "Test suite execution completed", Category: "test"},
		{Type: string(bus.EventWebhookTriggered), Description: "Webhook was triggered", Category: "webhook"},
		{Type: string(bus.EventEmailSent), Description: "Email was sent", Category: "notification"},
	}
}
