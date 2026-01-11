package rbac

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStorage(t *testing.T) {
	s := NewMemoryStorage()

	assert.NotNil(t, s)
	assert.NotNil(t, s.roles)
	assert.Empty(t, s.roles)
}

func TestNewMemoryStorageWithDefaults(t *testing.T) {
	s := NewMemoryStorageWithDefaults()

	assert.NotNil(t, s)
	assert.Len(t, s.roles, 3) // admin, editor, viewer

	// Verify roles are present
	assert.Contains(t, s.roles, "admin")
	assert.Contains(t, s.roles, "editor")
	assert.Contains(t, s.roles, "viewer")
}

func TestMemoryStorage_GetRole(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorageWithDefaults()

	role, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "admin", role.Name)
}

func TestMemoryStorage_GetRole_NotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role, err := s.GetRole(ctx, "nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
	assert.Nil(t, role)
}

func TestMemoryStorage_GetRole_ReturnsCopy(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorageWithDefaults()

	role1, _ := s.GetRole(ctx, "admin")
	role2, _ := s.GetRole(ctx, "admin")

	// Modifying one shouldn't affect the other
	role1.Description = "Modified"
	assert.NotEqual(t, role1.Description, role2.Description)
}

func TestMemoryStorage_ListRoles(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorageWithDefaults()

	roles, err := s.ListRoles(ctx)
	require.NoError(t, err)
	assert.Len(t, roles, 3)

	names := make([]string, 0, 3)
	for _, r := range roles {
		names = append(names, r.Name)
	}
	assert.Contains(t, names, "admin")
	assert.Contains(t, names, "editor")
	assert.Contains(t, names, "viewer")
}

func TestMemoryStorage_ListRoles_Empty(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	roles, err := s.ListRoles(ctx)
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestMemoryStorage_SaveRole(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role := &Role{
		Name:        "custom",
		Description: "Custom role",
		Permissions: []Permission{"resource:action"},
	}

	err := s.SaveRole(ctx, role)
	require.NoError(t, err)

	saved, err := s.GetRole(ctx, "custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", saved.Name)
	assert.Equal(t, "Custom role", saved.Description)
}

func TestMemoryStorage_SaveRole_Update(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role := &Role{
		Name:        "custom",
		Description: "Initial",
		Permissions: []Permission{"r:a"},
	}
	require.NoError(t, s.SaveRole(ctx, role))

	// Update the role
	role.Description = "Updated"
	role.Permissions = append(role.Permissions, Permission("r:b"))
	require.NoError(t, s.SaveRole(ctx, role))

	saved, _ := s.GetRole(ctx, "custom")
	assert.Equal(t, "Updated", saved.Description)
	assert.Len(t, saved.Permissions, 2)
}

func TestMemoryStorage_SaveRole_InvalidName(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role := &Role{Name: "", Permissions: []Permission{"r:a"}}
	err := s.SaveRole(ctx, role)
	assert.Equal(t, ErrInvalidRoleName, err)
}

func TestMemoryStorage_SaveRole_InvalidPermission(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role := &Role{Name: "test", Permissions: []Permission{"invalid"}}
	err := s.SaveRole(ctx, role)
	assert.Equal(t, ErrInvalidPermission, err)
}

func TestMemoryStorage_SaveRole_StoresCopy(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	role := &Role{
		Name:        "test",
		Description: "Original",
		Permissions: []Permission{"r:a"},
	}
	require.NoError(t, s.SaveRole(ctx, role))

	// Modify the original
	role.Description = "Modified"

	// Retrieved should be unchanged
	saved, _ := s.GetRole(ctx, "test")
	assert.Equal(t, "Original", saved.Description)
}

func TestMemoryStorage_DeleteRole(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorageWithDefaults()

	err := s.DeleteRole(ctx, "admin")
	require.NoError(t, err)

	_, err = s.GetRole(ctx, "admin")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestMemoryStorage_DeleteRole_NotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	err := s.DeleteRole(ctx, "nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
}

// CachedStorage Tests

func TestNewCachedStorage(t *testing.T) {
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	assert.NotNil(t, s)
	assert.NotNil(t, s.cache)
	assert.Equal(t, 5*time.Minute, s.ttl)
	assert.Equal(t, 1*time.Minute, s.listTTL)
}

func TestNewCachedStorage_WithOptions(t *testing.T) {
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend,
		WithTTL(10*time.Minute),
		WithListTTL(2*time.Minute),
	)

	assert.Equal(t, 10*time.Minute, s.ttl)
	assert.Equal(t, 2*time.Minute, s.listTTL)
}

func TestCachedStorage_GetRole(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	// First call should hit backend
	role, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)

	// Second call should hit cache
	role2, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role2.Name)
}

