package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWKSCache caches RSA public keys fetched from a JWKS endpoint.
type JWKSCache struct {
	url         string
	keys        map[string]*rsa.PublicKey
	mu          sync.RWMutex
	refreshTTL  time.Duration
	lastRefresh time.Time
	client      HTTPClient
	logger      *slog.Logger
}

// HTTPClient is an interface for HTTP operations (for testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kid string `json:"kid"` // Key ID
	Kty string `json:"kty"` // Key Type (RSA, EC, etc.)
	Alg string `json:"alg"` // Algorithm
	Use string `json:"use"` // Usage (sig, enc)
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// NewJWKSCache creates a new JWKS cache with the given URL and refresh interval.
func NewJWKSCache(url string, refreshTTL time.Duration) *JWKSCache {
	return &JWKSCache{
		url:        url,
		keys:       make(map[string]*rsa.PublicKey),
		refreshTTL: refreshTTL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: slog.Default().With("component", "jwks-cache"),
	}
}

// NewJWKSCacheWithClient creates a new JWKS cache with a custom HTTP client.
func NewJWKSCacheWithClient(url string, refreshTTL time.Duration, client HTTPClient) *JWKSCache {
	c := NewJWKSCache(url, refreshTTL)
	c.client = client
	return c
}

// SetLogger sets a custom logger for the cache.
func (c *JWKSCache) SetLogger(logger *slog.Logger) {
	c.logger = logger.With("component", "jwks-cache")
}

// GetKey retrieves a public key by its key ID.
// If the key is not in the cache, it attempts to refresh from the JWKS endpoint.
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
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, kid)
	}

	return key, nil
}

// GetKeyByID retrieves a public key by its key ID without triggering a refresh.
func (c *JWKSCache) GetKeyByID(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.keys[kid]
	return key, ok
}

// Refresh fetches the JWKS from the configured URL and updates the cache.
func (c *JWKSCache) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: endpoint returned %d", ErrJWKSFetchFailed, resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSDecodeFailed, err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			c.logger.Debug("skipping non-RSA key", "kid", jwk.Kid, "kty", jwk.Kty)
			continue
		}

		key, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			c.logger.Warn("failed to parse JWK", "kid", jwk.Kid, "error", err)
			continue
		}

		keys[jwk.Kid] = key
	}

	c.mu.Lock()
	c.keys = keys
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	c.logger.Debug("refreshed JWKS", "key_count", len(keys))
	return nil
}

// StartRefreshLoop starts a background goroutine that periodically refreshes the cache.
// The loop runs until the context is canceled.
func (c *JWKSCache) StartRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(c.refreshTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Refresh(ctx); err != nil {
				c.logger.Error("failed to refresh JWKS", "error", err)
			}
		}
	}
}

// LastRefresh returns the time of the last successful refresh.
func (c *JWKSCache) LastRefresh() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}

// KeyCount returns the number of keys currently in the cache.
func (c *JWKSCache) KeyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keys)
}

// jwkToRSAPublicKey converts a JWK to an RSA public key.
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	if jwk.N == "" || jwk.E == "" {
		return nil, fmt.Errorf("missing modulus or exponent")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}
