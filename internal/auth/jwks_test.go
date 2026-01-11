package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient is a mock HTTP client for testing.
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// generateTestJWKS generates a test JWKS response.
func generateTestJWKS(t *testing.T, keys map[string]*rsa.PublicKey) []byte {
	t.Helper()

	jwks := JWKS{Keys: make([]JWK, 0, len(keys))}

	for kid, key := range keys {
		jwk := JWK{
			Kid: kid,
			Kty: "RSA",
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), // 65537
		}
		jwks.Keys = append(jwks.Keys, jwk)
	}

	data, err := json.Marshal(jwks)
	require.NoError(t, err)
	return data
}

func TestNewJWKSCache(t *testing.T) {
	cache := NewJWKSCache("https://example.com/.well-known/jwks.json", 5*time.Minute)

	assert.Equal(t, "https://example.com/.well-known/jwks.json", cache.url)
	assert.Equal(t, 5*time.Minute, cache.refreshTTL)
	assert.NotNil(t, cache.keys)
	assert.NotNil(t, cache.client)
}

func TestNewJWKSCacheWithClient(t *testing.T) {
	mockClient := &mockHTTPClient{}
	cache := NewJWKSCacheWithClient("https://example.com/jwks", 10*time.Minute, mockClient)

	assert.Equal(t, mockClient, cache.client)
}

func TestJWKSCache_SetLogger(t *testing.T) {
	cache := NewJWKSCache("https://example.com/jwks", 5*time.Minute)
	logger := slog.Default()

	cache.SetLogger(logger)

	assert.NotNil(t, cache.logger)
}

func TestJWKSCache_Refresh_Success(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{
		"key1": &privateKey.PublicKey,
	}
	jwksData := generateTestJWKS(t, keys)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	err = cache.Refresh(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, cache.KeyCount())

	key, ok := cache.GetKeyByID("key1")
	assert.True(t, ok)
	assert.NotNil(t, key)
}

func TestJWKSCache_Refresh_HTTPError(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	err := cache.Refresh(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJWKSFetchFailed)
}

func TestJWKSCache_Refresh_Non200Status(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte("internal error"))),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	err := cache.Refresh(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJWKSFetchFailed)
}

func TestJWKSCache_Refresh_InvalidJSON(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("not json"))),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	err := cache.Refresh(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJWKSDecodeFailed)
}

func TestJWKSCache_Refresh_SkipsNonRSAKeys(t *testing.T) {
	jwks := JWKS{
		Keys: []JWK{
			{Kid: "ec-key", Kty: "EC", Alg: "ES256"},
		},
	}
	jwksData, _ := json.Marshal(jwks)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	err := cache.Refresh(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, cache.KeyCount())
}

func TestJWKSCache_GetKey_CacheHit(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{
		"key1": &privateKey.PublicKey,
	}
	jwksData := generateTestJWKS(t, keys)

	callCount := 0
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	// First call triggers refresh
	key1, err := cache.GetKey(context.Background(), "key1")
	require.NoError(t, err)
	assert.NotNil(t, key1)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	key2, err := cache.GetKey(context.Background(), "key1")
	require.NoError(t, err)
	assert.NotNil(t, key2)
	assert.Equal(t, 1, callCount) // No additional HTTP call
}

func TestJWKSCache_GetKey_NotFound(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{
		"key1": &privateKey.PublicKey,
	}
	jwksData := generateTestJWKS(t, keys)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	_, err = cache.GetKey(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestJWKSCache_GetKeyByID(t *testing.T) {
	cache := NewJWKSCache("https://example.com/jwks", 5*time.Minute)

	// Empty cache
	key, ok := cache.GetKeyByID("key1")
	assert.False(t, ok)
	assert.Nil(t, key)
}

func TestJWKSCache_LastRefresh(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{"key1": &privateKey.PublicKey}
	jwksData := generateTestJWKS(t, keys)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	assert.True(t, cache.LastRefresh().IsZero())

	beforeRefresh := time.Now()
	err = cache.Refresh(context.Background())
	require.NoError(t, err)

	assert.False(t, cache.LastRefresh().IsZero())
	assert.True(t, cache.LastRefresh().After(beforeRefresh) || cache.LastRefresh().Equal(beforeRefresh))
}

func TestJWKSCache_KeyCount(t *testing.T) {
	cache := NewJWKSCache("https://example.com/jwks", 5*time.Minute)
	assert.Equal(t, 0, cache.KeyCount())
}

func TestJWKSCache_StartRefreshLoop_CancelledContext(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{"key1": &privateKey.PublicKey}
	jwksData := generateTestJWKS(t, keys)

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksData)),
			}, nil
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 10*time.Millisecond, mockClient)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		cache.StartRefreshLoop(ctx)
		close(done)
	}()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Cancel and wait for exit
	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("refresh loop did not exit after context cancellation")
	}
}

func TestJWKToRSAPublicKey_Success(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
	}

	key, err := jwkToRSAPublicKey(jwk)

	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestJWKToRSAPublicKey_MissingModulus(t *testing.T) {
	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   "",
		E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
	}

	_, err := jwkToRSAPublicKey(jwk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing modulus")
}

func TestJWKToRSAPublicKey_MissingExponent(t *testing.T) {
	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}),
		E:   "",
	}

	_, err := jwkToRSAPublicKey(jwk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing modulus or exponent")
}

func TestJWKToRSAPublicKey_InvalidModulusEncoding(t *testing.T) {
	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   "!!!invalid-base64!!!",
		E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
	}

	_, err := jwkToRSAPublicKey(jwk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode modulus")
}

func TestJWKToRSAPublicKey_InvalidExponentEncoding(t *testing.T) {
	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}),
		E:   "!!!invalid-base64!!!",
	}

	_, err := jwkToRSAPublicKey(jwk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode exponent")
}

func TestJWKSCache_Integration_WithHTTPServer(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := map[string]*rsa.PublicKey{
		"integration-key": &privateKey.PublicKey,
	}
	jwksData := generateTestJWKS(t, keys)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksData)
	}))
	defer server.Close()

	cache := NewJWKSCache(server.URL, 5*time.Minute)

	err = cache.Refresh(context.Background())
	require.NoError(t, err)

	key, err := cache.GetKey(context.Background(), "integration-key")
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestJWKSCache_GetKey_RefreshError(t *testing.T) {
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 5*time.Minute, mockClient)

	_, err := cache.GetKey(context.Background(), "some-key")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJWKSFetchFailed)
}

func TestJWKSCache_StartRefreshLoop_RefreshError(t *testing.T) {
	callCount := 0
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return nil, errors.New("network error")
		},
	}

	cache := NewJWKSCacheWithClient("https://example.com/jwks", 10*time.Millisecond, mockClient)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		cache.StartRefreshLoop(ctx)
		close(done)
	}()

	// Let it attempt a few refreshes
	time.Sleep(50 * time.Millisecond)
	cancel()

	<-done

	// Should have attempted multiple refreshes despite errors
	assert.Greater(t, callCount, 0)
}
