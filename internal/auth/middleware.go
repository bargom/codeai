package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const userContextKey contextKey = "auth_user"

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(userContextKey).(*User); ok {
		return user
	}
	return nil
}

// ContextWithUser returns a new context with the user attached.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// AuthRequirement specifies the authentication requirement for an endpoint.
type AuthRequirement string

const (
	// AuthRequired requires a valid token; returns 401 if missing/invalid.
	AuthRequired AuthRequirement = "required"
	// AuthOptional accepts requests with or without a token; invalid tokens are ignored.
	AuthOptional AuthRequirement = "optional"
	// AuthPublic allows all requests without any token validation.
	AuthPublic AuthRequirement = "public"
)

// Middleware creates HTTP middleware that validates JWTs and attaches user info to the context.
type Middleware struct {
	validator *Validator
}

// NewMiddleware creates a new authentication middleware with the given validator.
func NewMiddleware(validator *Validator) *Middleware {
	return &Middleware{validator: validator}
}

// Authenticate returns middleware that enforces the specified authentication requirement.
func (m *Middleware) Authenticate(requirement AuthRequirement) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Public endpoints skip all validation
			if requirement == AuthPublic {
				next.ServeHTTP(w, r)
				return
			}

			token := ExtractToken(r)

			if token == "" {
				if requirement == AuthRequired {
					writeJSONError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				// AuthOptional: proceed without user
				next.ServeHTTP(w, r)
				return
			}

			user, err := m.validator.ValidateToken(r.Context(), token)
			if err != nil {
				if requirement == AuthRequired {
					status := http.StatusUnauthorized
					message := "invalid token"

					switch err {
					case ErrExpiredToken:
						message = "token has expired"
					case ErrInvalidIssuer:
						message = "invalid token issuer"
					case ErrInvalidAudience:
						message = "invalid token audience"
					}

					writeJSONError(w, status, message)
					return
				}
				// AuthOptional: proceed without user
				next.ServeHTTP(w, r)
				return
			}

			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is a convenience method that creates middleware requiring authentication.
func (m *Middleware) RequireAuth() func(http.Handler) http.Handler {
	return m.Authenticate(AuthRequired)
}

// OptionalAuth is a convenience method that creates middleware with optional authentication.
func (m *Middleware) OptionalAuth() func(http.Handler) http.Handler {
	return m.Authenticate(AuthOptional)
}

// RequireRole returns middleware that checks for a specific role.
// Must be used after Authenticate middleware.
func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !user.HasRole(role) {
				writeJSONError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission returns middleware that checks for a specific permission.
// Must be used after Authenticate middleware.
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !user.HasPermission(permission) {
				writeJSONError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole returns middleware that checks for any of the specified roles.
// Must be used after Authenticate middleware.
func (m *Middleware) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, role := range roles {
				if user.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeJSONError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// ExtractToken extracts the JWT from a request.
// It checks the Authorization header (Bearer token), query parameter, and cookie.
func ExtractToken(r *http.Request) string {
	// Check Authorization header (Bearer token)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check query parameter (useful for WebSocket connections)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	// Check cookie
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}

	return ""
}

// errorResponse represents a JSON error response.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}
