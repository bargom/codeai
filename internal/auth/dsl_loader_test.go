package auth

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDSLLoaderLoadProgram tests loading a complete program with auth, roles, and middleware.
func TestDSLLoaderLoadProgram(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
	jwks_url "https://auth.example.com/.well-known/jwks.json"
	issuer "https://auth.example.com"
	audience "api.example.com"
}

role admin {
	permissions ["users:read", "users:write", "users:delete"]
}

role viewer {
	permissions ["users:read"]
}

middleware auth_check {
	type authentication
	config {
		provider: jwt_provider
		required: true
	}
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	// Verify auth provider was loaded
	auth, ok := loader.GetAuth("jwt_provider")
	require.True(t, ok, "expected jwt_provider to be loaded")
	assert.Equal(t, "jwt_provider", auth.Name)
	assert.Equal(t, ast.AuthMethodJWT, auth.Method)
	require.NotNil(t, auth.JWKS)
	assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", auth.JWKS.URL)
	assert.Equal(t, "https://auth.example.com", auth.JWKS.Issuer)
	assert.Equal(t, "api.example.com", auth.JWKS.Audience)

	// Verify roles were loaded
	adminRole, ok := loader.GetRole("admin")
	require.True(t, ok, "expected admin role to be loaded")
	assert.Equal(t, "admin", adminRole.Name)
	assert.Equal(t, []string{"users:read", "users:write", "users:delete"}, adminRole.Permissions)

	viewerRole, ok := loader.GetRole("viewer")
	require.True(t, ok, "expected viewer role to be loaded")
	assert.Equal(t, "viewer", viewerRole.Name)
	assert.Equal(t, []string{"users:read"}, viewerRole.Permissions)

	// Verify middleware was loaded
	mw, ok := loader.GetMiddleware("auth_check")
	require.True(t, ok, "expected auth_check middleware to be loaded")
	assert.Equal(t, "auth_check", mw.Name)
	assert.Equal(t, "authentication", mw.MiddlewareType)
	assert.Equal(t, "jwt_provider", mw.Provider)
	assert.True(t, mw.Required)
}

// TestDSLLoaderLoadJWKSConfig tests loading JWKS configuration.
func TestDSLLoaderLoadJWKSConfig(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
	jwks_url "https://auth.example.com/.well-known/jwks.json"
	issuer "https://auth.example.com"
	audience "api.example.com"
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	auth, ok := loader.GetAuth("jwt_provider")
	require.True(t, ok)

	// Verify Config struct is properly populated
	require.NotNil(t, auth.Config)
	assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", auth.Config.JWKSURL)
	assert.Equal(t, "https://auth.example.com", auth.Config.Issuer)
	assert.Equal(t, "api.example.com", auth.Config.Audience)
}

