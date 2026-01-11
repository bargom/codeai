package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user extracted from a JWT.
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

// HasRole checks if the user has the specified role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has the specified permission.
func (u *User) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Config holds JWT validation configuration.
type Config struct {
	Issuer     string // Expected issuer (iss claim)
	Audience   string // Expected audience (aud claim)
	Secret     string // Secret key for HS256/HS384/HS512
	PublicKey  string // PEM-encoded public key for RS256/RS384/RS512
	JWKSURL    string // URL to fetch JWKS from
	RolesClaim string // Claim name containing roles (default: "roles")
	PermsClaim string // Claim name containing permissions
}

// Validator validates JWTs and extracts user information.
type Validator struct {
	config     Config
	publicKeys map[string]*rsa.PublicKey
	jwksCache  *JWKSCache
	mu         sync.RWMutex
	logger     *slog.Logger
}

// NewValidator creates a new JWT validator with the given configuration.
func NewValidator(config Config) (*Validator, error) {
	v := &Validator{
		config:     config,
		publicKeys: make(map[string]*rsa.PublicKey),
		logger:     slog.Default().With("component", "jwt-validator"),
	}

	// Load static public key if provided
	if config.PublicKey != "" {
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("invalid public key: %w", err)
		}
		v.publicKeys["default"] = key
	}

	// Initialize JWKS cache if URL provided
	if config.JWKSURL != "" {
		v.jwksCache = NewJWKSCache(config.JWKSURL, 5*time.Minute)
		if err := v.jwksCache.Refresh(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to load JWKS: %w", err)
		}
	}

	return v, nil
}

// NewValidatorWithLogger creates a new JWT validator with a custom logger.
func NewValidatorWithLogger(config Config, logger *slog.Logger) (*Validator, error) {
	v, err := NewValidator(config)
	if err != nil {
		return nil, err
	}
	if logger != nil {
		v.logger = logger.With("component", "jwt-validator")
	}
	return v, nil
}

// SetJWKSCache sets a custom JWKS cache (useful for testing).
func (v *Validator) SetJWKSCache(cache *JWKSCache) {
	v.jwksCache = cache
}

// ValidateToken validates a JWT string and returns the extracted user.
func (v *Validator) ValidateToken(ctx context.Context, tokenStr string) (*User, error) {
	if tokenStr == "" {
		return nil, ErrMissingToken
	}

	// Parse token without validation first to get headers
	token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		v.logger.Debug("failed to parse token", "error", err)
		return nil, ErrInvalidToken
	}

	// Get key ID from header
	kid, _ := token.Header["kid"].(string)

	// Parse with validation
	token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return v.getKey(ctx, kid, t.Method.Alg())
	})
	if err != nil {
		v.logger.Debug("token validation failed", "error", err)
		if isExpiredError(err) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer
	if v.config.Issuer != "" {
		iss, _ := claims.GetIssuer()
		if iss != v.config.Issuer {
			return nil, ErrInvalidIssuer
		}
	}

	// Validate audience
	if v.config.Audience != "" {
		aud, _ := claims.GetAudience()
		if !containsAudience(aud, v.config.Audience) {
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
	rolesClaim := v.config.RolesClaim
	if rolesClaim == "" {
		rolesClaim = "roles"
	}
	user.Roles = getStringListClaim(claims, rolesClaim)

	// Extract permissions
	if v.config.PermsClaim != "" {
		user.Permissions = getStringListClaim(claims, v.config.PermsClaim)
	}

	return user, nil
}

// getKey retrieves the appropriate signing key based on algorithm and key ID.
func (v *Validator) getKey(ctx context.Context, kid, alg string) (interface{}, error) {
	switch alg {
	case "HS256", "HS384", "HS512":
		if v.config.Secret == "" {
			return nil, ErrNoSecretConfigured
		}
		return []byte(v.config.Secret), nil

	case "RS256", "RS384", "RS512":
		// Try JWKS first
		if v.jwksCache != nil {
			key, err := v.jwksCache.GetKey(ctx, kid)
			if err == nil {
				return key, nil
			}
			v.logger.Debug("JWKS key lookup failed", "kid", kid, "error", err)
		}

		// Fall back to static key
		v.mu.RLock()
		defer v.mu.RUnlock()

		if kid != "" {
			if key, ok := v.publicKeys[kid]; ok {
				return key, nil
			}
		}

		if key, ok := v.publicKeys["default"]; ok {
			return key, nil
		}

		return nil, ErrNoPublicKeyConfigured

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, alg)
	}
}

// StartJWKSRefresh starts a background goroutine that periodically refreshes the JWKS.
func (v *Validator) StartJWKSRefresh(ctx context.Context) {
	if v.jwksCache != nil {
		go v.jwksCache.StartRefreshLoop(ctx)
	}
}

// isExpiredError checks if the error indicates an expired token.
func isExpiredError(err error) bool {
	return strings.Contains(err.Error(), "token is expired")
}

// containsAudience checks if the expected audience is in the audience list.
func containsAudience(aud jwt.ClaimStrings, expected string) bool {
	for _, a := range aud {
		if a == expected {
			return true
		}
	}
	return false
}

// getStringClaim extracts a string claim from the claims map.
func getStringClaim(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

// getStringListClaim extracts a string list claim from the claims map.
// Handles []interface{}, []string, and space-separated string formats.
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
		// Handle space-separated values
		result = strings.Fields(v)
	}

	return result
}
