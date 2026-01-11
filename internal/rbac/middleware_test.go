package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bargom/codeai/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a request with an authenticated user
func requestWithUser(user *auth.User) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if user != nil {
		ctx := auth.ContextWithUser(req.Context(), user)
		req = req.WithContext(ctx)
	}
	return req
}

// Helper to decode error response
func decodeError(t *testing.T, body []byte) string {
	var resp errorResponse
	err := json.Unmarshal(body, &resp)
	require.NoError(t, err)
	return resp.Error
}

func TestNewMiddleware(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	assert.NotNil(t, mw)
	assert.Equal(t, engine, mw.engine)
}

func TestMiddleware_RequirePermission(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequirePermission("configs:read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "viewer_has_permission",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "admin_has_permission",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "no_permission",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "direct_permission",
			user: &auth.User{
				ID:          "user1",
				Permissions: []string{"configs:read"},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_RequireAnyPermission(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequireAnyPermission("configs:create", "configs:delete")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "has_first_permission",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "viewer_denied",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "admin_allowed",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_RequireAllPermissions(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequireAllPermissions("configs:read", "configs:create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "has_both_permissions",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "viewer_missing_create",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "admin_has_all",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_RequireRole(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "is_admin",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_admin",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_RequireRole_WithInheritance(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	// Editor inherits from viewer
	handler := mw.RequireRole("viewer")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name: "direct_viewer",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "editor_inherits_viewer",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_RequireAnyRole(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequireAnyRole("admin", "editor")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "is_admin",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "is_editor",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "is_viewer",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_ResourcePermission(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	// Extract resource from URL path
	resourceExtractor := func(r *http.Request) string {
		return "configs" // simplified for test
	}

	handler := mw.ResourcePermission("read", resourceExtractor)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "viewer_can_read",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "no_permission",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{},
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMiddleware_Custom(t *testing.T) {
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	// Custom authorization: user ID must start with "allowed-"
	authorize := func(r *http.Request, user *auth.User) bool {
		return len(user.ID) >= 8 && user.ID[:8] == "allowed-"
	}

	handler := mw.Custom(authorize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
	}{
		{
			name:       "no_user",
			user:       nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "allowed_user",
			user: &auth.User{
				ID: "allowed-user1",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "denied_user",
			user: &auth.User{
				ID: "denied-user1",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "short_id",
			user: &auth.User{
				ID: "short",
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithUser(tt.user)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			message:    "authentication required",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			status:     http.StatusForbidden,
			message:    "insufficient permissions",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeError(rr, tt.status, tt.message)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			errMsg := decodeError(t, rr.Body.Bytes())
			assert.Equal(t, tt.message, errMsg)
		})
	}
}

func TestMiddleware_ChainWithAuthMiddleware(t *testing.T) {
	// Test that RBAC middleware works correctly when chained with auth middleware
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)
	rbacMw := NewMiddleware(engine)

	// Simulate auth middleware by wrapping handler with context
	authWrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate authenticated user
			user := &auth.User{
				ID:    "user1",
				Roles: []string{"editor"},
			}
			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Chain: auth -> rbac -> handler
	handler := authWrapper(
		rbacMw.RequirePermission("configs:read")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/configs", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

func TestMiddleware_ErrorResponseFormat(t *testing.T) {
	storage := NewMemoryStorage()
	engine := NewEngine(storage)
	mw := NewMiddleware(engine)

	handler := mw.RequirePermission("configs:read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test unauthorized response
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var errResp errorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "authentication required", errResp.Error)

	// Test forbidden response
	user := &auth.User{ID: "user1", Roles: []string{}}
	ctx := auth.ContextWithUser(context.Background(), user)
	req = httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	err = json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "insufficient permissions", errResp.Error)
}
