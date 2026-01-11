package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "memory", cfg.Type)
	assert.Equal(t, 5*time.Minute, cfg.DefaultTTL)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Equal(t, int64(100*1024*1024), cfg.MaxMemory)
	assert.Equal(t, 10000, cfg.MaxItems)
}

func TestNew(t *testing.T) {
	t.Run("memory cache", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Type = "memory"

		cache, err := New(cfg)
		require.NoError(t, err)
		require.NotNil(t, cache)
		defer cache.Close()

		_, ok := cache.(*MemoryCache)
		assert.True(t, ok)
	})

	t.Run("empty type defaults to memory", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Type = ""

		cache, err := New(cfg)
		require.NoError(t, err)
		require.NotNil(t, cache)
		defer cache.Close()

		_, ok := cache.(*MemoryCache)
		assert.True(t, ok)
	})

	t.Run("unsupported type", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Type = "invalid"

		cache, err := New(cfg)
		assert.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "unsupported cache type")
	})
}

// MemoryCache tests
func TestMemoryCache_BasicOperations(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	t.Run("set and get", func(t *testing.T) {
		err := cache.Set(ctx, "key1", []byte("value1"), 0)
		require.NoError(t, err)

		data, err := cache.Get(ctx, "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), data)
	})

	t.Run("get miss", func(t *testing.T) {
		data, err := cache.Get(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrCacheMiss)
		assert.Nil(t, data)
	})

	t.Run("delete", func(t *testing.T) {
		err := cache.Set(ctx, "key2", []byte("value2"), 0)
		require.NoError(t, err)

		err = cache.Delete(ctx, "key2")
		require.NoError(t, err)

		_, err = cache.Get(ctx, "key2")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("delete nonexistent key", func(t *testing.T) {
		err := cache.Delete(ctx, "nonexistent")
		require.NoError(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		err := cache.Set(ctx, "key3", []byte("value3"), 0)
		require.NoError(t, err)

		exists, err := cache.Exists(ctx, "key3")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = cache.Exists(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryCache_TTL(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: 50 * time.Millisecond})
	defer cache.Close()
	ctx := context.Background()

	t.Run("item expires after TTL", func(t *testing.T) {
		err := cache.Set(ctx, "expiring", []byte("value"), 50*time.Millisecond)
		require.NoError(t, err)

		// Should exist immediately
		data, err := cache.Get(ctx, "expiring")
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), data)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Should be expired
		_, err = cache.Get(ctx, "expiring")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("exists returns false for expired items", func(t *testing.T) {
		err := cache.Set(ctx, "expiring2", []byte("value"), 50*time.Millisecond)
		require.NoError(t, err)

		exists, err := cache.Exists(ctx, "expiring2")
		require.NoError(t, err)
		assert.True(t, exists)

		time.Sleep(100 * time.Millisecond)

		exists, err = cache.Exists(ctx, "expiring2")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryCache_JSON(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("set and get JSON", func(t *testing.T) {
		input := TestStruct{Name: "test", Value: 42}
		err := cache.SetJSON(ctx, "json1", input, 0)
		require.NoError(t, err)

		var output TestStruct
		err = cache.GetJSON(ctx, "json1", &output)
		require.NoError(t, err)
		assert.Equal(t, input, output)
	})

	t.Run("get JSON miss", func(t *testing.T) {
		var output TestStruct
		err := cache.GetJSON(ctx, "nonexistent", &output)
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("set JSON marshal error", func(t *testing.T) {
		// Channels cannot be marshaled to JSON
		err := cache.SetJSON(ctx, "bad", make(chan int), 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json marshal")
	})

	t.Run("get JSON unmarshal error", func(t *testing.T) {
		// Store invalid JSON
		err := cache.Set(ctx, "badjson", []byte("not json"), 0)
		require.NoError(t, err)

		var output TestStruct
		err = cache.GetJSON(ctx, "badjson", &output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json unmarshal")
	})
}

func TestMemoryCache_GetOrSet(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	t.Run("sets value on miss", func(t *testing.T) {
		callCount := 0
		fn := func() (any, error) {
			callCount++
			return map[string]string{"computed": "value"}, nil
		}

		result, err := cache.GetOrSet(ctx, "getorset1", time.Minute, fn)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)

		// Verify result contains expected data
		resultMap, ok := result.(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "value", resultMap["computed"])

		// Access again - should use cached value
		result2, err := cache.GetOrSet(ctx, "getorset1", time.Minute, fn)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // fn should not be called again

		// JSON unmarshaling creates map[string]interface{}, so check the value
		result2Map, ok := result2.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", result2Map["computed"])
	})

	t.Run("returns error from fn", func(t *testing.T) {
		fn := func() (any, error) {
			return nil, assert.AnError
		}

		_, err := cache.GetOrSet(ctx, "getorset2", time.Minute, fn)
		assert.Error(t, err)
	})
}

func TestMemoryCache_Bulk(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	t.Run("MGet", func(t *testing.T) {
		_ = cache.Set(ctx, "bulk1", []byte("v1"), 0)
		_ = cache.Set(ctx, "bulk2", []byte("v2"), 0)

		results, err := cache.MGet(ctx, "bulk1", "bulk2", "bulk3")
		require.NoError(t, err)
		require.Len(t, results, 3)
		assert.Equal(t, []byte("v1"), results[0])
		assert.Equal(t, []byte("v2"), results[1])
		assert.Nil(t, results[2])
	})

	t.Run("MSet", func(t *testing.T) {
		items := map[string][]byte{
			"mset1": []byte("v1"),
			"mset2": []byte("v2"),
		}

		err := cache.MSet(ctx, items, 0)
		require.NoError(t, err)

		data, err := cache.Get(ctx, "mset1")
		require.NoError(t, err)
		assert.Equal(t, []byte("v1"), data)

		data, err = cache.Get(ctx, "mset2")
		require.NoError(t, err)
		assert.Equal(t, []byte("v2"), data)
	})
}

func TestMemoryCache_Keys(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "user:1", []byte("v1"), 0)
	_ = cache.Set(ctx, "user:2", []byte("v2"), 0)
	_ = cache.Set(ctx, "order:1", []byte("v3"), 0)

	t.Run("keys with pattern", func(t *testing.T) {
		keys, err := cache.Keys(ctx, "user:*")
		require.NoError(t, err)
		assert.Len(t, keys, 2)
	})

	t.Run("keys all", func(t *testing.T) {
		keys, err := cache.Keys(ctx, "*")
		require.NoError(t, err)
		assert.Len(t, keys, 3)
	})

	t.Run("keys invalid pattern", func(t *testing.T) {
		_, err := cache.Keys(ctx, "[")
		assert.Error(t, err)
	})
}

func TestMemoryCache_DeletePattern(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "del:1", []byte("v1"), 0)
	_ = cache.Set(ctx, "del:2", []byte("v2"), 0)
	_ = cache.Set(ctx, "keep:1", []byte("v3"), 0)

	err := cache.DeletePattern(ctx, "del:*")
	require.NoError(t, err)

	_, err = cache.Get(ctx, "del:1")
	assert.ErrorIs(t, err, ErrCacheMiss)

	_, err = cache.Get(ctx, "del:2")
	assert.ErrorIs(t, err, ErrCacheMiss)

	data, err := cache.Get(ctx, "keep:1")
	require.NoError(t, err)
	assert.Equal(t, []byte("v3"), data)
}

func TestMemoryCache_IncrDecr(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	t.Run("incr new key", func(t *testing.T) {
		val, err := cache.Incr(ctx, "counter1")
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)
	})

	t.Run("incr existing key", func(t *testing.T) {
		val, err := cache.Incr(ctx, "counter1")
		require.NoError(t, err)
		assert.Equal(t, int64(2), val)

		val, err = cache.Incr(ctx, "counter1")
		require.NoError(t, err)
		assert.Equal(t, int64(3), val)
	})

	t.Run("decr", func(t *testing.T) {
		val, err := cache.Decr(ctx, "counter1")
		require.NoError(t, err)
		assert.Equal(t, int64(2), val)
	})

	t.Run("decr new key", func(t *testing.T) {
		val, err := cache.Decr(ctx, "counter2")
		require.NoError(t, err)
		assert.Equal(t, int64(-1), val)
	})

	t.Run("incr non-numeric value", func(t *testing.T) {
		_ = cache.Set(ctx, "notnum", []byte("hello"), 0)
		_, err := cache.Incr(ctx, "notnum")
		assert.Error(t, err)
	})

	t.Run("decr non-numeric value", func(t *testing.T) {
		_ = cache.Set(ctx, "notnum2", []byte("hello"), 0)
		_, err := cache.Decr(ctx, "notnum2")
		assert.Error(t, err)
	})
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	cache := NewMemoryCache(Config{
		DefaultTTL: time.Minute,
		MaxMemory:  100,
		MaxItems:   0, // No item limit
	})
	defer cache.Close()
	ctx := context.Background()

	// Fill cache to capacity
	_ = cache.Set(ctx, "lru1", make([]byte, 40), 0)
	_ = cache.Set(ctx, "lru2", make([]byte, 40), 0)

	// Access lru1 to make it most recently used
	_, _ = cache.Get(ctx, "lru1")

	// Add new item that requires eviction
	_ = cache.Set(ctx, "lru3", make([]byte, 40), 0)

	// lru2 should be evicted (least recently used)
	_, err := cache.Get(ctx, "lru2")
	assert.ErrorIs(t, err, ErrCacheMiss)

	// lru1 should still exist
	_, err = cache.Get(ctx, "lru1")
	require.NoError(t, err)
}

func TestMemoryCache_MaxItems(t *testing.T) {
	cache := NewMemoryCache(Config{
		DefaultTTL: time.Minute,
		MaxMemory:  0, // No memory limit
		MaxItems:   2,
	})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "item1", []byte("v1"), 0)
	_ = cache.Set(ctx, "item2", []byte("v2"), 0)
	_ = cache.Set(ctx, "item3", []byte("v3"), 0)

	// Should have evicted oldest item
	stats := cache.Stats()
	assert.LessOrEqual(t, stats.Keys, int64(2))
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "stat1", []byte("value"), 0)
	_, _ = cache.Get(ctx, "stat1")           // hit
	_, _ = cache.Get(ctx, "stat1")           // hit
	_, _ = cache.Get(ctx, "nonexistent")     // miss

	stats := cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Keys)
	assert.Greater(t, stats.MemoryUsed, int64(0))
}

func TestMemoryCache_Health(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	err := cache.Health(context.Background())
	assert.NoError(t, err)
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "clear1", []byte("v1"), 0)
	_ = cache.Set(ctx, "clear2", []byte("v2"), 0)

	cache.Clear()

	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Keys)
	assert.Equal(t, int64(0), stats.MemoryUsed)
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	ctx := context.Background()

	_ = cache.Set(ctx, "close1", []byte("v1"), 0)

	err := cache.Close()
	require.NoError(t, err)

	// Multiple closes should be safe
	err = cache.Close()
	require.NoError(t, err)
}

func TestMemoryCache_OverwriteKey(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "overwrite", []byte("original"), 0)
	_ = cache.Set(ctx, "overwrite", []byte("updated"), 0)

	data, err := cache.Get(ctx, "overwrite")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), data)
}

// Middleware tests
func TestMiddleware(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	middleware := NewMiddleware(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello"}`))
	})

	wrapped := middleware.Handler(time.Minute)(handler)

	t.Run("caches GET requests", func(t *testing.T) {
		// First request - MISS
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "MISS", rec.Header().Get("X-Cache"))
		assert.Equal(t, `{"message":"hello"}`, rec.Body.String())

		// Second request - HIT
		req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusOK, rec2.Code)
		assert.Equal(t, "HIT", rec2.Header().Get("X-Cache"))
		assert.Equal(t, `{"message":"hello"}`, rec2.Body.String())
	})

	t.Run("does not cache POST requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("X-Cache"))
	})
}

