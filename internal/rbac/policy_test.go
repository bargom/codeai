package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicy(t *testing.T) {
	p := NewPolicy()

	assert.NotNil(t, p)
	assert.NotNil(t, p.Roles)
	assert.Empty(t, p.Roles)
	assert.Equal(t, "1.0", p.Version)
}

func TestNewDefaultPolicy(t *testing.T) {
	p := NewDefaultPolicy()

	assert.NotNil(t, p)
	assert.Len(t, p.Roles, 3)

	// Check admin role
	admin, ok := p.Roles["admin"]
	assert.True(t, ok)
	assert.Equal(t, "admin", admin.Name)
	assert.Contains(t, admin.Permissions, Permission("*:*"))

	// Check editor role
	editor, ok := p.Roles["editor"]
	assert.True(t, ok)
	assert.Equal(t, "editor", editor.Name)
	assert.Contains(t, editor.Parents, "viewer")

	// Check viewer role
	viewer, ok := p.Roles["viewer"]
	assert.True(t, ok)
	assert.Equal(t, "viewer", viewer.Name)
	assert.Empty(t, viewer.Parents)
}

func TestPolicy_AddRole(t *testing.T) {
	p := NewPolicy()

	role := Role{
		Name:        "custom",
		Description: "Custom role",
		Permissions: []Permission{"resource:action"},
		Parents:     []string{},
	}

	err := p.AddRole(role)
	require.NoError(t, err)

	added, ok := p.Roles["custom"]
	assert.True(t, ok)
	assert.Equal(t, "custom", added.Name)
}

func TestPolicy_AddRole_AlreadyExists(t *testing.T) {
	p := NewPolicy()

	role := Role{
		Name:        "existing",
		Permissions: []Permission{"r:a"},
	}

	err := p.AddRole(role)
	require.NoError(t, err)

	err = p.AddRole(role)
	assert.Equal(t, ErrRoleAlreadyExists, err)
}

func TestPolicy_AddRole_InvalidName(t *testing.T) {
	p := NewPolicy()

	tests := []struct {
		name string
		role Role
	}{
		{
			name: "empty_name",
			role: Role{Name: "", Permissions: []Permission{"r:a"}},
		},
		{
			name: "whitespace_name",
			role: Role{Name: "role name", Permissions: []Permission{"r:a"}},
		},
		{
			name: "tab_name",
			role: Role{Name: "role\tname", Permissions: []Permission{"r:a"}},
		},
		{
			name: "newline_name",
			role: Role{Name: "role\nname", Permissions: []Permission{"r:a"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.AddRole(tt.role)
			assert.Equal(t, ErrInvalidRoleName, err)
		})
	}
}

func TestPolicy_AddRole_InvalidPermission(t *testing.T) {
	p := NewPolicy()

	tests := []struct {
		name string
		perm Permission
	}{
		{"empty", Permission("")},
		{"no_colon", Permission("resource")},
		{"empty_resource", Permission(":action")},
		{"empty_action", Permission("resource:")},
		{"too_many_colons", Permission("a:b:c")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := Role{
				Name:        "test",
				Permissions: []Permission{tt.perm},
			}
			err := p.AddRole(role)
			assert.Equal(t, ErrInvalidPermission, err)
		})
	}
}

func TestPolicy_AddRole_CyclicInheritance(t *testing.T) {
	p := NewPolicy()

	// Add base role
	base := Role{Name: "base", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(base))

	// Add middle role that inherits from base
	middle := Role{Name: "middle", Permissions: []Permission{"r:a"}, Parents: []string{"base"}}
	require.NoError(t, p.AddRole(middle))

	// Try to make base inherit from middle (creating a cycle)
	cyclic := Role{Name: "base", Permissions: []Permission{"r:a"}, Parents: []string{"middle"}}

	// Can't use AddRole because it already exists, use UpdateRole
	err := p.UpdateRole(cyclic)
	assert.Equal(t, ErrCyclicInheritance, err)
}

