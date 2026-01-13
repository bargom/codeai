package auth

import (
	"fmt"

	"github.com/bargom/codeai/internal/ast"
)

// LoadedAuth represents an auth provider loaded from DSL.
type LoadedAuth struct {
	Name     string
	Method   ast.AuthMethod
	Config   *Config
	JWKS     *ast.JWKSConfig
	RawDecl  *ast.AuthDecl
}

// LoadedRole represents a role loaded from DSL.
type LoadedRole struct {
	Name        string
	Permissions []string
	RawDecl     *ast.RoleDecl
}

// LoadedMiddleware represents middleware loaded from DSL.
type LoadedMiddleware struct {
	Name           string
	MiddlewareType string
	Config         map[string]any
	Provider       string // Reference to auth provider
	Required       bool
	RawDecl        *ast.MiddlewareDecl
}

// DSLLoader loads auth, role, and middleware configurations from parsed AST.
type DSLLoader struct {
	auths       map[string]*LoadedAuth
	roles       map[string]*LoadedRole
	middlewares map[string]*LoadedMiddleware
}

// NewDSLLoader creates a new DSL loader.
func NewDSLLoader() *DSLLoader {
	return &DSLLoader{
		auths:       make(map[string]*LoadedAuth),
		roles:       make(map[string]*LoadedRole),
		middlewares: make(map[string]*LoadedMiddleware),
	}
}

// LoadProgram loads all auth, role, and middleware declarations from a parsed program.
func (l *DSLLoader) LoadProgram(program *ast.Program) error {
	for _, stmt := range program.Statements {
		switch decl := stmt.(type) {
		case *ast.AuthDecl:
			if err := l.loadAuth(decl); err != nil {
				return fmt.Errorf("loading auth %q: %w", decl.Name, err)
			}
		case *ast.RoleDecl:
			if err := l.loadRole(decl); err != nil {
				return fmt.Errorf("loading role %q: %w", decl.Name, err)
			}
		case *ast.MiddlewareDecl:
			if err := l.loadMiddleware(decl); err != nil {
				return fmt.Errorf("loading middleware %q: %w", decl.Name, err)
			}
		}
	}
	return nil
}

// loadAuth loads an auth declaration into the loader.
func (l *DSLLoader) loadAuth(decl *ast.AuthDecl) error {
	if _, exists := l.auths[decl.Name]; exists {
		return fmt.Errorf("duplicate auth provider: %s", decl.Name)
	}

	loaded := &LoadedAuth{
		Name:    decl.Name,
		Method:  decl.Method,
		JWKS:    decl.JWKS,
		RawDecl: decl,
	}

	// Build Config from JWKS if present
	if decl.JWKS != nil {
		loaded.Config = &Config{
			JWKSURL:  decl.JWKS.URL,
			Issuer:   decl.JWKS.Issuer,
			Audience: decl.JWKS.Audience,
		}
	}

	l.auths[decl.Name] = loaded
	return nil
}

// loadRole loads a role declaration into the loader.
func (l *DSLLoader) loadRole(decl *ast.RoleDecl) error {
	if _, exists := l.roles[decl.Name]; exists {
		return fmt.Errorf("duplicate role: %s", decl.Name)
	}

	l.roles[decl.Name] = &LoadedRole{
		Name:        decl.Name,
		Permissions: decl.Permissions,
		RawDecl:     decl,
	}
	return nil
}

// loadMiddleware loads a middleware declaration into the loader.
func (l *DSLLoader) loadMiddleware(decl *ast.MiddlewareDecl) error {
	if _, exists := l.middlewares[decl.Name]; exists {
		return fmt.Errorf("duplicate middleware: %s", decl.Name)
	}

	loaded := &LoadedMiddleware{
		Name:           decl.Name,
		MiddlewareType: decl.MiddlewareType,
		Config:         make(map[string]any),
		RawDecl:        decl,
	}

	// Extract config values
	for key, expr := range decl.Config {
		value := l.extractExpressionValue(expr)
		loaded.Config[key] = value

		// Handle special keys
		switch key {
		case "provider":
			if s, ok := value.(string); ok {
				loaded.Provider = s
			}
		case "required":
			if b, ok := value.(bool); ok {
				loaded.Required = b
			}
		}
	}

	l.middlewares[decl.Name] = loaded
	return nil
}

// extractExpressionValue extracts a Go value from an AST expression.
func (l *DSLLoader) extractExpressionValue(expr ast.Expression) any {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.NumberLiteral:
		return e.Value
	case *ast.BoolLiteral:
		return e.Value
	case *ast.Identifier:
		return e.Name
	default:
		return nil
	}
}

// GetAuth returns a loaded auth provider by name.
func (l *DSLLoader) GetAuth(name string) (*LoadedAuth, bool) {
	auth, ok := l.auths[name]
	return auth, ok
}

// GetRole returns a loaded role by name.
func (l *DSLLoader) GetRole(name string) (*LoadedRole, bool) {
	role, ok := l.roles[name]
	return role, ok
}

// GetMiddleware returns a loaded middleware by name.
func (l *DSLLoader) GetMiddleware(name string) (*LoadedMiddleware, bool) {
	mw, ok := l.middlewares[name]
	return mw, ok
}

