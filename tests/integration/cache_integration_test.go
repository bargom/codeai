// Package integration provides integration tests for the CodeAI modules.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheIntegration tests the complete cache functionality including
// basic operations, TTL, and middleware.
func TestCacheIntegration(t *testing.T) {
	cfg := cache.DefaultConfig()
	cfg.Type = "memory"
	cfg.DefaultTTL = time.Minute
	cfg.MaxMemory = 10 * 1024 * 1024 // 10MB
	cfg.MaxItems = 1000

	c, err := cache.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	t.Run("basic CRUD operations", func(t *testing.T) {
		// Set
		err := c.Set(ctx, "test:key1", []byte("value1"), 0)
		require.NoError(t, err)

		// Get
		data, err := c.Get(ctx, "test:key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), data)

		// Exists
		exists, err := c.Exists(ctx, "test:key1")
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete
		err = c.Delete(ctx, "test:key1")
		require.NoError(t, err)

		// Verify deleted
		exists, err = c.Exists(ctx, "test:key1")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("JSON operations", func(t *testing.T) {
		type TestData struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		input := TestData{Name: "test", Count: 42}

		// SetJSON
		err := c.SetJSON(ctx, "test:json", input, 0)
		require.NoError(t, err)

		// GetJSON
		var output TestData
		err = c.GetJSON(ctx, "test:json", &output)
		require.NoError(t, err)
		assert.Equal(t, input, output)
	})

	t.Run("GetOrSet pattern", func(t *testing.T) {
		callCount := 0
		loader := func() (any, error) {
			callCount++
			return map[string]string{"loaded": "data"}, nil
		}

		// First call - should invoke loader
		result1, err := c.GetOrSet(ctx, "test:getorset", time.Minute, loader)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.NotNil(t, result1)

		// Second call - should use cache
		result2, err := c.GetOrSet(ctx, "test:getorset", time.Minute, loader)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // Still 1, loader not called
		assert.NotNil(t, result2)
	})

	t.Run("bulk operations", func(t *testing.T) {
		// MSet
		items := map[string][]byte{
			"bulk:1": []byte("v1"),
			"bulk:2": []byte("v2"),
			"bulk:3": []byte("v3"),
		}
		err := c.MSet(ctx, items, 0)
		require.NoError(t, err)

		// MGet
		results, err := c.MGet(ctx, "bulk:1", "bulk:2", "bulk:3", "bulk:missing")
		require.NoError(t, err)
		assert.Len(t, results, 4)
		assert.Equal(t, []byte("v1"), results[0])
		assert.Equal(t, []byte("v2"), results[1])
		assert.Equal(t, []byte("v3"), results[2])
		assert.Nil(t, results[3]) // Missing key
	})

	t.Run("keys and pattern matching", func(t *testing.T) {
		_ = c.Set(ctx, "pattern:a", []byte("1"), 0)
		_ = c.Set(ctx, "pattern:b", []byte("2"), 0)
		_ = c.Set(ctx, "other:c", []byte("3"), 0)

		keys, err := c.Keys(ctx, "pattern:*")
		require.NoError(t, err)
		assert.Len(t, keys, 2)
	})

	t.Run("delete by pattern", func(t *testing.T) {
		_ = c.Set(ctx, "delete:1", []byte("1"), 0)
		_ = c.Set(ctx, "delete:2", []byte("2"), 0)
		_ = c.Set(ctx, "keep:1", []byte("3"), 0)

		err := c.DeletePattern(ctx, "delete:*")
		require.NoError(t, err)

		// Verify deleted
		exists1, _ := c.Exists(ctx, "delete:1")
		exists2, _ := c.Exists(ctx, "delete:2")
		existsKeep, _ := c.Exists(ctx, "keep:1")

		assert.False(t, exists1)
		assert.False(t, exists2)
		assert.True(t, existsKeep)
	})

	t.Run("increment and decrement", func(t *testing.T) {
		// Incr on new key
		val, err := c.Incr(ctx, "counter:new")
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		// Incr again
		val, err = c.Incr(ctx, "counter:new")
		require.NoError(t, err)
		assert.Equal(t, int64(2), val)

		// Decr
		val, err = c.Decr(ctx, "counter:new")
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)
	})
}

// TestCacheTTLExpiration tests TTL-based expiration.
func TestCacheTTLExpiration(t *testing.T) {
	cfg := cache.DefaultConfig()
	cfg.Type = "memory"
	cfg.DefaultTTL = 50 * time.Millisecond

	c, err := cache.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Set item with short TTL
	err = c.Set(ctx, "expiring", []byte("value"), 50*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	exists, err := c.Exists(ctx, "expiring")
	require.NoError(t, err)
	assert.True(t, exists)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, err = c.Get(ctx, "expiring")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

// TestCacheMiddlewareIntegration tests the HTTP caching middleware.
func TestCacheMiddlewareIntegration(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	middleware := cache.NewMiddleware(c)

	responseCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "response"}`))
	})

	wrapped := middleware.Handler(time.Minute)(handler)

	t.Run("caches GET requests", func(t *testing.T) {
		responseCount = 0

		// First request - should hit handler
		req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec1 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec1, req1)

		assert.Equal(t, http.StatusOK, rec1.Code)
		assert.Equal(t, "MISS", rec1.Header().Get("X-Cache"))
		assert.Equal(t, 1, responseCount)

		// Second request - should hit cache
		req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusOK, rec2.Code)
		assert.Equal(t, "HIT", rec2.Header().Get("X-Cache"))
		assert.Equal(t, 1, responseCount) // Handler not called again
	})

	t.Run("does not cache POST requests", func(t *testing.T) {
		responseCount = 0

		req1 := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec1 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec1, req1)

		assert.Equal(t, http.StatusOK, rec1.Code)
		assert.Empty(t, rec1.Header().Get("X-Cache"))
		assert.Equal(t, 1, responseCount)

		// Second POST - should also hit handler
		req2 := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, 2, responseCount)
	})

	t.Run("does not cache error responses", func(t *testing.T) {
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		})

		errorWrapped := middleware.Handler(time.Minute)(errorHandler)

		req1 := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rec1 := httptest.NewRecorder()
		errorWrapped.ServeHTTP(rec1, req1)

		assert.Equal(t, http.StatusInternalServerError, rec1.Code)

		// Second request should also be a MISS (errors not cached)
		req2 := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rec2 := httptest.NewRecorder()
		errorWrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, "MISS", rec2.Header().Get("X-Cache"))
	})

	t.Run("invalidate cached request", func(t *testing.T) {
		responseCount = 0

		// Cache the response
		req1 := httptest.NewRequest(http.MethodGet, "/api/invalidate", nil)
		rec1 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec1, req1)

		assert.Equal(t, "MISS", rec1.Header().Get("X-Cache"))
		assert.Equal(t, 1, responseCount)

		// Invalidate
		err := middleware.Invalidate(req1)
		require.NoError(t, err)

		// Next request should be a MISS
		req2 := httptest.NewRequest(http.MethodGet, "/api/invalidate", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, "MISS", rec2.Header().Get("X-Cache"))
		assert.Equal(t, 2, responseCount)
	})
}

// TestCacheMiddlewareWithCustomKeyFunc tests custom key function for caching.
func TestCacheMiddlewareWithCustomKeyFunc(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	middleware := cache.NewMiddleware(c)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	})

	// Custom key function that includes query params
	keyFunc := func(r *http.Request) string {
		return "custom:" + r.URL.Path + "?" + r.URL.RawQuery
	}

	wrapped := middleware.HandlerWithKeyFunc(time.Minute, keyFunc)(handler)

	t.Run("different query params get different cache entries", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/api/data?id=1", nil)
		rec1 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec1, req1)
		assert.Equal(t, "MISS", rec1.Header().Get("X-Cache"))

		req2 := httptest.NewRequest(http.MethodGet, "/api/data?id=2", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)
		assert.Equal(t, "MISS", rec2.Header().Get("X-Cache"))

		// Same query should be HIT
		req3 := httptest.NewRequest(http.MethodGet, "/api/data?id=1", nil)
		rec3 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec3, req3)
		assert.Equal(t, "HIT", rec3.Header().Get("X-Cache"))
	})
}

// TestCacheLRUEviction tests LRU eviction when cache is full.
func TestCacheLRUEviction(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{
		DefaultTTL: time.Minute,
		MaxMemory:  100, // Very small - 100 bytes
		MaxItems:   0,
	})
	defer c.Close()

	ctx := context.Background()

	// Fill cache
	_ = c.Set(ctx, "lru:1", make([]byte, 40), 0)
	_ = c.Set(ctx, "lru:2", make([]byte, 40), 0)

	// Access lru:1 to make it recently used
	_, _ = c.Get(ctx, "lru:1")

	// Add new item - should evict lru:2 (least recently used)
	_ = c.Set(ctx, "lru:3", make([]byte, 40), 0)

	// lru:2 should be evicted
	_, err := c.Get(ctx, "lru:2")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// lru:1 should still exist
	_, err = c.Get(ctx, "lru:1")
	assert.NoError(t, err)
}

// TestCacheMaxItemsEviction tests eviction when max items limit is reached.
func TestCacheMaxItemsEviction(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{
		DefaultTTL: time.Minute,
		MaxMemory:  0, // No memory limit
		MaxItems:   2,
	})
	defer c.Close()

	ctx := context.Background()

	_ = c.Set(ctx, "item:1", []byte("v1"), 0)
	_ = c.Set(ctx, "item:2", []byte("v2"), 0)
	_ = c.Set(ctx, "item:3", []byte("v3"), 0)

	// Should have evicted oldest item
	stats := c.Stats()
	assert.LessOrEqual(t, stats.Keys, int64(2))
}

// TestCacheStats tests cache statistics.
func TestCacheStats(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	ctx := context.Background()

	_ = c.Set(ctx, "stat:1", []byte("value"), 0)

	// Generate hits and misses
	_, _ = c.Get(ctx, "stat:1")     // hit
	_, _ = c.Get(ctx, "stat:1")     // hit
	_, _ = c.Get(ctx, "stat:miss")  // miss

	stats := c.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Keys)
	assert.Greater(t, stats.MemoryUsed, int64(0))
}

// TestCacheHealth tests cache health check.
func TestCacheHealth(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	err := c.Health(context.Background())
	assert.NoError(t, err)
}

// TestCacheCleanup tests manual cache cleanup.
func TestCacheCleanup(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: 10 * time.Millisecond})
	defer c.Close()

	ctx := context.Background()

	// Set item with short TTL
	_ = c.Set(ctx, "cleanup:1", []byte("v1"), 10*time.Millisecond)
	_ = c.Set(ctx, "cleanup:2", []byte("v2"), time.Hour) // Long TTL

	// Wait for first item to expire
	time.Sleep(20 * time.Millisecond)

	// Trigger cleanup
	c.Cleanup()

	// First item should be gone
	_, err := c.Get(ctx, "cleanup:1")
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// Second item should still exist
	_, err = c.Get(ctx, "cleanup:2")
	assert.NoError(t, err)
}

// TestCacheClear tests clearing all cache entries.
func TestCacheClear(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	ctx := context.Background()

	_ = c.Set(ctx, "clear:1", []byte("v1"), 0)
	_ = c.Set(ctx, "clear:2", []byte("v2"), 0)

	c.Clear()

	stats := c.Stats()
	assert.Equal(t, int64(0), stats.Keys)
	assert.Equal(t, int64(0), stats.MemoryUsed)
}

// TestCacheConcurrentAccess tests thread-safety of cache operations.
func TestCacheConcurrentAccess(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	ctx := context.Background()
	done := make(chan bool, 100)

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func(i int) {
			key := "concurrent:" + string(rune(i))
			_ = c.Set(ctx, key, []byte("value"), 0)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func(i int) {
			key := "concurrent:" + string(rune(i))
			_, _ = c.Get(ctx, key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify cache is in valid state
	err := c.Health(ctx)
	assert.NoError(t, err)
}

// TestCacheErrorCases tests error handling.
func TestCacheErrorCases(t *testing.T) {
	c := cache.NewMemoryCache(cache.Config{DefaultTTL: time.Minute})
	defer c.Close()

	ctx := context.Background()

	t.Run("get non-existent key", func(t *testing.T) {
		_, err := c.Get(ctx, "nonexistent")
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})

	t.Run("get JSON into wrong type", func(t *testing.T) {
		_ = c.Set(ctx, "badjson", []byte("not json"), 0)

		var result map[string]string
		err := c.GetJSON(ctx, "badjson", &result)
		assert.Error(t, err)
	})

	t.Run("set JSON with unmarshalable value", func(t *testing.T) {
		err := c.SetJSON(ctx, "bad", make(chan int), 0)
		assert.Error(t, err)
	})

	t.Run("incr non-numeric value", func(t *testing.T) {
		_ = c.Set(ctx, "notnum", []byte("hello"), 0)
		_, err := c.Incr(ctx, "notnum")
		assert.Error(t, err)
	})

	t.Run("invalid pattern for keys", func(t *testing.T) {
		_, err := c.Keys(ctx, "[")
		assert.Error(t, err)
	})

	t.Run("invalid pattern for delete pattern", func(t *testing.T) {
		err := c.DeletePattern(ctx, "[")
		assert.Error(t, err)
	})
}

// TestCacheNewFunction tests the New factory function.
func TestCacheNewFunction(t *testing.T) {
	t.Run("memory cache", func(t *testing.T) {
		cfg := cache.DefaultConfig()
		cfg.Type = "memory"

		c, err := cache.New(cfg)
		require.NoError(t, err)
		defer c.Close()

		_, ok := c.(*cache.MemoryCache)
		assert.True(t, ok)
	})

	t.Run("empty type defaults to memory", func(t *testing.T) {
		cfg := cache.DefaultConfig()
		cfg.Type = ""

		c, err := cache.New(cfg)
		require.NoError(t, err)
		defer c.Close()

		_, ok := c.(*cache.MemoryCache)
		assert.True(t, ok)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		cfg := cache.DefaultConfig()
		cfg.Type = "invalid"

		_, err := cache.New(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported cache type")
	})
}