func TestCachedStorage_GetRole_NotFound(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	_, err := s.GetRole(ctx, "nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestCachedStorage_GetRole_ReturnsCopy(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	role1, _ := s.GetRole(ctx, "admin")
	role2, _ := s.GetRole(ctx, "admin")

	// Modifying one shouldn't affect the other
	role1.Description = "Modified"
	assert.NotEqual(t, role1.Description, role2.Description)
}

func TestCachedStorage_GetRole_CacheExpiry(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend, WithTTL(1*time.Millisecond))

	// First call populates cache
	_, _ = s.GetRole(ctx, "admin")

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Should go back to backend
	role, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestCachedStorage_ListRoles(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	roles, err := s.ListRoles(ctx)
	require.NoError(t, err)
	assert.Len(t, roles, 3)
}

func TestCachedStorage_SaveRole(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	role := &Role{Name: "test", Permissions: []Permission{"r:a"}}
	err := s.SaveRole(ctx, role)
	require.NoError(t, err)

	// Should be available
	saved, err := s.GetRole(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "test", saved.Name)
}

func TestCachedStorage_SaveRole_InvalidatesCache(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	role := &Role{Name: "test", Description: "v1", Permissions: []Permission{"r:a"}}
	require.NoError(t, s.SaveRole(ctx, role))

	// Populate cache
	_, _ = s.GetRole(ctx, "test")

	// Update role
	role.Description = "v2"
	require.NoError(t, s.SaveRole(ctx, role))

	// Should get updated version
	saved, _ := s.GetRole(ctx, "test")
	assert.Equal(t, "v2", saved.Description)
}

func TestCachedStorage_DeleteRole(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	err := s.DeleteRole(ctx, "admin")
	require.NoError(t, err)

	_, err = s.GetRole(ctx, "admin")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestCachedStorage_DeleteRole_InvalidatesCache(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	// Populate cache
	_, _ = s.GetRole(ctx, "admin")

	// Delete role
	require.NoError(t, s.DeleteRole(ctx, "admin"))

	// Should not be found
	_, err := s.GetRole(ctx, "admin")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestCachedStorage_InvalidateCache(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	// Populate cache
	_, _ = s.GetRole(ctx, "admin")
	_, _ = s.GetRole(ctx, "editor")

	// Invalidate all
	s.InvalidateCache()

	// Cache should be empty but backend still accessible
	role, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestCachedStorage_InvalidateRole(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	// Populate cache
	_, _ = s.GetRole(ctx, "admin")
	_, _ = s.GetRole(ctx, "editor")

	// Invalidate only admin
	s.InvalidateRole("admin")

	// Admin cache entry should be gone, but still accessible from backend
	role, err := s.GetRole(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

// Concurrent access tests

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	// Start multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			role := &Role{
				Name:        "concurrent",
				Permissions: []Permission{"r:a"},
			}
			_ = s.SaveRole(ctx, role)
			_, _ = s.GetRole(ctx, "concurrent")
			_, _ = s.ListRoles(ctx)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCachedStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorageWithDefaults()
	s := NewCachedStorage(backend)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, _ = s.GetRole(ctx, "admin")
			_, _ = s.GetRole(ctx, "editor")
			_, _ = s.ListRoles(ctx)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCachedStorage_SaveRole_BackendError(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	// Try to save a role with invalid permission
	role := &Role{Name: "test", Permissions: []Permission{"invalid"}}
	err := s.SaveRole(ctx, role)
	assert.Equal(t, ErrInvalidPermission, err)
}

func TestCachedStorage_DeleteRole_BackendError(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryStorage()
	s := NewCachedStorage(backend)

	// Try to delete a nonexistent role
	err := s.DeleteRole(ctx, "nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
}