// AllAuths returns all loaded auth providers.
func (l *DSLLoader) AllAuths() map[string]*LoadedAuth {
	return l.auths
}

// AllRoles returns all loaded roles.
func (l *DSLLoader) AllRoles() map[string]*LoadedRole {
	return l.roles
}

// AllMiddlewares returns all loaded middlewares.
func (l *DSLLoader) AllMiddlewares() map[string]*LoadedMiddleware {
	return l.middlewares
}

// CreateValidator creates a JWT validator for the specified auth provider.
func (l *DSLLoader) CreateValidator(authName string) (*Validator, error) {
	auth, ok := l.auths[authName]
	if !ok {
		return nil, fmt.Errorf("unknown auth provider: %s", authName)
	}

	if auth.Config == nil {
		return nil, fmt.Errorf("auth provider %s has no configuration", authName)
	}

	return NewValidator(*auth.Config)
}

// CreateMiddlewareChain creates an authentication middleware for the specified middleware name.
func (l *DSLLoader) CreateMiddlewareChain(middlewareName string) (*Middleware, error) {
	mw, ok := l.middlewares[middlewareName]
	if !ok {
		return nil, fmt.Errorf("unknown middleware: %s", middlewareName)
	}

	if mw.MiddlewareType != "authentication" {
		return nil, fmt.Errorf("middleware %s is not an authentication middleware", middlewareName)
	}

	if mw.Provider == "" {
		return nil, fmt.Errorf("middleware %s has no provider configured", middlewareName)
	}

	validator, err := l.CreateValidator(mw.Provider)
	if err != nil {
		return nil, fmt.Errorf("creating validator for middleware %s: %w", middlewareName, err)
	}

	return NewMiddleware(validator), nil
}

// RoleHasPermission checks if a role has a specific permission.
func (l *DSLLoader) RoleHasPermission(roleName, permission string) bool {
	role, ok := l.roles[roleName]
	if !ok {
		return false
	}

	for _, p := range role.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissionsForRoles returns all permissions for the given roles.
func (l *DSLLoader) GetPermissionsForRoles(roleNames []string) []string {
	seen := make(map[string]bool)
	var perms []string

	for _, roleName := range roleNames {
		role, ok := l.roles[roleName]
		if !ok {
			continue
		}

		for _, p := range role.Permissions {
			if !seen[p] {
				seen[p] = true
				perms = append(perms, p)
			}
		}
	}

	return perms
}

// LoadAuthConfig loads an auth configuration from a parsed Auth struct.
// Returns an error if the auth is nil or has invalid configuration.
func LoadAuthConfig(auth *ast.AuthDecl) (*LoadedAuth, error) {
	if auth == nil {
		return nil, fmt.Errorf("auth declaration cannot be nil")
	}

	loaded := &LoadedAuth{
		Name:    auth.Name,
		Method:  auth.Method,
		JWKS:    auth.JWKS,
		RawDecl: auth,
	}

	// Validate HTTPS for JWKS URLs
	if auth.JWKS != nil {
		if auth.JWKS.URL != "" && len(auth.JWKS.URL) > 4 && auth.JWKS.URL[:4] == "http" && auth.JWKS.URL[:5] != "https" {
			return nil, fmt.Errorf("JWKS URL must use HTTPS, got: %s", auth.JWKS.URL)
		}

		loaded.Config = &Config{
			JWKSURL:  auth.JWKS.URL,
			Issuer:   auth.JWKS.Issuer,
			Audience: auth.JWKS.Audience,
		}
	}

	return loaded, nil
}

// LoadRoleConfig loads a role configuration from a parsed Role struct.
// Returns an error if the role is nil or has invalid permissions.
func LoadRoleConfig(role *ast.RoleDecl) (*LoadedRole, error) {
	if role == nil {
		return nil, fmt.Errorf("role declaration cannot be nil")
	}

	// Validate permission format (should be "resource:action")
	for _, perm := range role.Permissions {
		if !isValidPermissionFormat(perm) {
			return nil, fmt.Errorf("invalid permission format: %s (expected format: 'resource:action')", perm)
		}
	}

	return &LoadedRole{
		Name:        role.Name,
		Permissions: role.Permissions,
		RawDecl:     role,
	}, nil
}

// LoadRolesMap loads multiple roles into a map, checking for duplicates.
// Returns an error if there are duplicate role names.
func LoadRolesMap(roles []*ast.RoleDecl) (map[string]*LoadedRole, error) {
	roleMap := make(map[string]*LoadedRole)

	for _, role := range roles {
		if _, exists := roleMap[role.Name]; exists {
			return nil, fmt.Errorf("duplicate role: %s", role.Name)
		}

		loaded, err := LoadRoleConfig(role)
		if err != nil {
			return nil, fmt.Errorf("loading role %q: %w", role.Name, err)
		}

		roleMap[role.Name] = loaded
	}

	return roleMap, nil
}

// isValidPermissionFormat validates that a permission follows the "resource:action" format.
func isValidPermissionFormat(permission string) bool {
	for i, char := range permission {
		if char == ':' {
			// Must have at least one character before and after the colon
			return i > 0 && i < len(permission)-1
		}
	}
	return false
}
