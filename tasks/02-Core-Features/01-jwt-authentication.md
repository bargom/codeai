# Task: JWT Authentication Module

## Overview
Implement JWT (JSON Web Token) authentication module that validates tokens, extracts user information, and integrates with CodeAI's declarative auth syntax.

## Phase
Phase 2: Core Features

## Priority
Critical - Foundation for all authenticated endpoints.

## Dependencies
- Phase 1 complete (HTTP Module, Runtime Engine)

## Description
Create a comprehensive JWT authentication system supporting RS256/HS256 algorithms, JWKS key rotation, token validation, and seamless integration with CodeAI endpoint declarations.

## Detailed Requirements

### 1. Auth Module Interface (internal/modules/auth/module.go)

```go
package auth

import (
    "context"
    "crypto/rsa"
    "sync"
    "time"
)

type AuthModule interface {
    Module
    ValidateToken(ctx context.Context, token string) (*User, error)
    HasPermission(user *User, permission string) bool
    HasRole(user *User, role string) bool
}

type User struct {
    ID          string
    Email       string
    Name        string
    Roles       []string
    Permissions []string
    Claims      map[string]any
    Token       string
    ExpiresAt   time.Time
}

func (u *User) HasRole(role string) bool {
    for _, r := range u.Roles {
        if r == role {
            return true
        }
    }
    return false
}

type AuthConfig struct {
    Type        string   // "jwt", "api_key", "basic"
    Issuer      string
    Audience    string
    Secret      string   // For HS256
    PublicKey   string   // For RS256 (PEM)
    JWKSURL     string   // For dynamic key loading
    RolesClaim  string   // Claim containing roles (default: "roles")
    PermsClaim  string   // Claim containing permissions
}
```

### 2. JWT Implementation (internal/modules/auth/jwt.go)

