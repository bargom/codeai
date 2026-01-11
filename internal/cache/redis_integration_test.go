//go:build integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisContainer(t *testing.T) (*RedisCache, func()) {
	ctx := context.Background()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	cache, err := NewRedisCache(Config{
		Type:       "redis",
		URL:        connStr,
		DefaultTTL: time.Minute,
		Prefix:     "test",
	})
	require.NoError(t, err)

	cleanup := func() {
		cache.Close()
		redisContainer.Terminate(ctx)
	}

	return cache, cleanup
}

func TestRedisCache_Integration_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
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

func TestRedisCache_Integration_TTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("item expires after TTL", func(t *testing.T) {
		err := cache.Set(ctx, "expiring", []byte("value"), 100*time.Millisecond)
		require.NoError(t, err)

		// Should exist immediately
		data, err := cache.Get(ctx, "expiring")
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), data)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Should be expired
		_, err = cache.Get(ctx, "expiring")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})
}

func TestRedisCache_Integration_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
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
}

func TestRedisCache_Integration_GetOrSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
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

		resultMap, ok := result.(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "value", resultMap["computed"])

		// Access again - should use cached value
		result2, err := cache.GetOrSet(ctx, "getorset1", time.Minute, fn)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)

		result2Map, ok := result2.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", result2Map["computed"])
	})
}

func TestRedisCache_Integration_Bulk(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
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

	t.Run("MGet empty", func(t *testing.T) {
		results, err := cache.MGet(ctx)
		require.NoError(t, err)
		assert.Nil(t, results)
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
	})

	t.Run("MSet empty", func(t *testing.T) {
		err := cache.MSet(ctx, nil, 0)
		require.NoError(t, err)
	})
}

func TestRedisCache_Integration_Keys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	_ = cache.Set(ctx, "user:1", []byte("v1"), 0)
	_ = cache.Set(ctx, "user:2", []byte("v2"), 0)
	_ = cache.Set(ctx, "order:1", []byte("v3"), 0)

	t.Run("keys with pattern", func(t *testing.T) {
		keys, err := cache.Keys(ctx, "user:*")
		require.NoError(t, err)
		assert.Len(t, keys, 2)
	})
}

func TestRedisCache_Integration_DeletePattern(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	_ = cache.Set(ctx, "del:1", []byte("v1"), 0)
	_ = cache.Set(ctx, "del:2", []byte("v2"), 0)
	_ = cache.Set(ctx, "keep:1", []byte("v3"), 0)

	err := cache.DeletePattern(ctx, "*del:*")
	require.NoError(t, err)

	_, err = cache.Get(ctx, "del:1")
	assert.ErrorIs(t, err, ErrCacheMiss)

	_, err = cache.Get(ctx, "del:2")
	assert.ErrorIs(t, err, ErrCacheMiss)

	data, err := cache.Get(ctx, "keep:1")
	require.NoError(t, err)
	assert.Equal(t, []byte("v3"), data)
}

func TestRedisCache_Integration_IncrDecr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
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
	})

	t.Run("decr", func(t *testing.T) {
		val, err := cache.Decr(ctx, "counter1")
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)
	})

	t.Run("decr new key", func(t *testing.T) {
		val, err := cache.Decr(ctx, "counter2")
		require.NoError(t, err)
		assert.Equal(t, int64(-1), val)
	})
}

func TestRedisCache_Integration_Health(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	err := cache.Health(ctx)
	assert.NoError(t, err)
}

func TestRedisCache_Integration_Stats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	_ = cache.Set(ctx, "stat1", []byte("value"), 0)
	_, _ = cache.Get(ctx, "stat1")
	_, _ = cache.Get(ctx, "stat1")
	_, _ = cache.Get(ctx, "nonexistent")

	stats := cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestRedisCache_Integration_Prefix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cache, cleanup := setupRedisContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Set with prefix
	err := cache.Set(ctx, "mykey", []byte("myvalue"), 0)
	require.NoError(t, err)

	// Get should work
	data, err := cache.Get(ctx, "mykey")
	require.NoError(t, err)
	assert.Equal(t, []byte("myvalue"), data)

	// Keys should return unprefixed key
	keys, err := cache.Keys(ctx, "*")
	require.NoError(t, err)
	found := false
	for _, k := range keys {
		if k == "mykey" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find unprefixed key")
}
