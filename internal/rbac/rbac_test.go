package rbac

import (
	"context"
	"log/slog"
	"testing"

	"github.com/bargom/codeai/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermission_String(t *testing.T) {
	p := Permission("users:read")
	assert.Equal(t, "users:read", p.String())
}

func TestPermission_Resource(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected string
	}{
		{"simple", Permission("users:read"), "users"},
		{"wildcard", Permission("*:read"), "*"},
		{"empty", Permission(""), ""},
		{"no_colon", Permission("users"), "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.perm.Resource())
		})
	}
}

func TestPermission_Action(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected string
	}{
		{"simple", Permission("users:read"), "read"},
		{"wildcard", Permission("users:*"), "*"},
		{"empty", Permission(""), ""},
		{"no_colon", Permission("users"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.perm.Action())
		})
	}
}

func TestPermission_Matches(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		target   Permission
		expected bool
	}{
		{"exact_match", Permission("users:read"), Permission("users:read"), true},
		{"no_match_resource", Permission("users:read"), Permission("configs:read"), false},
		{"no_match_action", Permission("users:read"), Permission("users:write"), false},
		{"wildcard_resource", Permission("*:read"), Permission("users:read"), true},
		{"wildcard_action", Permission("users:*"), Permission("users:write"), true},
		{"full_wildcard", Permission("*:*"), Permission("anything:anywhere"), true},
		{"wildcard_no_match", Permission("*:read"), Permission("users:write"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.perm.Matches(tt.target))
		})
	}
}

func TestNewEngine(t *testing.T) {
	storage := NewMemoryStorage()
	engine := NewEngine(storage)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.storage)
	assert.NotNil(t, engine.cache)
	assert.NotNil(t, engine.logger)
}

func TestNewEngineWithLogger(t *testing.T) {
	storage := NewMemoryStorage()
	logger := slog.Default()
	engine := NewEngineWithLogger(storage, logger)

	assert.NotNil(t, engine)

	// Test with nil logger
	engine2 := NewEngineWithLogger(storage, nil)
	assert.NotNil(t, engine2)
}

func TestEngine_CheckPermission(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	tests := []struct {
		name       string
		user       *auth.User
		permission Permission
		expected   bool
	}{
		{
			name:       "nil_user",
			user:       nil,
			permission: Permission("users:read"),
			expected:   false,
		},
		{
			name: "direct_permission",
			user: &auth.User{
				ID:          "user1",
				Permissions: []string{"users:read"},
			},
			permission: Permission("users:read"),
			expected:   true,
		},
		{
			name: "wildcard_permission_in_claims",
			user: &auth.User{
				ID:          "user1",
				Permissions: []string{"*:read"},
			},
			permission: Permission("users:read"),
			expected:   true,
		},
		{
			name: "role_permission_admin",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			permission: Permission("anything:anything"),
			expected:   true,
		},
		{
			name: "role_permission_viewer",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			permission: Permission("configs:read"),
			expected:   true,
		},
		{
			name: "role_permission_viewer_denied",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			permission: Permission("configs:delete"),
			expected:   false,
		},
		{
			name: "inherited_permission_editor",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			permission: Permission("health:read"), // inherited from viewer
			expected:   true,
		},
		{
			name: "no_matching_permission",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"viewer"},
			},
			permission: Permission("admin:superpower"),
			expected:   false,
		},
		{
			name: "unknown_role",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"nonexistent"},
			},
			permission: Permission("configs:read"),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CheckPermission(ctx, tt.user, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_CheckRole(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	tests := []struct {
		name     string
		user     *auth.User
		role     string
		expected bool
	}{
		{
			name:     "nil_user",
			user:     nil,
			role:     "admin",
			expected: false,
		},
		{
			name: "direct_role",
			user: &auth.User{
				ID:    "admin1",
				Roles: []string{"admin"},
			},
			role:     "admin",
			expected: true,
		},
		{
			name: "inherited_role",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			role:     "viewer", // editor inherits from viewer
			expected: true,
		},
		{
			name: "no_matching_role",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"viewer"},
			},
			role:     "admin",
			expected: false,
		},
		{
			name: "unknown_role_in_user",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"nonexistent"},
			},
			role:     "admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CheckRole(ctx, tt.user, tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_GetUserPermissions(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	tests := []struct {
		name     string
		user     *auth.User
		minPerms int
	}{
		{
			name:     "nil_user",
			user:     nil,
			minPerms: 0,
		},
		{
			name: "user_with_direct_perms",
			user: &auth.User{
				ID:          "user1",
				Permissions: []string{"custom:perm"},
			},
			minPerms: 1,
		},
		{
			name: "viewer_role",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			minPerms: 4, // configs:read, executions:read, deployments:read, health:read
		},
		{
			name: "editor_role_with_inheritance",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			minPerms: 8, // editor perms + inherited viewer perms
		},
		{
			name: "combined_direct_and_role",
			user: &auth.User{
				ID:          "user1",
				Permissions: []string{"custom:perm"},
				Roles:       []string{"viewer"},
			},
			minPerms: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := engine.GetUserPermissions(ctx, tt.user)
			assert.GreaterOrEqual(t, len(perms), tt.minPerms)
		})
	}
}