func TestMiddleware_HandlerWithKeyFunc(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	middleware := NewMiddleware(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	})

	keyFunc := func(r *http.Request) string {
		return "custom:" + r.URL.Path
	}

	wrapped := middleware.HandlerWithKeyFunc(time.Minute, keyFunc)(handler)

	t.Run("uses custom key function", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/custom/path", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, "MISS", rec.Header().Get("X-Cache"))

		// Verify it's cached with custom key
		req2 := httptest.NewRequest(http.MethodGet, "/custom/path", nil)
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		assert.Equal(t, "HIT", rec2.Header().Get("X-Cache"))
	})

	t.Run("skips caching if key is empty", func(t *testing.T) {
		emptyKeyFunc := func(r *http.Request) string {
			return ""
		}

		wrapped2 := middleware.HandlerWithKeyFunc(time.Minute, emptyKeyFunc)(handler)

		req := httptest.NewRequest(http.MethodGet, "/nocache", nil)
		rec := httptest.NewRecorder()
		wrapped2.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("X-Cache"))
	})
}

func TestMiddleware_Invalidate(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	middleware := NewMiddleware(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	})

	wrapped := middleware.Handler(time.Minute)(handler)

	// Cache a response
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Invalidate it
	err := middleware.Invalidate(req)
	require.NoError(t, err)

	// Next request should be a MISS
	req2 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	assert.Equal(t, "MISS", rec2.Header().Get("X-Cache"))
}

