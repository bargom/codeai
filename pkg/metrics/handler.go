package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler for the Prometheus metrics endpoint.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(
		r.registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
			ErrorHandling:     promhttp.ContinueOnError,
		},
	)
}

// HandlerWithAuth returns an HTTP handler for the Prometheus metrics endpoint
// that requires authentication via the provided auth function.
func (r *Registry) HandlerWithAuth(authFn func(r *http.Request) bool) http.Handler {
	handler := r.Handler()

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !authFn(req) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, req)
	})
}

// ServeHTTP implements http.Handler for the Registry.
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Handler().ServeHTTP(w, req)
}

// RegisterMetricsRoute registers the /metrics endpoint on a Chi router.
// Example usage:
//
//	r := chi.NewRouter()
//	registry := metrics.NewRegistry(metrics.DefaultConfig())
//	registry.RegisterMetricsRoute(r)
func (r *Registry) RegisterMetricsRoute(mux interface {
	Handle(pattern string, handler http.Handler)
}) {
	mux.Handle("/metrics", r.Handler())
}

// Handler returns the global registry's HTTP handler.
func Handler() http.Handler {
	return Global().Handler()
}
