package rbac

import (
	"encoding/json"
	"net/http"

	"github.com/bargom/codeai/internal/auth"
)

// Middleware provides HTTP middleware for authorization checks.
type Middleware struct {
	engine *Engine
}

// NewMiddleware creates a new RBAC middleware with the given engine.
func NewMiddleware(engine *Engine) *Middleware {
	return &Middleware{engine: engine}
}

// RequirePermission returns middleware that checks for a specific permission.
// Must be used after authentication middleware.
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !m.engine.CheckPermission(r.Context(), user, Permission(permission)) {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission returns middleware that checks for any of the specified permissions.
func (m *Middleware) RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, p := range permissions {
				if m.engine.CheckPermission(r.Context(), user, Permission(p)) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// RequireAllPermissions returns middleware that checks for all specified permissions.
func (m *Middleware) RequireAllPermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, p := range permissions {
				if !m.engine.CheckPermission(r.Context(), user, Permission(p)) {
					writeError(w, http.StatusForbidden, "insufficient permissions")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that checks for a specific role.
func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !m.engine.CheckRole(r.Context(), user, role) {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole returns middleware that checks for any of the specified roles.
func (m *Middleware) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, role := range roles {
				if m.engine.CheckRole(r.Context(), user, role) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// ResourcePermission returns middleware that checks for a permission on a specific resource.
// The resource is extracted from the request using the provided function.
func (m *Middleware) ResourcePermission(action string, resourceExtractor func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			resource := resourceExtractor(r)
			permission := Permission(resource + ":" + action)

			if !m.engine.CheckPermission(r.Context(), user, permission) {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthorizeFunc is a function type for custom authorization logic.
type AuthorizeFunc func(r *http.Request, user *auth.User) bool

// Custom returns middleware with custom authorization logic.
func (m *Middleware) Custom(authorize AuthorizeFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !authorize(r, user) {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// errorResponse represents a JSON error response.
type errorResponse struct {
	Error string `json:"error"`
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}