func TestMiddleware_NoCache_ErrorResponses(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	middleware := NewMiddleware(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	wrapped := middleware.Handler(time.Minute)(handler)

	// First request
	req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Second request should still be a miss (errors not cached)
	req2 := httptest.NewRequest(http.MethodGet, "/api/error", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	assert.Equal(t, "MISS", rec2.Header().Get("X-Cache"))
}

func TestMiddleware_InvalidatePattern(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	middleware := NewMiddleware(cache)

	// Set some cached values directly
	_ = cache.Set(ctx, "http:abc123", []byte("cached1"), 0)
	_ = cache.Set(ctx, "http:def456", []byte("cached2"), 0)

	// Invalidate pattern
	err := middleware.InvalidatePattern("*")
	require.NoError(t, err)
}

func TestMiddleware_POST_Method(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	middleware := NewMiddleware(cache)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	})

	wrapped := middleware.HandlerWithKeyFunc(time.Minute, func(r *http.Request) string {
		return "key"
	})(handler)

	// POST should not be cached
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("X-Cache"))
}

func TestMemoryCache_MSetError(t *testing.T) {
	cache := NewMemoryCache(Config{
		DefaultTTL: time.Minute,
		MaxMemory:  10, // Very small
		MaxItems:   1,  // Only 1 item allowed
	})
	defer cache.Close()
	ctx := context.Background()

	// This should work due to eviction
	items := map[string][]byte{
		"key1": []byte("v1"),
		"key2": []byte("v2"),
	}

	err := cache.MSet(ctx, items, 0)
	require.NoError(t, err)
}

