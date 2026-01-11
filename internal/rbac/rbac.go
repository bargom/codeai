// Package rbac provides role-based access control with permission inheritance.
package rbac

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/bargom/codeai/internal/auth"
)

// Permission represents a permission in the format "resource:action".
type Permission string

// String returns the string representation of the permission.
func (p Permission) String() string {
	return string(p)
}

// Resource returns the resource part of the permission.
func (p Permission) Resource() string {
	parts := strings.SplitN(string(p), ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Action returns the action part of the permission.
func (p Permission) Action() string {
	parts := strings.SplitN(string(p), ":", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Matches checks if this permission matches another permission.
// Supports wildcards: "*:read" matches any resource with read action,
// "users:*" matches any action on users resource.
func (p Permission) Matches(target Permission) bool {
	pResource := p.Resource()
	pAction := p.Action()
	tResource := target.Resource()
	tAction := target.Action()

	resourceMatch := pResource == "*" || pResource == tResource
	actionMatch := pAction == "*" || pAction == tAction

	return resourceMatch && actionMatch
}

// Role represents a role with associated permissions.
type Role struct {
	Name        string
	Description string
	Permissions []Permission
	Parents     []string // Parent roles for inheritance
}

// Engine is the RBAC engine that manages roles and permissions.
type Engine struct {
	storage Storage
	cache   *permissionCache
	logger  *slog.Logger
	mu      sync.RWMutex
}

// permissionCache caches resolved permissions for efficiency.
type permissionCache struct {
	rolePermissions map[string][]Permission
	mu              sync.RWMutex
}

func newPermissionCache() *permissionCache {
	return &permissionCache{
		rolePermissions: make(map[string][]Permission),
	}
}

func (c *permissionCache) get(role string) ([]Permission, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	perms, ok := c.rolePermissions[role]
	return perms, ok
}

func (c *permissionCache) set(role string, perms []Permission) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rolePermissions[role] = perms
}

func (c *permissionCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rolePermissions = make(map[string][]Permission)
}

// NewEngine creates a new RBAC engine with the given storage backend.
func NewEngine(storage Storage) *Engine {
	return &Engine{
		storage: storage,
		cache:   newPermissionCache(),
		logger:  slog.Default().With("component", "rbac-engine"),
	}
}

// NewEngineWithLogger creates a new RBAC engine with a custom logger.
func NewEngineWithLogger(storage Storage, logger *slog.Logger) *Engine {
	e := NewEngine(storage)
	if logger != nil {
		e.logger = logger.With("component", "rbac-engine")
	}
	return e
}

// CheckPermission checks if the user has the specified permission.
// It resolves all permissions from the user's roles including inherited ones.
func (e *Engine) CheckPermission(ctx context.Context, user *auth.User, permission Permission) bool {
	if user == nil {
		return false
	}

	// Check direct permissions from JWT claims first
	for _, p := range user.Permissions {
		if Permission(p).Matches(permission) {
			e.logger.Debug("permission granted via direct claim",
				"user", user.ID,
				"permission", permission,
				"granted_by", p)
			return true
		}
	}

	// Check permissions from roles
	allPerms := e.resolveUserPermissions(ctx, user.Roles)
	for _, p := range allPerms {
		if p.Matches(permission) {
			e.logger.Debug("permission granted via role",
				"user", user.ID,
				"permission", permission,
				"granted_by", p)
			return true
		}
	}

	e.logger.Debug("permission denied",
		"user", user.ID,
		"permission", permission)
	return false
}

// CheckRole checks if the user has the specified role.
func (e *Engine) CheckRole(ctx context.Context, user *auth.User, roleName string) bool {
	if user == nil {
		return false
	}

	for _, r := range user.Roles {
		if r == roleName {
			return true
		}
	}

	// Check inherited roles
	for _, r := range user.Roles {
		if e.hasInheritedRole(ctx, r, roleName) {
			return true
		}
	}

	return false
}

// hasInheritedRole checks if parentRole inherits targetRole.
func (e *Engine) hasInheritedRole(ctx context.Context, parentRole, targetRole string) bool {
	role, err := e.storage.GetRole(ctx, parentRole)
	if err != nil || role == nil {
		return false
	}

	for _, parent := range role.Parents {
		if parent == targetRole {
			return true
		}
		if e.hasInheritedRole(ctx, parent, targetRole) {
			return true
		}
	}

	return false
}

// resolveUserPermissions resolves all permissions for the given roles including inheritance.
func (e *Engine) resolveUserPermissions(ctx context.Context, roles []string) []Permission {
	var allPerms []Permission
	seen := make(map[string]bool)

	for _, roleName := range roles {
		perms := e.resolveRolePermissions(ctx, roleName, seen)
		allPerms = append(allPerms, perms...)
	}

	return allPerms
}

// resolveRolePermissions recursively resolves permissions for a role and its parents.
func (e *Engine) resolveRolePermissions(ctx context.Context, roleName string, seen map[string]bool) []Permission {
	if seen[roleName] {
		return nil // Avoid cycles
	}
	seen[roleName] = true

	// Check cache first
	if cached, ok := e.cache.get(roleName); ok {
		return cached
	}

	role, err := e.storage.GetRole(ctx, roleName)
	if err != nil || role == nil {
		e.logger.Debug("role not found in storage", "role", roleName)
		return nil
	}

	perms := make([]Permission, len(role.Permissions))
	copy(perms, role.Permissions)

	// Resolve parent roles
	for _, parent := range role.Parents {
		parentPerms := e.resolveRolePermissions(ctx, parent, seen)
		perms = append(perms, parentPerms...)
	}

	// Cache the resolved permissions
	e.cache.set(roleName, perms)

	return perms
}

// InvalidateCache clears the permission cache.
// Call this when roles or permissions are updated.
func (e *Engine) InvalidateCache() {
	e.cache.clear()
	e.logger.Debug("permission cache invalidated")
}

// GetUserPermissions returns all effective permissions for a user.
func (e *Engine) GetUserPermissions(ctx context.Context, user *auth.User) []Permission {
	if user == nil {
		return nil
	}

	// Start with direct permissions
	perms := make([]Permission, 0, len(user.Permissions))
	for _, p := range user.Permissions {
		perms = append(perms, Permission(p))
	}

	// Add role-based permissions
	rolePerms := e.resolveUserPermissions(ctx, user.Roles)
	perms = append(perms, rolePerms...)

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]Permission, 0, len(perms))
	for _, p := range perms {
		if !seen[p.String()] {
			seen[p.String()] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// GetUserRoles returns all effective roles for a user including inherited ones.
func (e *Engine) GetUserRoles(ctx context.Context, user *auth.User) []string {
	if user == nil {
		return nil
	}

	seen := make(map[string]bool)
	roles := make([]string, 0)

	var collectRoles func(roleName string)
	collectRoles = func(roleName string) {
		if seen[roleName] {
			return
		}
		seen[roleName] = true
		roles = append(roles, roleName)

		role, err := e.storage.GetRole(ctx, roleName)
		if err != nil || role == nil {
			return
		}

		for _, parent := range role.Parents {
			collectRoles(parent)
		}
	}

	for _, r := range user.Roles {
		collectRoles(r)
	}

	return roles
}
