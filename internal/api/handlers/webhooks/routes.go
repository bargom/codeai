package webhooks

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers all webhook routes on the given router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/webhooks", func(r chi.Router) {
		// Webhook CRUD
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)

		// Webhook actions
		r.Post("/{id}/test", h.Test)
		r.Get("/{id}/deliveries", h.ListDeliveries)

		// Delivery actions
		r.Post("/deliveries/{id}/retry", h.RetryDelivery)
	})
}
