// Package api provides the HTTP API for CodeAI.
package api

import (
	"net/http"
	"time"

	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new Chi router with all routes and middleware configured.
func NewRouter(h *handlers.Handler) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(jsonContentType)

	// Health check
	r.Get("/health", h.Health)

	// API routes
	r.Route("/deployments", func(r chi.Router) {
		r.Post("/", h.CreateDeployment)
		r.Get("/", h.ListDeployments)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetDeployment)
			r.Put("/", h.UpdateDeployment)
			r.Delete("/", h.DeleteDeployment)
			r.Post("/execute", h.ExecuteDeployment)
			r.Get("/executions", h.ListDeploymentExecutions)
		})
	})

	r.Route("/configs", func(r chi.Router) {
		r.Post("/", h.CreateConfig)
		r.Get("/", h.ListConfigs)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetConfig)
			r.Put("/", h.UpdateConfig)
			r.Delete("/", h.DeleteConfig)
			r.Post("/validate", h.ValidateConfig)
		})
	})

	r.Route("/executions", func(r chi.Router) {
		r.Get("/", h.ListExecutions)
		r.Get("/{id}", h.GetExecution)
	})

	return r
}

// jsonContentType is middleware that sets the Content-Type header to application/json.
func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