func TestMemoryCache_DeletePattern_Error(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "test", []byte("v"), 0)

	// Invalid pattern
	err := cache.DeletePattern(ctx, "[")
	assert.Error(t, err)
}

func TestMemoryCache_Keys_Expired(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: 10 * time.Millisecond})
	defer cache.Close()
	ctx := context.Background()

	_ = cache.Set(ctx, "expiring", []byte("v"), 10*time.Millisecond)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Keys should not return expired items
	keys, err := cache.Keys(ctx, "*")
	require.NoError(t, err)
	assert.NotContains(t, keys, "expiring")
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: 10 * time.Millisecond})
	defer cache.Close()
	ctx := context.Background()

	// Set an item with short TTL
	_ = cache.Set(ctx, "cleanup1", []byte("v1"), 10*time.Millisecond)
	_ = cache.Set(ctx, "cleanup2", []byte("v2"), time.Hour) // Long TTL

	// Wait for first item to expire
	time.Sleep(20 * time.Millisecond)

	// Trigger cleanup
	cache.Cleanup()

	// First item should be gone
	_, err := cache.Get(ctx, "cleanup1")
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Second item should still exist
	_, err = cache.Get(ctx, "cleanup2")
	require.NoError(t, err)
}

func TestMemoryCache_Cleanup_Empty(t *testing.T) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()

	// Cleanup on empty cache should not panic
	cache.Cleanup()
}

func TestMemoryCache_DefaultTTL_Zero(t *testing.T) {
	// When DefaultTTL is not set, it should default to 5 minutes
	cache := NewMemoryCache(Config{})
	defer cache.Close()

	assert.Equal(t, 5*time.Minute, cache.config.DefaultTTL)
}
