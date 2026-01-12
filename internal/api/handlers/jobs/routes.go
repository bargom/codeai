package jobs

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers job API routes on the given router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/jobs", func(r chi.Router) {
		// List jobs
		r.Get("/", h.List)

		// Get queue statistics
		r.Get("/stats", h.GetStats)

		// Submit a new job for immediate processing
		r.Post("/", h.Submit)

		// Schedule a job for future execution
		r.Post("/schedule", h.Schedule)

		// Create a recurring job
		r.Post("/recurring", h.CreateRecurring)

		// Job-specific operations
		r.Route("/{id}", func(r chi.Router) {
			// Get job status
			r.Get("/", h.GetStatus)

			// Cancel a job
			r.Delete("/", h.Cancel)
		})
	})
}