// TestDSLLoaderLoadRoles tests loading role definitions.
func TestDSLLoaderLoadRoles(t *testing.T) {
	t.Parallel()

	input := `
role admin {
	permissions ["users:read", "users:write", "users:delete", "posts:read", "posts:write", "posts:delete"]
}

role editor {
	permissions ["posts:read", "posts:write"]
}

role guest {
	permissions []
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	// Verify all roles were loaded
	allRoles := loader.AllRoles()
	assert.Len(t, allRoles, 3)

	// Verify admin role
	admin, ok := loader.GetRole("admin")
	require.True(t, ok)
	assert.Len(t, admin.Permissions, 6)

	// Verify editor role
	editor, ok := loader.GetRole("editor")
	require.True(t, ok)
	assert.Len(t, editor.Permissions, 2)

	// Verify guest role (empty permissions)
	guest, ok := loader.GetRole("guest")
	require.True(t, ok)
	assert.Len(t, guest.Permissions, 0)
}

// TestDSLLoaderRoleHasPermission tests checking permissions.
func TestDSLLoaderRoleHasPermission(t *testing.T) {
	t.Parallel()

	input := `
role admin {
	permissions ["users:read", "users:write", "users:delete"]
}

role viewer {
	permissions ["users:read"]
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	// Test admin permissions
	assert.True(t, loader.RoleHasPermission("admin", "users:read"))
	assert.True(t, loader.RoleHasPermission("admin", "users:write"))
	assert.True(t, loader.RoleHasPermission("admin", "users:delete"))
	assert.False(t, loader.RoleHasPermission("admin", "posts:write"))

	// Test viewer permissions
	assert.True(t, loader.RoleHasPermission("viewer", "users:read"))
	assert.False(t, loader.RoleHasPermission("viewer", "users:write"))
	assert.False(t, loader.RoleHasPermission("viewer", "users:delete"))

	// Test unknown role
	assert.False(t, loader.RoleHasPermission("unknown", "users:read"))
}

// TestDSLLoaderGetPermissionsForRoles tests getting permissions for multiple roles.
func TestDSLLoaderGetPermissionsForRoles(t *testing.T) {
	t.Parallel()

	input := `
role admin {
	permissions ["users:delete", "system:manage"]
}

role editor {
	permissions ["users:read", "users:write"]
}

role viewer {
	permissions ["users:read"]
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	// Get permissions for multiple roles
	perms := loader.GetPermissionsForRoles([]string{"editor", "viewer"})
	assert.Contains(t, perms, "users:read")
	assert.Contains(t, perms, "users:write")
	assert.NotContains(t, perms, "users:delete") // admin only
	assert.NotContains(t, perms, "system:manage")

	// Verify no duplicates
	assert.Len(t, perms, 2) // "users:read" should appear only once

	// Get permissions for all roles
	allPerms := loader.GetPermissionsForRoles([]string{"admin", "editor", "viewer"})
	assert.Len(t, allPerms, 4) // unique permissions
}

// TestDSLLoaderMiddlewareConfig tests loading middleware configuration.
func TestDSLLoaderMiddlewareConfig(t *testing.T) {
	t.Parallel()

	input := `
middleware rate_limit {
	type rate_limiting
	config {
		requests: 100
		window: "1m"
		strategy: sliding_window
	}
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	mw, ok := loader.GetMiddleware("rate_limit")
	require.True(t, ok)

	assert.Equal(t, "rate_limiting", mw.MiddlewareType)
	assert.Equal(t, 100.0, mw.Config["requests"])
	assert.Equal(t, "1m", mw.Config["window"])
	assert.Equal(t, "sliding_window", mw.Config["strategy"])
}

// TestDSLLoaderDuplicateAuth tests that duplicate auth providers are rejected.
func TestDSLLoaderDuplicateAuth(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
}

auth jwt_provider {
	method oauth2
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate auth provider")
}

// TestDSLLoaderDuplicateRole tests that duplicate roles are rejected.
func TestDSLLoaderDuplicateRole(t *testing.T) {
	t.Parallel()

	input := `
role admin {
	permissions ["users:read"]
}

role admin {
	permissions ["users:write"]
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate role")
}

// TestDSLLoaderDuplicateMiddleware tests that duplicate middleware is rejected.
func TestDSLLoaderDuplicateMiddleware(t *testing.T) {
	t.Parallel()

	input := `
middleware auth_check {
	type authentication
}

middleware auth_check {
	type rate_limiting
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate middleware")
}

// TestDSLLoaderAllAuths tests retrieving all auth providers.
func TestDSLLoaderAllAuths(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
}

auth oauth_provider {
	method oauth2
}

auth apikey_provider {
	method apikey
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	allAuths := loader.AllAuths()
	assert.Len(t, allAuths, 3)
	assert.Contains(t, allAuths, "jwt_provider")
	assert.Contains(t, allAuths, "oauth_provider")
	assert.Contains(t, allAuths, "apikey_provider")
}

// TestDSLLoaderAllMiddlewares tests retrieving all middlewares.
func TestDSLLoaderAllMiddlewares(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
	jwks_url "https://auth.example.com/.well-known/jwks.json"
	issuer "https://auth.example.com"
	audience "api.example.com"
}

middleware auth_check {
	type authentication
	config {
		provider: jwt_provider
		required: true
	}
}

middleware rate_limit {
	type rate_limiting
	config {
		requests: 100
		window: "1m"
	}
}

middleware audit_log {
	type logging
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	allMW := loader.AllMiddlewares()
	assert.Len(t, allMW, 3)
	assert.Contains(t, allMW, "auth_check")
	assert.Contains(t, allMW, "rate_limit")
	assert.Contains(t, allMW, "audit_log")
}

// TestDSLLoaderFromExampleFile tests loading from the example file.
func TestDSLLoaderFromExampleFile(t *testing.T) {
	t.Parallel()

	prog, err := parser.ParseFile("../../examples/08-with-auth/with_auth.cai")
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	// Verify auth providers
	allAuths := loader.AllAuths()
	assert.Len(t, allAuths, 4)

	// Verify roles
	allRoles := loader.AllRoles()
	assert.Len(t, allRoles, 5)

	// Verify middlewares
	allMW := loader.AllMiddlewares()
	assert.Len(t, allMW, 7)

	// Verify specific role permissions
	assert.True(t, loader.RoleHasPermission("admin", "users:delete"))
	assert.True(t, loader.RoleHasPermission("admin", "system:manage"))
	assert.False(t, loader.RoleHasPermission("reader", "users:delete"))
}

// TestDSLLoaderAuthWithoutJWKS tests loading auth without JWKS config.
func TestDSLLoaderAuthWithoutJWKS(t *testing.T) {
	t.Parallel()

	input := `
auth basic_auth {
	method basic
}
`

	prog, err := parser.Parse(input)
	require.NoError(t, err)

	loader := NewDSLLoader()
	err = loader.LoadProgram(prog)
	require.NoError(t, err)

	auth, ok := loader.GetAuth("basic_auth")
	require.True(t, ok)
	assert.Equal(t, ast.AuthMethodBasic, auth.Method)
	assert.Nil(t, auth.JWKS)
	assert.Nil(t, auth.Config)
}

// TestLoadAuthConfig tests loading auth configuration directly.
func TestLoadAuthConfig(t *testing.T) {
	t.Parallel()

	auth := &ast.AuthDecl{
		Name:   "jwt_provider",
		Method: ast.AuthMethodJWT,
		JWKS: &ast.JWKSConfig{
			URL:      "https://auth.example.com/.well-known/jwks.json",
			Issuer:   "https://auth.example.com",
			Audience: "api.example.com",
		},
	}

	config, err := LoadAuthConfig(auth)
	if err != nil {
		t.Fatalf("Failed to load auth config: %v", err)
	}

	if config.Name != "jwt_provider" {
		t.Errorf("Expected provider 'jwt_provider', got '%s'", config.Name)
	}

	if config.Method != ast.AuthMethodJWT {
		t.Errorf("Expected method 'jwt', got '%s'", config.Method)
	}

	if config.JWKS == nil {
		t.Fatal("Expected JWKS config, got nil")
	}

	if config.JWKS.URL != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("Unexpected JWKS URL: %s", config.JWKS.URL)
	}

	if config.Config == nil {
		t.Fatal("Expected Config to be populated, got nil")
	}

	if config.Config.JWKSURL != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("Unexpected Config JWKS URL: %s", config.Config.JWKSURL)
	}
}

// TestLoadAuthConfigInvalidHTTP tests that HTTP JWKS URLs are rejected.
func TestLoadAuthConfigInvalidHTTP(t *testing.T) {
	t.Parallel()

	auth := &ast.AuthDecl{
		Name:   "jwt_provider",
		Method: ast.AuthMethodJWT,
		JWKS: &ast.JWKSConfig{
			URL:      "http://auth.example.com/.well-known/jwks.json",
			Issuer:   "https://auth.example.com",
			Audience: "api.example.com",
		},
	}

	_, err := LoadAuthConfig(auth)
	if err == nil {
		t.Error("Expected error for HTTP JWKS URL")
	}

	if !assert.Contains(t, err.Error(), "JWKS URL must use HTTPS") {
		t.Errorf("Expected HTTPS validation error, got: %v", err)
	}
}

// TestLoadRoleConfig tests loading role configuration directly.
func TestLoadRoleConfig(t *testing.T) {
	t.Parallel()

	role := &ast.RoleDecl{
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "users:delete"},
	}

	config, err := LoadRoleConfig(role)
	if err != nil {
		t.Fatalf("Failed to load role config: %v", err)
	}

	if config.Name != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", config.Name)
	}

	if len(config.Permissions) != 3 {
		t.Errorf("Expected 3 permissions, got %d", len(config.Permissions))
	}

	expectedPerms := []string{"users:read", "users:write", "users:delete"}
	for i, perm := range expectedPerms {
		if config.Permissions[i] != perm {
			t.Errorf("Expected permission '%s', got '%s'", perm, config.Permissions[i])
		}
	}
}