```go
package auth

import (
    "context"
    "crypto/rsa"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type JWTAuthModule struct {
    config     AuthConfig
    publicKeys map[string]*rsa.PublicKey
    jwksCache  *JWKSCache
    mu         sync.RWMutex
    logger     *slog.Logger
}

func NewJWTAuthModule(config AuthConfig) (*JWTAuthModule, error) {
    m := &JWTAuthModule{
        config:     config,
        publicKeys: make(map[string]*rsa.PublicKey),
        logger:     slog.Default().With("module", "jwt-auth"),
    }

    // Load static public key if provided
    if config.PublicKey != "" {
        key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.PublicKey))
        if err != nil {
            return nil, fmt.Errorf("invalid public key: %w", err)
        }
        m.publicKeys["default"] = key
    }

    // Initialize JWKS cache if URL provided
    if config.JWKSURL != "" {
        m.jwksCache = NewJWKSCache(config.JWKSURL, 15*time.Minute)
        if err := m.jwksCache.Refresh(context.Background()); err != nil {
            return nil, fmt.Errorf("failed to load JWKS: %w", err)
        }
    }

    return m, nil
}

func (m *JWTAuthModule) Name() string { return "jwt-auth" }

func (m *JWTAuthModule) Initialize(config *Config) error {
    return nil
}

func (m *JWTAuthModule) Start(ctx context.Context) error {
    // Start JWKS refresh goroutine if configured
    if m.jwksCache != nil {
        go m.jwksCache.StartRefreshLoop(ctx)
    }
    return nil
}

func (m *JWTAuthModule) Stop(ctx context.Context) error {
    return nil
}

func (m *JWTAuthModule) Health() HealthStatus {
    return HealthStatus{Status: "healthy"}
}

func (m *JWTAuthModule) ValidateToken(ctx context.Context, tokenStr string) (*User, error) {
    // Parse token without validation first to get headers
    token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
    if err != nil {
        return nil, ErrInvalidToken
    }

    // Get key ID from header
    kid, _ := token.Header["kid"].(string)

    // Parse with validation
    token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        return m.getKey(ctx, kid, t.Method.Alg())
    })
    if err != nil {
        m.logger.Debug("token validation failed", "error", err)
        return nil, ErrInvalidToken
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    // Validate issuer
    if m.config.Issuer != "" {
        if !claims.VerifyIssuer(m.config.Issuer, true) {
            return nil, ErrInvalidIssuer
        }
    }

    // Validate audience
    if m.config.Audience != "" {
        if !claims.VerifyAudience(m.config.Audience, true) {
            return nil, ErrInvalidAudience
        }
    }

    // Extract user info
    user := &User{
        ID:     getStringClaim(claims, "sub"),
        Email:  getStringClaim(claims, "email"),
        Name:   getStringClaim(claims, "name"),
        Claims: claims,
        Token:  tokenStr,
    }

    // Extract expiry
    if exp, err := claims.GetExpirationTime(); err == nil && exp != nil {
        user.ExpiresAt = exp.Time
    }

    // Extract roles
    rolesClaim := m.config.RolesClaim
    if rolesClaim == "" {
        rolesClaim = "roles"
    }
    user.Roles = getStringListClaim(claims, rolesClaim)

    // Extract permissions
    if m.config.PermsClaim != "" {
        user.Permissions = getStringListClaim(claims, m.config.PermsClaim)
    }

    return user, nil
}

func (m *JWTAuthModule) getKey(ctx context.Context, kid, alg string) (interface{}, error) {
    switch alg {
    case "HS256", "HS384", "HS512":
        if m.config.Secret == "" {
            return nil, fmt.Errorf("no secret configured for %s", alg)
        }
        return []byte(m.config.Secret), nil

    case "RS256", "RS384", "RS512":
        // Try JWKS first
        if m.jwksCache != nil {
            key, err := m.jwksCache.GetKey(ctx, kid)
            if err == nil {
                return key, nil
            }
        }

        // Fall back to static key
        m.mu.RLock()
        defer m.mu.RUnlock()

        if kid != "" {
            if key, ok := m.publicKeys[kid]; ok {
                return key, nil
            }
        }

        if key, ok := m.publicKeys["default"]; ok {
            return key, nil
        }

        return nil, fmt.Errorf("no key found for kid: %s", kid)

    default:
        return nil, fmt.Errorf("unsupported algorithm: %s", alg)
    }
}

func getStringClaim(claims jwt.MapClaims, key string) string {
    if v, ok := claims[key].(string); ok {
        return v
    }
    return ""
}

func getStringListClaim(claims jwt.MapClaims, key string) []string {
    var result []string

    switch v := claims[key].(type) {
    case []interface{}:
        for _, item := range v {
            if s, ok := item.(string); ok {
                result = append(result, s)
            }
        }
    case []string:
        result = v
    case string:
        // Handle space-separated roles
        result = strings.Fields(v)
    }

    return result
}
```

### 3. JWKS Cache (internal/modules/auth/jwks.go)

