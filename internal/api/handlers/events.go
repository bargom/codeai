package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/event/repository"
	"github.com/bargom/codeai/internal/event/subscribers"
	"github.com/go-chi/chi/v5"
)

// EventHandler provides HTTP handlers for events API.
type EventHandler struct {
	repository repository.EventRepository
	metrics    *subscribers.MetricsSubscriber
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(repo repository.EventRepository, metrics *subscribers.MetricsSubscriber) *EventHandler {
	return &EventHandler{
		repository: repo,
		metrics:    metrics,
	}
}

// ListEvents handles GET /events.
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPaginationParams(r)

	filter := repository.EventFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Parse type filter
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		filter.Types = []bus.EventType{bus.EventType(typeParam)}
	}

	// Parse source filter
	if sourceParam := r.URL.Query().Get("source"); sourceParam != "" {
		filter.Sources = []string{sourceParam}
	}

	// Parse time filters
	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}

	events, err := h.repository.ListEvents(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	responses := types.EventsFromBus(events)
	respondJSON(w, http.StatusOK, types.NewListResponse(responses, limit, offset))
}

// GetEvent handles GET /events/{id}.
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	event, err := h.repository.GetEvent(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			respondError(w, http.StatusNotFound, "event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get event")
		return
	}

	respondJSON(w, http.StatusOK, types.EventFromBus(*event))
}

// ListEventTypes handles GET /events/types.
func (h *EventHandler) ListEventTypes(w http.ResponseWriter, r *http.Request) {
	eventTypes := types.GetEventTypes()
	respondJSON(w, http.StatusOK, eventTypes)
}

// GetEventStats handles GET /events/stats.
func (h *EventHandler) GetEventStats(w http.ResponseWriter, r *http.Request) {
	// If we have a metrics subscriber, use its stats
	if h.metrics != nil {
		stats := h.metrics.GetStats()

		response := types.EventStatsResponse{
			TotalEvents: h.metrics.GetTotalCount(),
			ByType:      make(map[string]int64),
			BySource:    make(map[string]int64),
		}

		if byType, ok := stats["by_type"].(map[string]int64); ok {
			response.ByType = byType
		}
		if bySource, ok := stats["by_source"].(map[string]int64); ok {
			response.BySource = bySource
		}

		respondJSON(w, http.StatusOK, response)
		return
	}

	// Otherwise, get stats from repository
	filter := repository.EventFilter{}
	total, err := h.repository.CountEvents(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get event stats")
		return
	}

	response := types.EventStatsResponse{
		TotalEvents: total,
		ByType:      make(map[string]int64),
		BySource:    make(map[string]int64),
	}

	respondJSON(w, http.StatusOK, response)
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		if err := encodeJSON(w, data); err != nil {
			return
		}
	}
}

// respondError writes a JSON error response with the given status code.
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, types.ErrorResponse{Error: message})
}

// encodeJSON encodes data as JSON.
func encodeJSON(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}
