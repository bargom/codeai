package handlers

import (
	"context"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/event/repository"
)

// JobEventHandler handles job-related events.
type JobEventHandler struct {
	repository repository.EventRepository
	logger     bus.Logger
}

// NewJobEventHandler creates a new JobEventHandler.
func NewJobEventHandler(repo repository.EventRepository, logger bus.Logger) *JobEventHandler {
	return &JobEventHandler{
		repository: repo,
		logger:     logger,
	}
}

// Handle processes job events.
func (h *JobEventHandler) Handle(ctx context.Context, event bus.Event) error {
	switch event.Type {
	case bus.EventJobEnqueued:
		return h.handleJobEnqueued(ctx, event)
	case bus.EventJobStarted:
		return h.handleJobStarted(ctx, event)
	case bus.EventJobCompleted:
		return h.handleJobCompleted(ctx, event)
	case bus.EventJobFailed:
		return h.handleJobFailed(ctx, event)
	default:
		// Unknown job event type, ignore
		return nil
	}
}

// handleJobEnqueued processes job enqueued events.
func (h *JobEventHandler) handleJobEnqueued(ctx context.Context, event bus.Event) error {
	jobID, _ := event.Data["jobID"].(string)
	jobType, _ := event.Data["jobType"].(string)

	if h.logger != nil {
		h.logger.Info("job enqueued",
			"jobID", jobID,
			"jobType", jobType,
			"eventID", event.ID,
		)
	}

	return nil
}

// handleJobStarted processes job started events.
func (h *JobEventHandler) handleJobStarted(ctx context.Context, event bus.Event) error {
	jobID, _ := event.Data["jobID"].(string)
	workerID, _ := event.Data["workerID"].(string)

	if h.logger != nil {
		h.logger.Info("job started",
			"jobID", jobID,
			"workerID", workerID,
			"eventID", event.ID,
		)
	}

	return nil
}

// handleJobCompleted processes job completed events.
func (h *JobEventHandler) handleJobCompleted(ctx context.Context, event bus.Event) error {
	jobID, _ := event.Data["jobID"].(string)
	duration, _ := event.Data["duration"].(float64)

	if h.logger != nil {
		h.logger.Info("job completed",
			"jobID", jobID,
			"duration", duration,
			"eventID", event.ID,
		)
	}

	return nil
}

// handleJobFailed processes job failed events.
func (h *JobEventHandler) handleJobFailed(ctx context.Context, event bus.Event) error {
	jobID, _ := event.Data["jobID"].(string)
	errorMsg, _ := event.Data["error"].(string)
	attempts, _ := event.Data["attempts"].(float64)

	if h.logger != nil {
		h.logger.Error("job failed",
			"jobID", jobID,
			"error", errorMsg,
			"attempts", int(attempts),
			"eventID", event.ID,
		)
	}

	return nil
}

// SupportedEventTypes returns the event types this handler supports.
func (h *JobEventHandler) SupportedEventTypes() []bus.EventType {
	return []bus.EventType{
		bus.EventJobEnqueued,
		bus.EventJobStarted,
		bus.EventJobCompleted,
		bus.EventJobFailed,
	}
}