// TestLoadRoleConfigInvalidPermission tests invalid permission format validation.
func TestLoadRoleConfigInvalidPermission(t *testing.T) {
	t.Parallel()

	role := &ast.RoleDecl{
		Name:        "admin",
		Permissions: []string{"invalid_permission"},
	}

	_, err := LoadRoleConfig(role)
	if err == nil {
		t.Error("Expected error for invalid permission format")
	}

	if !assert.Contains(t, err.Error(), "invalid permission format") {
		t.Errorf("Expected permission format error, got: %v", err)
	}
}

// TestLoadRolesMap tests loading multiple roles into a map.
func TestLoadRolesMap(t *testing.T) {
	t.Parallel()

	roles := []*ast.RoleDecl{
		{
			Name:        "admin",
			Permissions: []string{"users:read", "users:write"},
		},
		{
			Name:        "viewer",
			Permissions: []string{"users:read"},
		},
	}

	roleMap, err := LoadRolesMap(roles)
	if err != nil {
		t.Fatalf("Failed to load roles map: %v", err)
	}

	if len(roleMap) != 2 {
		t.Errorf("Expected 2 roles in map, got %d", len(roleMap))
	}

	admin, ok := roleMap["admin"]
	if !ok {
		t.Error("Expected 'admin' role in map")
	}

	if len(admin.Permissions) != 2 {
		t.Errorf("Expected 2 permissions for admin, got %d", len(admin.Permissions))
	}

	viewer, ok := roleMap["viewer"]
	if !ok {
		t.Error("Expected 'viewer' role in map")
	}

	if len(viewer.Permissions) != 1 {
		t.Errorf("Expected 1 permission for viewer, got %d", len(viewer.Permissions))
	}
}

