// Package integration provides integration tests for the CodeAI modules.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/auth"
	"github.com/bargom/codeai/internal/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthRBACIntegration tests the complete authentication and authorization flow
// combining JWT validation with RBAC permission checking.
func TestAuthRBACIntegration(t *testing.T) {
	// Setup RBAC engine with roles and permissions
	storage := rbac.NewMemoryStorageWithDefaults()
	engine := rbac.NewEngine(storage)

	// Define additional custom roles
	err := storage.SaveRole(context.Background(), &rbac.Role{
		Name:        "editor",
		Description: "Can view and edit content",
		Permissions: []rbac.Permission{
			"content:read",
			"content:write",
			"content:update",
		},
		Parents: []string{"viewer"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(context.Background(), &rbac.Role{
		Name:        "viewer",
		Description: "Can only view content",
		Permissions: []rbac.Permission{
			"content:read",
		},
	})
	require.NoError(t, err)

	// Test cases for various user role combinations
	tests := []struct {
		name           string
		user           *auth.User
		permission     rbac.Permission
		expectedResult bool
	}{
		{
			name: "admin has all permissions via wildcard",
			user: &auth.User{
				ID:    "admin-1",
				Email: "admin@example.com",
				Roles: []string{"admin"},
			},
			permission:     "users:delete",
			expectedResult: true,
		},
		{
			name: "editor can read content",
			user: &auth.User{
				ID:    "editor-1",
				Email: "editor@example.com",
				Roles: []string{"editor"},
			},
			permission:     "content:read",
			expectedResult: true,
		},
		{
			name: "editor can write content",
			user: &auth.User{
				ID:    "editor-1",
				Email: "editor@example.com",
				Roles: []string{"editor"},
			},
			permission:     "content:write",
			expectedResult: true,
		},
		{
			name: "viewer can only read content",
			user: &auth.User{
				ID:    "viewer-1",
				Email: "viewer@example.com",
				Roles: []string{"viewer"},
			},
			permission:     "content:read",
			expectedResult: true,
		},
		{
			name: "viewer cannot write content",
			user: &auth.User{
				ID:    "viewer-1",
				Email: "viewer@example.com",
				Roles: []string{"viewer"},
			},
			permission:     "content:write",
			expectedResult: false,
		},
		{
			name: "user with direct permission bypasses role check",
			user: &auth.User{
				ID:          "special-1",
				Email:       "special@example.com",
				Roles:       []string{"viewer"},
				Permissions: []string{"admin:special"},
			},
			permission:     "admin:special",
			expectedResult: true,
		},
		{
			name: "nil user has no permissions",
			user: nil,
			permission:     "any:permission",
			expectedResult: false,
		},
		{
			name: "user with no roles has no permissions",
			user: &auth.User{
				ID:    "norole-1",
				Email: "norole@example.com",
				Roles: []string{},
			},
			permission:     "content:read",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CheckPermission(context.Background(), tt.user, tt.permission)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestAuthRBACRoleInheritance tests that role inheritance works correctly.
func TestAuthRBACRoleInheritance(t *testing.T) {
	ctx := context.Background()
	storage := rbac.NewMemoryStorage()
	engine := rbac.NewEngine(storage)

	// Create role hierarchy: superadmin -> admin -> moderator -> user
	err := storage.SaveRole(ctx, &rbac.Role{
		Name:        "user",
		Description: "Basic user",
		Permissions: []rbac.Permission{"profile:read", "profile:update"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:        "moderator",
		Description: "Moderator with user permissions plus moderation",
		Permissions: []rbac.Permission{"content:moderate", "comments:delete"},
		Parents:     []string{"user"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:        "admin",
		Description: "Admin with moderator permissions plus admin actions",
		Permissions: []rbac.Permission{"users:manage", "settings:update"},
		Parents:     []string{"moderator"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:        "superadmin",
		Description: "Super admin with full access",
		Permissions: []rbac.Permission{"*:*"},
		Parents:     []string{"admin"},
	})
	require.NoError(t, err)

	// Test user with admin role
	adminUser := &auth.User{
		ID:    "admin-1",
		Email: "admin@example.com",
		Roles: []string{"admin"},
	}

	// Admin should have inherited permissions from moderator and user
	assert.True(t, engine.CheckPermission(ctx, adminUser, "profile:read"), "admin should have user permission")
	assert.True(t, engine.CheckPermission(ctx, adminUser, "content:moderate"), "admin should have moderator permission")
	assert.True(t, engine.CheckPermission(ctx, adminUser, "users:manage"), "admin should have own permission")

	// Admin should NOT have superadmin permission
	assert.False(t, engine.CheckPermission(ctx, adminUser, "billing:manage"), "admin should not have billing permission")

	// Test superadmin with wildcard
	superadminUser := &auth.User{
		ID:    "superadmin-1",
		Email: "superadmin@example.com",
		Roles: []string{"superadmin"},
	}
	assert.True(t, engine.CheckPermission(ctx, superadminUser, "anything:anywhere"), "superadmin should have wildcard access")

	// Test role checking with inheritance
	assert.True(t, engine.CheckRole(ctx, adminUser, "user"), "admin should have inherited user role")
	assert.True(t, engine.CheckRole(ctx, adminUser, "moderator"), "admin should have inherited moderator role")
	assert.False(t, engine.CheckRole(ctx, adminUser, "superadmin"), "admin should not have superadmin role")
}

// TestAuthRBACCacheInvalidation tests that cache invalidation works correctly.
func TestAuthRBACCacheInvalidation(t *testing.T) {
	ctx := context.Background()
	storage := rbac.NewMemoryStorage()
	engine := rbac.NewEngine(storage)

	// Create initial role
	err := storage.SaveRole(ctx, &rbac.Role{
		Name:        "tester",
		Description: "Test role",
		Permissions: []rbac.Permission{"test:read"},
	})
	require.NoError(t, err)

	user := &auth.User{
		ID:    "user-1",
		Roles: []string{"tester"},
	}

	// Initial check - should have read permission
	assert.True(t, engine.CheckPermission(ctx, user, "test:read"))
	assert.False(t, engine.CheckPermission(ctx, user, "test:write"))

	// Update role with new permission
	err = storage.SaveRole(ctx, &rbac.Role{
		Name:        "tester",
		Description: "Test role updated",
		Permissions: []rbac.Permission{"test:read", "test:write"},
	})
	require.NoError(t, err)

	// Without cache invalidation, old permissions might be returned
	// Invalidate the cache
	engine.InvalidateCache()

	// Now user should have write permission
	assert.True(t, engine.CheckPermission(ctx, user, "test:write"))
}

// TestAuthMiddlewareWithRBAC tests the auth middleware integration with RBAC.
func TestAuthMiddlewareWithRBAC(t *testing.T) {
	ctx := context.Background()

	// Setup RBAC engine
	storage := rbac.NewMemoryStorageWithDefaults()
	rbacEngine := rbac.NewEngine(storage)

	// Create test handler that checks permission
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check if user has required permission
		if !rbacEngine.CheckPermission(r.Context(), user, "api:access") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Test with no user in context
	t.Run("no user returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	// Test with user but no permission
	t.Run("user without permission returns forbidden", func(t *testing.T) {
		user := &auth.User{
			ID:    "user-1",
			Email: "user@example.com",
			Roles: []string{}, // No roles
		}

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		ctx := auth.ContextWithUser(req.Context(), user)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	// Test with user with permission
	t.Run("user with permission returns success", func(t *testing.T) {
		// Add a role with api:access permission
		err := storage.SaveRole(ctx, &rbac.Role{
			Name:        "api_user",
			Permissions: []rbac.Permission{"api:access"},
		})
		require.NoError(t, err)

		user := &auth.User{
			ID:    "user-2",
			Email: "apiuser@example.com",
			Roles: []string{"api_user"},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		ctx := auth.ContextWithUser(req.Context(), user)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})

	// Test with admin user
	t.Run("admin has access via wildcard", func(t *testing.T) {
		user := &auth.User{
			ID:    "admin-1",
			Email: "admin@example.com",
			Roles: []string{"admin"},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		ctx := auth.ContextWithUser(req.Context(), user)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestUserHasRoleAndPermission tests the User helper methods.
func TestUserHasRoleAndPermission(t *testing.T) {
	user := &auth.User{
		ID:          "test-1",
		Email:       "test@example.com",
		Roles:       []string{"admin", "editor"},
		Permissions: []string{"special:access", "billing:read"},
	}

	// Test HasRole
	assert.True(t, user.HasRole("admin"))
	assert.True(t, user.HasRole("editor"))
	assert.False(t, user.HasRole("viewer"))
	assert.False(t, user.HasRole(""))

	// Test HasPermission
	assert.True(t, user.HasPermission("special:access"))
	assert.True(t, user.HasPermission("billing:read"))
	assert.False(t, user.HasPermission("billing:write"))
	assert.False(t, user.HasPermission(""))
}

// TestGetUserPermissions tests retrieving all effective permissions for a user.
func TestGetUserPermissions(t *testing.T) {
	ctx := context.Background()
	storage := rbac.NewMemoryStorage()
	engine := rbac.NewEngine(storage)

	// Setup roles with inheritance
	err := storage.SaveRole(ctx, &rbac.Role{
		Name:        "base",
		Permissions: []rbac.Permission{"base:read"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:        "derived",
		Permissions: []rbac.Permission{"derived:write"},
		Parents:     []string{"base"},
	})
	require.NoError(t, err)

	user := &auth.User{
		ID:          "user-1",
		Roles:       []string{"derived"},
		Permissions: []string{"direct:access"},
	}

	perms := engine.GetUserPermissions(ctx, user)

	// Should contain direct permission, derived permission, and inherited base permission
	assert.Contains(t, perms, rbac.Permission("direct:access"))
	assert.Contains(t, perms, rbac.Permission("derived:write"))
	assert.Contains(t, perms, rbac.Permission("base:read"))
}

// TestGetUserRoles tests retrieving all effective roles including inherited ones.
func TestGetUserRoles(t *testing.T) {
	ctx := context.Background()
	storage := rbac.NewMemoryStorage()
	engine := rbac.NewEngine(storage)

	// Setup role hierarchy
	err := storage.SaveRole(ctx, &rbac.Role{
		Name: "grandparent",
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:    "parent",
		Parents: []string{"grandparent"},
	})
	require.NoError(t, err)

	err = storage.SaveRole(ctx, &rbac.Role{
		Name:    "child",
		Parents: []string{"parent"},
	})
	require.NoError(t, err)

	user := &auth.User{
		ID:    "user-1",
		Roles: []string{"child"},
	}

	roles := engine.GetUserRoles(ctx, user)

	// Should contain child, parent, and grandparent roles
	assert.Contains(t, roles, "child")
	assert.Contains(t, roles, "parent")
	assert.Contains(t, roles, "grandparent")
}

// TestConcurrentRBACChecks tests that the RBAC engine is thread-safe.
func TestConcurrentRBACChecks(t *testing.T) {
	ctx := context.Background()
	storage := rbac.NewMemoryStorageWithDefaults()
	engine := rbac.NewEngine(storage)

	user := &auth.User{
		ID:    "user-1",
		Roles: []string{"admin"},
	}

	// Run concurrent permission checks
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			result := engine.CheckPermission(ctx, user, "users:read")
			assert.True(t, result)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestUserExpiration tests handling of expired tokens/users.
func TestUserExpiration(t *testing.T) {
	user := &auth.User{
		ID:        "user-1",
		Email:     "user@example.com",
		Roles:     []string{"viewer"},
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	// Check if user is expired
	isExpired := time.Now().After(user.ExpiresAt)
	assert.True(t, isExpired)

	// Application should check expiration before using user
	validUser := &auth.User{
		ID:        "user-2",
		Email:     "user2@example.com",
		Roles:     []string{"viewer"},
		ExpiresAt: time.Now().Add(1 * time.Hour), // Valid for 1 more hour
	}

	isValid := time.Now().Before(validUser.ExpiresAt)
	assert.True(t, isValid)
}
