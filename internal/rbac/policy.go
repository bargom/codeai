package rbac

import (
	"errors"
	"strings"
)

// Common errors for policy operations.
var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleAlreadyExists  = errors.New("role already exists")
	ErrInvalidPermission  = errors.New("invalid permission format")
	ErrCyclicInheritance  = errors.New("cyclic role inheritance detected")
	ErrInvalidRoleName    = errors.New("invalid role name")
)

// DefaultRoles defines the standard roles with their permissions.
var DefaultRoles = []Role{
	{
		Name:        "admin",
		Description: "Full system access",
		Permissions: []Permission{
			"*:*", // Full access to everything
		},
		Parents: []string{}, // Admin is the top role
	},
	{
		Name:        "editor",
		Description: "Can read and modify resources",
		Permissions: []Permission{
			"configs:read",
			"configs:create",
			"configs:update",
			"configs:delete",
			"executions:read",
			"executions:create",
			"deployments:read",
			"deployments:create",
		},
		Parents: []string{"viewer"}, // Inherits from viewer
	},
	{
		Name:        "viewer",
		Description: "Read-only access",
		Permissions: []Permission{
			"configs:read",
			"executions:read",
			"deployments:read",
			"health:read",
		},
		Parents: []string{}, // Base role
	},
}

// Policy represents a collection of roles and their permissions.
type Policy struct {
	Roles       map[string]*Role
	Description string
	Version     string
}

// NewPolicy creates a new empty policy.
func NewPolicy() *Policy {
	return &Policy{
		Roles:   make(map[string]*Role),
		Version: "1.0",
	}
}

// NewDefaultPolicy creates a policy with the default roles.
func NewDefaultPolicy() *Policy {
	p := NewPolicy()
	for i := range DefaultRoles {
		role := DefaultRoles[i]
		p.Roles[role.Name] = &role
	}
	return p
}

// AddRole adds a new role to the policy.
func (p *Policy) AddRole(role Role) error {
	if err := validateRoleName(role.Name); err != nil {
		return err
	}

	if _, exists := p.Roles[role.Name]; exists {
		return ErrRoleAlreadyExists
	}

	// Validate permissions format
	for _, perm := range role.Permissions {
		if err := validatePermission(perm); err != nil {
			return err
		}
	}

	// Check for cyclic inheritance
	if err := p.checkCyclicInheritance(role.Name, role.Parents); err != nil {
		return err
	}

	p.Roles[role.Name] = &role
	return nil
}

// UpdateRole updates an existing role.
func (p *Policy) UpdateRole(role Role) error {
	if err := validateRoleName(role.Name); err != nil {
		return err
	}

	if _, exists := p.Roles[role.Name]; !exists {
		return ErrRoleNotFound
	}

	// Validate permissions format
	for _, perm := range role.Permissions {
		if err := validatePermission(perm); err != nil {
			return err
		}
	}

	// Check for cyclic inheritance
	if err := p.checkCyclicInheritance(role.Name, role.Parents); err != nil {
		return err
	}

	p.Roles[role.Name] = &role
	return nil
}

// RemoveRole removes a role from the policy.
func (p *Policy) RemoveRole(name string) error {
	if _, exists := p.Roles[name]; !exists {
		return ErrRoleNotFound
	}

	delete(p.Roles, name)
	return nil
}

// GetRole returns a role by name.
func (p *Policy) GetRole(name string) (*Role, error) {
	role, exists := p.Roles[name]
	if !exists {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

// ListRoles returns all roles in the policy.
func (p *Policy) ListRoles() []*Role {
	roles := make([]*Role, 0, len(p.Roles))
	for _, role := range p.Roles {
		roles = append(roles, role)
	}
	return roles
}

// checkCyclicInheritance detects cycles in role inheritance.
func (p *Policy) checkCyclicInheritance(roleName string, parents []string) error {
	visited := make(map[string]bool)
	return p.detectCycle(roleName, parents, visited)
}

func (p *Policy) detectCycle(roleName string, parents []string, visited map[string]bool) error {
	for _, parent := range parents {
		if parent == roleName {
			return ErrCyclicInheritance
		}

		if visited[parent] {
			continue
		}
		visited[parent] = true

		if parentRole, exists := p.Roles[parent]; exists {
			if err := p.detectCycle(roleName, parentRole.Parents, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateRoleName checks if a role name is valid.
func validateRoleName(name string) error {
	if name == "" {
		return ErrInvalidRoleName
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return ErrInvalidRoleName
	}
	return nil
}

// validatePermission checks if a permission string is valid.
func validatePermission(perm Permission) error {
	s := string(perm)
	if s == "" {
		return ErrInvalidPermission
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return ErrInvalidPermission
	}
	if parts[0] == "" || parts[1] == "" {
		return ErrInvalidPermission
	}
	return nil
}

// RoleBuilder provides a fluent API for building roles.
type RoleBuilder struct {
	role Role
}

// NewRoleBuilder creates a new role builder.
func NewRoleBuilder(name string) *RoleBuilder {
	return &RoleBuilder{
		role: Role{
			Name:        name,
			Permissions: make([]Permission, 0),
			Parents:     make([]string, 0),
		},
	}
}

// Description sets the role description.
func (b *RoleBuilder) Description(desc string) *RoleBuilder {
	b.role.Description = desc
	return b
}

// Permission adds a permission to the role.
func (b *RoleBuilder) Permission(perm string) *RoleBuilder {
	b.role.Permissions = append(b.role.Permissions, Permission(perm))
	return b
}

// Permissions adds multiple permissions to the role.
func (b *RoleBuilder) Permissions(perms ...string) *RoleBuilder {
	for _, p := range perms {
		b.role.Permissions = append(b.role.Permissions, Permission(p))
	}
	return b
}

// Inherits adds parent roles for inheritance.
func (b *RoleBuilder) Inherits(parents ...string) *RoleBuilder {
	b.role.Parents = append(b.role.Parents, parents...)
	return b
}

// Build returns the constructed role.
func (b *RoleBuilder) Build() Role {
	return b.role
}