```go
package auth

import (
    "context"
    "crypto/rsa"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "math/big"
    "net/http"
    "sync"
    "time"
)

type JWKSCache struct {
    url        string
    keys       map[string]*rsa.PublicKey
    mu         sync.RWMutex
    refreshTTL time.Duration
    lastRefresh time.Time
    client     *http.Client
}

type JWKS struct {
    Keys []JWK `json:"keys"`
}

type JWK struct {
    Kid string `json:"kid"`
    Kty string `json:"kty"`
    Alg string `json:"alg"`
    Use string `json:"use"`
    N   string `json:"n"`
    E   string `json:"e"`
}

func NewJWKSCache(url string, refreshTTL time.Duration) *JWKSCache {
    return &JWKSCache{
        url:        url,
        keys:       make(map[string]*rsa.PublicKey),
        refreshTTL: refreshTTL,
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *JWKSCache) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
    c.mu.RLock()
    key, ok := c.keys[kid]
    c.mu.RUnlock()

    if ok {
        return key, nil
    }

    // Key not found, try refreshing
    if err := c.Refresh(ctx); err != nil {
        return nil, err
    }

    c.mu.RLock()
    key, ok = c.keys[kid]
    c.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("key not found: %s", kid)
    }

    return key, nil
}

func (c *JWKSCache) Refresh(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
    if err != nil {
        return err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to fetch JWKS: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
    }

    var jwks JWKS
    if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
        return fmt.Errorf("failed to decode JWKS: %w", err)
    }

    keys := make(map[string]*rsa.PublicKey)
    for _, jwk := range jwks.Keys {
        if jwk.Kty != "RSA" {
            continue
        }

        key, err := jwkToRSAPublicKey(jwk)
        if err != nil {
            continue
        }

        keys[jwk.Kid] = key
    }

    c.mu.Lock()
    c.keys = keys
    c.lastRefresh = time.Now()
    c.mu.Unlock()

    return nil
}

func (c *JWKSCache) StartRefreshLoop(ctx context.Context) {
    ticker := time.NewTicker(c.refreshTTL)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := c.Refresh(ctx); err != nil {
                slog.Error("failed to refresh JWKS", "error", err)
            }
        }
    }
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
    nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
    if err != nil {
        return nil, err
    }

    eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
    if err != nil {
        return nil, err
    }

    n := new(big.Int).SetBytes(nBytes)
    e := int(new(big.Int).SetBytes(eBytes).Int64())

    return &rsa.PublicKey{N: n, E: e}, nil
}
```

### 4. HTTP Middleware (internal/modules/auth/middleware.go)

```go
package auth

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const userContextKey contextKey = "user"

func UserFromContext(ctx context.Context) *User {
    if user, ok := ctx.Value(userContextKey).(*User); ok {
        return user
    }
    return nil
}

func ContextWithUser(ctx context.Context, user *User) context.Context {
    return context.WithValue(ctx, userContextKey, user)
}

// AuthMiddleware creates authentication middleware
func (m *JWTAuthModule) AuthMiddleware(requirement AuthRequirement) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r)

            if token == "" {
                if requirement == AuthRequired {
                    writeError(w, http.StatusUnauthorized, "authentication required")
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            user, err := m.ValidateToken(r.Context(), token)
            if err != nil {
                if requirement == AuthRequired {
                    writeError(w, http.StatusUnauthorized, "invalid token")
                    return
                }
                next.ServeHTTP(w, r)
                return
            }

            ctx := ContextWithUser(r.Context(), user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractToken(r *http.Request) string {
    // Check Authorization header
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }

    // Check query parameter (for WebSocket connections)
    if token := r.URL.Query().Get("token"); token != "" {
        return token
    }

    // Check cookie
    if cookie, err := r.Cookie("token"); err == nil {
        return cookie.Value
    }

    return ""
}

func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type AuthRequirement string

const (
    AuthRequired AuthRequirement = "required"
    AuthOptional AuthRequirement = "optional"
    AuthPublic   AuthRequirement = "public"
)
```

### 5. Error Types

```go
package auth

import "errors"

var (
    ErrInvalidToken    = errors.New("invalid token")
    ErrExpiredToken    = errors.New("token has expired")
    ErrInvalidIssuer   = errors.New("invalid token issuer")
    ErrInvalidAudience = errors.New("invalid token audience")
    ErrMissingToken    = errors.New("missing authentication token")
)
```

## Acceptance Criteria
- [ ] JWT validation with HS256 and RS256
- [ ] JWKS endpoint support with caching
- [ ] Token extraction from header, query, cookie
- [ ] Claims extraction (roles, permissions)
- [ ] Middleware for HTTP endpoints
- [ ] Context propagation of user info
- [ ] Graceful handling of expired tokens
- [ ] Comprehensive error messages

## Testing Strategy
- Unit tests for token validation
- Unit tests for JWKS caching
- Integration tests with real JWTs
- Performance tests for token validation

## Files to Create
- `internal/modules/auth/module.go`
- `internal/modules/auth/jwt.go`
- `internal/modules/auth/jwks.go`
- `internal/modules/auth/middleware.go`
- `internal/modules/auth/errors.go`
- `internal/modules/auth/jwt_test.go`