func TestEngine_GetUserRoles(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	tests := []struct {
		name         string
		user         *auth.User
		expectedLen  int
		containsRole string
	}{
		{
			name:        "nil_user",
			user:        nil,
			expectedLen: 0,
		},
		{
			name: "viewer_only",
			user: &auth.User{
				ID:    "viewer1",
				Roles: []string{"viewer"},
			},
			expectedLen:  1,
			containsRole: "viewer",
		},
		{
			name: "editor_with_inheritance",
			user: &auth.User{
				ID:    "editor1",
				Roles: []string{"editor"},
			},
			expectedLen:  2, // editor + inherited viewer
			containsRole: "viewer",
		},
		{
			name: "unknown_role",
			user: &auth.User{
				ID:    "user1",
				Roles: []string{"nonexistent"},
			},
			expectedLen:  1,
			containsRole: "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := engine.GetUserRoles(ctx, tt.user)
			assert.Len(t, roles, tt.expectedLen)
			if tt.containsRole != "" {
				assert.Contains(t, roles, tt.containsRole)
			}
		})
	}
}

func TestEngine_InvalidateCache(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	// Populate cache
	user := &auth.User{
		ID:    "viewer1",
		Roles: []string{"viewer"},
	}
	engine.CheckPermission(ctx, user, Permission("configs:read"))

	// Verify cache is populated
	_, cached := engine.cache.get("viewer")
	assert.True(t, cached)

	// Invalidate
	engine.InvalidateCache()

	// Verify cache is cleared
	_, cached = engine.cache.get("viewer")
	assert.False(t, cached)
}

func TestPermissionCache(t *testing.T) {
	cache := newPermissionCache()

	// Test get on empty cache
	_, ok := cache.get("nonexistent")
	assert.False(t, ok)

	// Test set and get
	perms := []Permission{"users:read", "users:write"}
	cache.set("test", perms)

	cached, ok := cache.get("test")
	assert.True(t, ok)
	assert.Equal(t, perms, cached)

	// Test clear
	cache.clear()
	_, ok = cache.get("test")
	assert.False(t, ok)
}

func TestEngine_CyclicInheritanceHandling(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Create roles with what would be a cycle if we allowed it
	role1 := &Role{
		Name:        "role1",
		Permissions: []Permission{"perm1:read"},
		Parents:     []string{"role2"},
	}
	role2 := &Role{
		Name:        "role2",
		Permissions: []Permission{"perm2:read"},
		Parents:     []string{"role1"}, // This would create a cycle
	}

	require.NoError(t, storage.SaveRole(ctx, role1))
	require.NoError(t, storage.SaveRole(ctx, role2))

	engine := NewEngine(storage)

	// The engine should handle this without infinite loop
	user := &auth.User{
		ID:    "user1",
		Roles: []string{"role1"},
	}

	// Should complete without hanging
	perms := engine.GetUserPermissions(ctx, user)
	assert.NotNil(t, perms)

	// Should find permissions from both roles
	hasP1 := false
	hasP2 := false
	for _, p := range perms {
		if p == Permission("perm1:read") {
			hasP1 = true
		}
		if p == Permission("perm2:read") {
			hasP2 = true
		}
	}
	assert.True(t, hasP1, "should have perm1:read")
	assert.True(t, hasP2, "should have perm2:read")
}

func TestEngine_DeepInheritance(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Create a deep inheritance chain
	role1 := &Role{
		Name:        "level1",
		Permissions: []Permission{"level1:perm"},
		Parents:     []string{},
	}
	role2 := &Role{
		Name:        "level2",
		Permissions: []Permission{"level2:perm"},
		Parents:     []string{"level1"},
	}
	role3 := &Role{
		Name:        "level3",
		Permissions: []Permission{"level3:perm"},
		Parents:     []string{"level2"},
	}
	role4 := &Role{
		Name:        "level4",
		Permissions: []Permission{"level4:perm"},
		Parents:     []string{"level3"},
	}

	require.NoError(t, storage.SaveRole(ctx, role1))
	require.NoError(t, storage.SaveRole(ctx, role2))
	require.NoError(t, storage.SaveRole(ctx, role3))
	require.NoError(t, storage.SaveRole(ctx, role4))

	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{"level4"},
	}

	// Should have all permissions from the chain
	assert.True(t, engine.CheckPermission(ctx, user, Permission("level1:perm")))
	assert.True(t, engine.CheckPermission(ctx, user, Permission("level2:perm")))
	assert.True(t, engine.CheckPermission(ctx, user, Permission("level3:perm")))
	assert.True(t, engine.CheckPermission(ctx, user, Permission("level4:perm")))

	// Should have all roles
	roles := engine.GetUserRoles(ctx, user)
	assert.Len(t, roles, 4)
	assert.Contains(t, roles, "level1")
	assert.Contains(t, roles, "level2")
	assert.Contains(t, roles, "level3")
	assert.Contains(t, roles, "level4")
}