func TestPolicy_AddRole_SelfInheritance(t *testing.T) {
	p := NewPolicy()

	role := Role{
		Name:        "self",
		Permissions: []Permission{"r:a"},
		Parents:     []string{"self"},
	}

	err := p.AddRole(role)
	assert.Equal(t, ErrCyclicInheritance, err)
}

func TestPolicy_UpdateRole(t *testing.T) {
	p := NewPolicy()

	// Add initial role
	initial := Role{
		Name:        "test",
		Description: "Initial",
		Permissions: []Permission{"r:a"},
	}
	require.NoError(t, p.AddRole(initial))

	// Update the role
	updated := Role{
		Name:        "test",
		Description: "Updated",
		Permissions: []Permission{"r:a", "r:b"},
	}
	err := p.UpdateRole(updated)
	require.NoError(t, err)

	role, ok := p.Roles["test"]
	assert.True(t, ok)
	assert.Equal(t, "Updated", role.Description)
	assert.Len(t, role.Permissions, 2)
}

func TestPolicy_UpdateRole_NotFound(t *testing.T) {
	p := NewPolicy()

	role := Role{Name: "nonexistent", Permissions: []Permission{"r:a"}}
	err := p.UpdateRole(role)
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestPolicy_UpdateRole_InvalidName(t *testing.T) {
	p := NewPolicy()

	role := Role{Name: "", Permissions: []Permission{"r:a"}}
	err := p.UpdateRole(role)
	assert.Equal(t, ErrInvalidRoleName, err)
}

func TestPolicy_UpdateRole_InvalidPermission(t *testing.T) {
	p := NewPolicy()

	initial := Role{Name: "test", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(initial))

	updated := Role{Name: "test", Permissions: []Permission{"invalid"}}
	err := p.UpdateRole(updated)
	assert.Equal(t, ErrInvalidPermission, err)
}

func TestPolicy_RemoveRole(t *testing.T) {
	p := NewPolicy()

	role := Role{Name: "test", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(role))

	err := p.RemoveRole("test")
	require.NoError(t, err)

	_, ok := p.Roles["test"]
	assert.False(t, ok)
}

func TestPolicy_RemoveRole_NotFound(t *testing.T) {
	p := NewPolicy()

	err := p.RemoveRole("nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestPolicy_GetRole(t *testing.T) {
	p := NewPolicy()

	role := Role{Name: "test", Description: "Test role", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(role))

	retrieved, err := p.GetRole("test")
	require.NoError(t, err)
	assert.Equal(t, "test", retrieved.Name)
	assert.Equal(t, "Test role", retrieved.Description)
}

func TestPolicy_GetRole_NotFound(t *testing.T) {
	p := NewPolicy()

	_, err := p.GetRole("nonexistent")
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestPolicy_ListRoles(t *testing.T) {
	p := NewPolicy()

	role1 := Role{Name: "role1", Permissions: []Permission{"r:a"}}
	role2 := Role{Name: "role2", Permissions: []Permission{"r:b"}}

	require.NoError(t, p.AddRole(role1))
	require.NoError(t, p.AddRole(role2))

	roles := p.ListRoles()
	assert.Len(t, roles, 2)

	names := make([]string, 0, 2)
	for _, r := range roles {
		names = append(names, r.Name)
	}
	assert.Contains(t, names, "role1")
	assert.Contains(t, names, "role2")
}

func TestRoleBuilder(t *testing.T) {
	role := NewRoleBuilder("custom").
		Description("A custom role").
		Permission("resource1:read").
		Permission("resource1:write").
		Permissions("resource2:read", "resource2:write").
		Inherits("base1", "base2").
		Build()

	assert.Equal(t, "custom", role.Name)
	assert.Equal(t, "A custom role", role.Description)
	assert.Len(t, role.Permissions, 4)
	assert.Contains(t, role.Permissions, Permission("resource1:read"))
	assert.Contains(t, role.Permissions, Permission("resource1:write"))
	assert.Contains(t, role.Permissions, Permission("resource2:read"))
	assert.Contains(t, role.Permissions, Permission("resource2:write"))
	assert.Len(t, role.Parents, 2)
	assert.Contains(t, role.Parents, "base1")
	assert.Contains(t, role.Parents, "base2")
}

func TestValidatePermission(t *testing.T) {
	tests := []struct {
		name      string
		perm      Permission
		expectErr bool
	}{
		{"valid_simple", Permission("users:read"), false},
		{"valid_wildcard_resource", Permission("*:read"), false},
		{"valid_wildcard_action", Permission("users:*"), false},
		{"valid_full_wildcard", Permission("*:*"), false},
		{"empty", Permission(""), true},
		{"no_colon", Permission("usersread"), true},
		{"empty_resource", Permission(":read"), true},
		{"empty_action", Permission("users:"), true},
		{"multiple_colons", Permission("a:b:c"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePermission(tt.perm)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name      string
		roleName  string
		expectErr bool
	}{
		{"valid_simple", "admin", false},
		{"valid_with_underscore", "super_admin", false},
		{"valid_with_dash", "super-admin", false},
		{"valid_with_numbers", "admin123", false},
		{"empty", "", true},
		{"with_space", "role name", true},
		{"with_tab", "role\tname", true},
		{"with_newline", "role\nname", true},
		{"with_carriage_return", "role\rname", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoleName(tt.roleName)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultRoles(t *testing.T) {
	// Verify DefaultRoles are properly configured
	assert.Len(t, DefaultRoles, 3)

	roleMap := make(map[string]Role)
	for _, r := range DefaultRoles {
		roleMap[r.Name] = r
	}

	// Admin should have full access
	admin := roleMap["admin"]
	assert.Equal(t, "admin", admin.Name)
	assert.Contains(t, admin.Permissions, Permission("*:*"))
	assert.Empty(t, admin.Parents)

	// Editor should inherit from viewer
	editor := roleMap["editor"]
	assert.Equal(t, "editor", editor.Name)
	assert.Contains(t, editor.Parents, "viewer")
	assert.Contains(t, editor.Permissions, Permission("configs:create"))

	// Viewer should be the base role
	viewer := roleMap["viewer"]
	assert.Equal(t, "viewer", viewer.Name)
	assert.Empty(t, viewer.Parents)
	assert.Contains(t, viewer.Permissions, Permission("configs:read"))
}

func TestPolicy_CyclicDetection_VisitedSkip(t *testing.T) {
	p := NewPolicy()

	// Create a diamond inheritance pattern (A -> B -> D, A -> C -> D)
	// This tests the visited skip path in detectCycle
	base := Role{Name: "base", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(base))

	left := Role{Name: "left", Permissions: []Permission{"r:a"}, Parents: []string{"base"}}
	require.NoError(t, p.AddRole(left))

	right := Role{Name: "right", Permissions: []Permission{"r:a"}, Parents: []string{"base"}}
	require.NoError(t, p.AddRole(right))

	// Adding a role that inherits from both left and right should work (diamond)
	diamond := Role{Name: "diamond", Permissions: []Permission{"r:a"}, Parents: []string{"left", "right"}}
	err := p.AddRole(diamond)
	assert.NoError(t, err)
}

func TestPolicy_UpdateRole_CyclicInheritance(t *testing.T) {
	p := NewPolicy()

	// Setup initial roles
	roleA := Role{Name: "roleA", Permissions: []Permission{"r:a"}}
	require.NoError(t, p.AddRole(roleA))

	roleB := Role{Name: "roleB", Permissions: []Permission{"r:a"}, Parents: []string{"roleA"}}
	require.NoError(t, p.AddRole(roleB))

	// Try to update roleA to inherit from roleB (creating a cycle)
	updated := Role{Name: "roleA", Permissions: []Permission{"r:a"}, Parents: []string{"roleB"}}
	err := p.UpdateRole(updated)
	assert.Equal(t, ErrCyclicInheritance, err)
}
