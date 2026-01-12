// Package handlers provides specialized event handlers for complex event processing.
package handlers

import (
	"context"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/event/repository"
)

// WorkflowEventHandler handles workflow-related events.
type WorkflowEventHandler struct {
	repository repository.EventRepository
	logger     bus.Logger
}

// NewWorkflowEventHandler creates a new WorkflowEventHandler.
func NewWorkflowEventHandler(repo repository.EventRepository, logger bus.Logger) *WorkflowEventHandler {
	return &WorkflowEventHandler{
		repository: repo,
		logger:     logger,
	}
}

// Handle processes workflow events.
func (h *WorkflowEventHandler) Handle(ctx context.Context, event bus.Event) error {
	switch event.Type {
	case bus.EventWorkflowStarted:
		return h.handleWorkflowStarted(ctx, event)
	case bus.EventWorkflowCompleted:
		return h.handleWorkflowCompleted(ctx, event)
	case bus.EventWorkflowFailed:
		return h.handleWorkflowFailed(ctx, event)
	default:
		// Unknown workflow event type, ignore
		return nil
	}
}

// handleWorkflowStarted processes workflow started events.
func (h *WorkflowEventHandler) handleWorkflowStarted(ctx context.Context, event bus.Event) error {
	workflowID, _ := event.Data["workflowID"].(string)

	if h.logger != nil {
		h.logger.Info("workflow started",
			"workflowID", workflowID,
			"eventID", event.ID,
		)
	}

	return nil
}

// handleWorkflowCompleted processes workflow completed events.
func (h *WorkflowEventHandler) handleWorkflowCompleted(ctx context.Context, event bus.Event) error {
	workflowID, _ := event.Data["workflowID"].(string)
	duration, _ := event.Data["duration"].(float64)

	if h.logger != nil {
		h.logger.Info("workflow completed",
			"workflowID", workflowID,
			"duration", duration,
			"eventID", event.ID,
		)
	}

	return nil
}

// handleWorkflowFailed processes workflow failed events.
func (h *WorkflowEventHandler) handleWorkflowFailed(ctx context.Context, event bus.Event) error {
	workflowID, _ := event.Data["workflowID"].(string)
	errorMsg, _ := event.Data["error"].(string)

	if h.logger != nil {
		h.logger.Error("workflow failed",
			"workflowID", workflowID,
			"error", errorMsg,
			"eventID", event.ID,
		)
	}

	return nil
}

// SupportedEventTypes returns the event types this handler supports.
func (h *WorkflowEventHandler) SupportedEventTypes() []bus.EventType {
	return []bus.EventType{
		bus.EventWorkflowStarted,
		bus.EventWorkflowCompleted,
		bus.EventWorkflowFailed,
	}
}