// TestLoadRolesMapDuplicate tests duplicate role detection.
func TestLoadRolesMapDuplicate(t *testing.T) {
	t.Parallel()

	roles := []*ast.RoleDecl{
		{
			Name:        "admin",
			Permissions: []string{"users:read"},
		},
		{
			Name:        "admin",
			Permissions: []string{"users:write"},
		},
	}

	_, err := LoadRolesMap(roles)
	if err == nil {
		t.Error("Expected error for duplicate role")
	}

	if !assert.Contains(t, err.Error(), "duplicate role") {
		t.Errorf("Expected duplicate role error, got: %v", err)
	}
}

// TestLoadAuthConfigNil tests nil input handling for auth config.
func TestLoadAuthConfigNil(t *testing.T) {
	t.Parallel()

	_, err := LoadAuthConfig(nil)
	if err == nil {
		t.Error("Expected error for nil auth")
	}

	if !assert.Contains(t, err.Error(), "auth declaration cannot be nil") {
		t.Errorf("Expected nil auth error, got: %v", err)
	}
}

// TestLoadRoleConfigNil tests nil input handling for role config.
func TestLoadRoleConfigNil(t *testing.T) {
	t.Parallel()

	_, err := LoadRoleConfig(nil)
	if err == nil {
		t.Error("Expected error for nil role")
	}

	if !assert.Contains(t, err.Error(), "role declaration cannot be nil") {
		t.Errorf("Expected nil role error, got: %v", err)
	}
}
