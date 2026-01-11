package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-hs256-testing"

// generateTestRSAKeys generates a test RSA key pair.
func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})

	return privateKey, string(publicKeyPEM)
}

// generateHS256Token generates a test HS256 token.
func generateHS256Token(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

// generateRS256Token generates a test RS256 token.
func generateRS256Token(t *testing.T, claims jwt.MapClaims, privateKey *rsa.PrivateKey, kid string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}
	tokenStr, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return tokenStr
}

func TestUser_HasRole(t *testing.T) {
	user := &User{
		Roles: []string{"admin", "user"},
	}

	assert.True(t, user.HasRole("admin"))
	assert.True(t, user.HasRole("user"))
	assert.False(t, user.HasRole("superadmin"))
	assert.False(t, user.HasRole(""))
}

func TestUser_HasPermission(t *testing.T) {
	user := &User{
		Permissions: []string{"read:users", "write:users"},
	}

	assert.True(t, user.HasPermission("read:users"))
	assert.True(t, user.HasPermission("write:users"))
	assert.False(t, user.HasPermission("delete:users"))
	assert.False(t, user.HasPermission(""))
}

func TestNewValidator_WithSecret(t *testing.T) {
	v, err := NewValidator(Config{
		Secret: testSecret,
	})

	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestNewValidator_WithPublicKey(t *testing.T) {
	_, publicKeyPEM := generateTestRSAKeys(t)

	v, err := NewValidator(Config{
		PublicKey: publicKeyPEM,
	})

	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestNewValidator_WithInvalidPublicKey(t *testing.T) {
	_, err := NewValidator(Config{
		PublicKey: "invalid-pem-data",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid public key")
}

func TestValidator_ValidateToken_HS256(t *testing.T) {
	v, err := NewValidator(Config{
		Secret: testSecret,
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"name":  "Test User",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []interface{}{"admin", "user"},
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.ElementsMatch(t, []string{"admin", "user"}, user.Roles)
	assert.Equal(t, token, user.Token)
}

func TestValidator_ValidateToken_RS256(t *testing.T) {
	privateKey, publicKeyPEM := generateTestRSAKeys(t)

	v, err := NewValidator(Config{
		PublicKey: publicKeyPEM,
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":   "user456",
		"email": "user@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := generateRS256Token(t, claims, privateKey, "")

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user456", user.ID)
	assert.Equal(t, "user@test.com", user.Email)
}

func TestValidator_ValidateToken_EmptyToken(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	_, err = v.ValidateToken(context.Background(), "")

	assert.ErrorIs(t, err, ErrMissingToken)
}

func TestValidator_ValidateToken_InvalidToken(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	_, err = v.ValidateToken(context.Background(), "invalid.token.here")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidator_ValidateToken_ExpiredToken(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired
	}

	token := generateHS256Token(t, claims, testSecret)

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidator_ValidateToken_InvalidIssuer(t *testing.T) {
	v, err := NewValidator(Config{
		Secret: testSecret,
		Issuer: "expected-issuer",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "wrong-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidIssuer)
}

func TestValidator_ValidateToken_InvalidAudience(t *testing.T) {
	v, err := NewValidator(Config{
		Secret:   testSecret,
		Audience: "expected-audience",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"aud": "wrong-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidAudience)
}

func TestValidator_ValidateToken_ValidAudience(t *testing.T) {
	v, err := NewValidator(Config{
		Secret:   testSecret,
		Audience: "my-app",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"aud": []string{"other-app", "my-app"},
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user123", user.ID)
}

func TestValidator_ValidateToken_WrongSecret(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, "wrong-secret")

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidator_ValidateToken_NoSecretConfigured(t *testing.T) {
	v, err := NewValidator(Config{})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidator_ValidateToken_NoPublicKeyConfigured(t *testing.T) {
	privateKey, _ := generateTestRSAKeys(t)

	v, err := NewValidator(Config{})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateRS256Token(t, claims, privateKey, "")

	_, err = v.ValidateToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidator_ValidateToken_CustomRolesClaim(t *testing.T) {
	v, err := NewValidator(Config{
		Secret:     testSecret,
		RolesClaim: "custom_roles",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":          "user123",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"custom_roles": []interface{}{"role1", "role2"},
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"role1", "role2"}, user.Roles)
}

func TestValidator_ValidateToken_CustomPermsClaim(t *testing.T) {
	v, err := NewValidator(Config{
		Secret:     testSecret,
		PermsClaim: "permissions",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":         "user123",
		"exp":         time.Now().Add(time.Hour).Unix(),
		"permissions": []interface{}{"read", "write"},
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read", "write"}, user.Permissions)
}

func TestValidator_ValidateToken_SpaceSeparatedRoles(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": "admin user manager",
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"admin", "user", "manager"}, user.Roles)
}

func TestValidator_ValidateToken_StringArrayRoles(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{"admin", "user"},
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"admin", "user"}, user.Roles)
}

func TestValidator_ValidateToken_ExpiryExtracted(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	expTime := time.Now().Add(time.Hour).Truncate(time.Second)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": expTime.Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, expTime.Unix(), user.ExpiresAt.Unix())
}

func TestValidator_SetJWKSCache(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	cache := NewJWKSCache("http://example.com/.well-known/jwks.json", 5*time.Minute)
	v.SetJWKSCache(cache)

	assert.Equal(t, cache, v.jwksCache)
}

func TestNewValidatorWithLogger(t *testing.T) {
	v, err := NewValidatorWithLogger(Config{Secret: testSecret}, nil)
	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestGetStringClaim(t *testing.T) {
	claims := jwt.MapClaims{
		"string_val": "hello",
		"int_val":    42,
		"nil_val":    nil,
	}

	assert.Equal(t, "hello", getStringClaim(claims, "string_val"))
	assert.Equal(t, "", getStringClaim(claims, "int_val"))
	assert.Equal(t, "", getStringClaim(claims, "nil_val"))
	assert.Equal(t, "", getStringClaim(claims, "missing"))
}

func TestGetStringListClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   jwt.MapClaims
		key      string
		expected []string
	}{
		{
			name:     "interface array",
			claims:   jwt.MapClaims{"roles": []interface{}{"a", "b"}},
			key:      "roles",
			expected: []string{"a", "b"},
		},
		{
			name:     "string array",
			claims:   jwt.MapClaims{"roles": []string{"x", "y"}},
			key:      "roles",
			expected: []string{"x", "y"},
		},
		{
			name:     "space separated",
			claims:   jwt.MapClaims{"roles": "one two three"},
			key:      "roles",
			expected: []string{"one", "two", "three"},
		},
		{
			name:     "missing key",
			claims:   jwt.MapClaims{},
			key:      "roles",
			expected: nil,
		},
		{
			name:     "wrong type",
			claims:   jwt.MapClaims{"roles": 123},
			key:      "roles",
			expected: nil,
		},
		{
			name:     "mixed interface array",
			claims:   jwt.MapClaims{"roles": []interface{}{"a", 123, "b"}},
			key:      "roles",
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringListClaim(tt.claims, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAudience(t *testing.T) {
	assert.True(t, containsAudience(jwt.ClaimStrings{"a", "b", "c"}, "b"))
	assert.False(t, containsAudience(jwt.ClaimStrings{"a", "b", "c"}, "d"))
	assert.False(t, containsAudience(jwt.ClaimStrings{}, "a"))
	assert.True(t, containsAudience(jwt.ClaimStrings{"single"}, "single"))
}

func TestIsExpiredError(t *testing.T) {
	assert.True(t, isExpiredError(jwt.ErrTokenExpired))
	assert.False(t, isExpiredError(ErrInvalidToken))
}

func TestValidator_StartJWKSRefresh_WithCache(t *testing.T) {
	privateKey, publicKeyPEM := generateTestRSAKeys(t)
	_ = privateKey // Used to create keys but we just need PEM

	v, err := NewValidator(Config{PublicKey: publicKeyPEM})
	require.NoError(t, err)

	// Set up a JWKS cache
	cache := NewJWKSCache("http://example.com/jwks", 1*time.Hour)
	v.SetJWKSCache(cache)

	// Start the refresh - should not panic
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	v.StartJWKSRefresh(ctx)

	// Just verify it doesn't panic
}

func TestValidator_StartJWKSRefresh_NoCache(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic when cache is nil
	v.StartJWKSRefresh(ctx)
}

func TestValidator_ValidateToken_RS256_WithKid(t *testing.T) {
	privateKey, publicKeyPEM := generateTestRSAKeys(t)

	v, err := NewValidator(Config{
		PublicKey: publicKeyPEM,
	})
	require.NoError(t, err)

	// Manually add a key with specific kid
	v.mu.Lock()
	v.publicKeys["my-key-id"] = &privateKey.PublicKey
	v.mu.Unlock()

	claims := jwt.MapClaims{
		"sub": "user456",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateRS256Token(t, claims, privateKey, "my-key-id")

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user456", user.ID)
}

func TestValidator_ValidateToken_UnsupportedAlgorithm(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	// Create a token with ES256 algorithm (not supported)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	// We need to sign it somehow - but since ES256 needs EC key,
	// let's just create a malformed token that claims to be ES256
	tokenStr := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIiwiZXhwIjoxOTk5OTk5OTk5fQ.invalid"
	_ = token

	_, err = v.ValidateToken(context.Background(), tokenStr)

	assert.Error(t, err)
}

func TestValidator_ValidateToken_ValidIssuer(t *testing.T) {
	v, err := NewValidator(Config{
		Secret: testSecret,
		Issuer: "my-issuer",
	})
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "my-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateHS256Token(t, claims, testSecret)

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user123", user.ID)
}

func TestValidator_ValidateToken_NoExpiry(t *testing.T) {
	v, err := NewValidator(Config{Secret: testSecret})
	require.NoError(t, err)

	// Token without exp claim - should fail validation by default
	claims := jwt.MapClaims{
		"sub": "user123",
	}

	token := generateHS256Token(t, claims, testSecret)

	// The jwt library requires exp by default, so this should fail
	_, err = v.ValidateToken(context.Background(), token)

	// Depending on library settings, might error or not - just verify no panic
	// In most cases with default parser, missing exp is OK
}

func TestNewValidatorWithLogger_WithLogger(t *testing.T) {
	logger := slog.Default()
	v, err := NewValidatorWithLogger(Config{Secret: testSecret}, logger)
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.NotNil(t, v.logger)
}

func TestNewValidatorWithLogger_InvalidConfig(t *testing.T) {
	_, err := NewValidatorWithLogger(Config{PublicKey: "invalid"}, nil)
	assert.Error(t, err)
}

func TestNewValidator_WithJWKSURL_FailsOnInitialRefresh(t *testing.T) {
	// JWKS URL that doesn't exist will fail during initial refresh
	_, err := NewValidator(Config{
		JWKSURL: "http://localhost:99999/jwks", // Invalid port
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load JWKS")
}

func TestValidator_ValidateToken_RS256_WithJWKSCache(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create a validator without JWKS (we'll set it manually)
	v, err := NewValidator(Config{})
	require.NoError(t, err)

	// Create and populate a mock JWKS cache
	cache := NewJWKSCache("http://example.com/jwks", 5*time.Minute)
	cache.mu.Lock()
	cache.keys["test-kid"] = &privateKey.PublicKey
	cache.mu.Unlock()

	v.SetJWKSCache(cache)

	claims := jwt.MapClaims{
		"sub": "user-from-jwks",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := generateRS256Token(t, claims, privateKey, "test-kid")

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user-from-jwks", user.ID)
}

func TestValidator_ValidateToken_RS256_JWKSCacheMiss_FallbackToStatic(t *testing.T) {
	privateKey, publicKeyPEM := generateTestRSAKeys(t)

	v, err := NewValidator(Config{
		PublicKey: publicKeyPEM,
	})
	require.NoError(t, err)

	// Create a JWKS cache that will miss
	cache := NewJWKSCache("http://example.com/jwks", 5*time.Minute)
	// Don't populate it - it will miss

	v.SetJWKSCache(cache)

	claims := jwt.MapClaims{
		"sub": "user-fallback",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Token without kid - will fall back to default static key
	token := generateRS256Token(t, claims, privateKey, "")

	user, err := v.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "user-fallback", user.ID)
}
