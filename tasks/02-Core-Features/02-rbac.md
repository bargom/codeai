# Task: Role-Based Access Control (RBAC)

## Overview
Implement role-based access control with permission inheritance, resource-level policies, and owner-based access patterns.

## Phase
Phase 2: Core Features

## Priority
High - Required for secure multi-user applications.

## Dependencies
- 02-Core-Features/01-jwt-authentication.md

## Description
Create an RBAC system supporting hierarchical roles, permission inheritance, and resource-level access control that integrates with CodeAI's declarative syntax.

## Detailed Requirements

### 1. Core RBAC Types (internal/modules/auth/rbac.go)

```go
package auth

import "sync"

type Role struct {
    Name        string
    Description string
    Permissions []Permission
    Inherits    []string
}

type Permission struct {
    Resource string // "users", "products", "*"
    Action   string // "read", "write", "delete", "*"
}

func (p Permission) String() string {
    return p.Resource + ":" + p.Action
}

func (p Permission) Matches(other Permission) bool {
    resourceMatch := p.Resource == "*" || p.Resource == other.Resource
    actionMatch := p.Action == "*" || p.Action == other.Action
    return resourceMatch && actionMatch
}

type RBACModule struct {
    roles       map[string]*Role
    permissions map[string][]Permission // Cached: role -> all permissions
    mu          sync.RWMutex
}

func NewRBACModule() *RBACModule {
    m := &RBACModule{
        roles:       make(map[string]*Role),
        permissions: make(map[string][]Permission),
    }
    m.registerDefaultRoles()
    return m
}

func (m *RBACModule) registerDefaultRoles() {
    defaults := []*Role{
        {Name: "admin", Permissions: []Permission{{Resource: "*", Action: "*"}}},
        {Name: "user", Permissions: []Permission{{Resource: "profile", Action: "*"}}},
        {Name: "viewer", Permissions: []Permission{{Resource: "*", Action: "read"}}},
        {Name: "editor", Inherits: []string{"viewer"}, Permissions: []Permission{{Resource: "*", Action: "write"}}},
    }
    for _, role := range defaults {
        m.RegisterRole(role)
    }
}

func (m *RBACModule) RegisterRole(role *Role) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.roles[role.Name] = role
    m.recomputePermissions()
}

func (m *RBACModule) recomputePermissions() {
    m.permissions = make(map[string][]Permission)
    for name := range m.roles {
        m.permissions[name] = m.collectPermissions(name, make(map[string]bool))
    }
}

func (m *RBACModule) collectPermissions(roleName string, visited map[string]bool) []Permission {
    if visited[roleName] {
        return nil
    }
    visited[roleName] = true

    role, ok := m.roles[roleName]
    if !ok {
        return nil
    }

    perms := append([]Permission{}, role.Permissions...)
    for _, inherited := range role.Inherits {
        perms = append(perms, m.collectPermissions(inherited, visited)...)
    }
    return perms
}

func (m *RBACModule) HasPermission(user *User, resource, action string) bool {
    if user == nil {
        return false
    }

    required := Permission{Resource: resource, Action: action}

    m.mu.RLock()
    defer m.mu.RUnlock()

    for _, roleName := range user.Roles {
        for _, perm := range m.permissions[roleName] {
            if perm.Matches(required) {
                return true
            }
        }
    }
    return false
}
```

### 2. Resource Policies (internal/modules/auth/policy.go)

```go
package auth

import (
    "context"
    "sync"
)

type ResourcePolicy struct {
    Resource string
    Rules    []AccessRule
}

type AccessRule struct {
    Actions    []string
    Roles      []string
    Conditions []Condition
}

type Condition struct {
    Type  string // "owner", "field_equals", "field_in_user"
    Field string
    Value any
}

type PolicyEngine struct {
    policies map[string]*ResourcePolicy
    rbac     *RBACModule
    mu       sync.RWMutex
}

func NewPolicyEngine(rbac *RBACModule) *PolicyEngine {
    return &PolicyEngine{
        policies: make(map[string]*ResourcePolicy),
        rbac:     rbac,
    }
}

func (e *PolicyEngine) RegisterPolicy(policy *ResourcePolicy) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.policies[policy.Resource] = policy
}

func (e *PolicyEngine) CanAccess(ctx context.Context, user *User, resource, action string, data map[string]any) bool {
    // Check RBAC first
    if e.rbac.HasPermission(user, resource, action) {
        return true
    }

    // Check resource policies
    e.mu.RLock()
    policy, ok := e.policies[resource]
    e.mu.RUnlock()

    if !ok {
        return false
    }

    for _, rule := range policy.Rules {
        if !contains(rule.Actions, action) && !contains(rule.Actions, "*") {
            continue
        }

        hasRole := false
        for _, role := range rule.Roles {
            if user.HasRole(role) {
                hasRole = true
                break
            }
        }
        if !hasRole && len(rule.Roles) > 0 {
            continue
        }

        if e.evaluateConditions(user, rule.Conditions, data) {
            return true
        }
    }
    return false
}

func (e *PolicyEngine) evaluateConditions(user *User, conditions []Condition, data map[string]any) bool {
    for _, cond := range conditions {
        switch cond.Type {
        case "owner":
            if ownerID, _ := data[cond.Field].(string); ownerID != user.ID {
                return false
            }
        case "field_equals":
            if data[cond.Field] != cond.Value {
                return false
            }
        }
    }
    return true
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 3. Authorization Middleware

```go
func (m *RBACModule) AuthorizationMiddleware(roles []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            if user == nil {
                writeError(w, http.StatusUnauthorized, "authentication required")
                return
            }

            if len(roles) > 0 {
                hasRole := false
                for _, role := range roles {
                    if user.HasRole(role) {
                        hasRole = true
                        break
                    }
                }
                if !hasRole {
                    writeError(w, http.StatusForbidden, "insufficient permissions")
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Acceptance Criteria
- [ ] Role registration with inheritance
- [ ] Permission matching with wildcards
- [ ] Resource-level policies with conditions
- [ ] Owner-based access control
- [ ] Authorization middleware
- [ ] Integration with JWT claims

## Testing Strategy
- Unit tests for permission matching
- Tests for role inheritance
- Integration tests with policies

## Files to Create
- `internal/modules/auth/rbac.go`
- `internal/modules/auth/policy.go`
- `internal/modules/auth/rbac_test.go`