func TestEngine_MultipleRoles(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	roleA := &Role{
		Name:        "roleA",
		Permissions: []Permission{"a:read", "a:write"},
		Parents:     []string{},
	}
	roleB := &Role{
		Name:        "roleB",
		Permissions: []Permission{"b:read", "b:write"},
		Parents:     []string{},
	}

	require.NoError(t, storage.SaveRole(ctx, roleA))
	require.NoError(t, storage.SaveRole(ctx, roleB))

	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{"roleA", "roleB"},
	}

	// Should have permissions from both roles
	assert.True(t, engine.CheckPermission(ctx, user, Permission("a:read")))
	assert.True(t, engine.CheckPermission(ctx, user, Permission("b:write")))

	perms := engine.GetUserPermissions(ctx, user)
	assert.Len(t, perms, 4)
}

func TestEngine_PermissionDeduplication(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	role1 := &Role{
		Name:        "role1",
		Permissions: []Permission{"shared:perm", "unique1:perm"},
		Parents:     []string{},
	}
	role2 := &Role{
		Name:        "role2",
		Permissions: []Permission{"shared:perm", "unique2:perm"},
		Parents:     []string{},
	}

	require.NoError(t, storage.SaveRole(ctx, role1))
	require.NoError(t, storage.SaveRole(ctx, role2))

	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{"role1", "role2"},
	}

	perms := engine.GetUserPermissions(ctx, user)
	// Should deduplicate: shared:perm, unique1:perm, unique2:perm = 3
	assert.Len(t, perms, 3)
}

func TestEngine_HasInheritedRole_NilRole(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()
	engine := NewEngine(storage)

	// Test with unknown role - storage returns error
	user := &auth.User{
		ID:    "user1",
		Roles: []string{"unknown"},
	}

	// Should return false without panic
	result := engine.CheckRole(ctx, user, "target")
	assert.False(t, result)
}

func TestEngine_GetUserRoles_WithErrorFromStorage(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Add a role with a parent that doesn't exist
	role := &Role{
		Name:        "child",
		Permissions: []Permission{"child:perm"},
		Parents:     []string{"nonexistent_parent"},
	}
	require.NoError(t, storage.SaveRole(ctx, role))

	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{"child"},
	}

	// Should handle missing parent gracefully
	roles := engine.GetUserRoles(ctx, user)
	assert.Contains(t, roles, "child")
}

func TestPermission_Resource_Empty(t *testing.T) {
	// Test with completely empty permission
	p := Permission("")
	assert.Equal(t, "", p.Resource())
	assert.Equal(t, "", p.Action())
}

func TestPermission_Resource_SinglePart(t *testing.T) {
	// Test with single part (no colon)
	p := Permission("onlyresource")
	assert.Equal(t, "onlyresource", p.Resource())
	assert.Equal(t, "", p.Action())
}

func TestEngine_CheckRole_DeepInheritance(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Create a deeper inheritance chain: role4 -> role3 -> role2 -> role1
	role1 := &Role{Name: "role1", Permissions: []Permission{"r:a"}}
	role2 := &Role{Name: "role2", Permissions: []Permission{"r:a"}, Parents: []string{"role1"}}
	role3 := &Role{Name: "role3", Permissions: []Permission{"r:a"}, Parents: []string{"role2"}}
	role4 := &Role{Name: "role4", Permissions: []Permission{"r:a"}, Parents: []string{"role3"}}

	require.NoError(t, storage.SaveRole(ctx, role1))
	require.NoError(t, storage.SaveRole(ctx, role2))
	require.NoError(t, storage.SaveRole(ctx, role3))
	require.NoError(t, storage.SaveRole(ctx, role4))

	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{"role4"},
	}

	// Should find deeply inherited role through recursive hasInheritedRole
	assert.True(t, engine.CheckRole(ctx, user, "role1"))
	assert.True(t, engine.CheckRole(ctx, user, "role2"))
	assert.True(t, engine.CheckRole(ctx, user, "role3"))
	assert.True(t, engine.CheckRole(ctx, user, "role4"))
}

func TestEngine_GetUserRoles_NoRolesInUser(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorageWithDefaults()
	engine := NewEngine(storage)

	user := &auth.User{
		ID:    "user1",
		Roles: []string{}, // Empty roles
	}

	roles := engine.GetUserRoles(ctx, user)
	assert.Empty(t, roles)
}
