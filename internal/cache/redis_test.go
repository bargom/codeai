package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisCache_WithURL(t *testing.T) {
	cache, err := NewRedisCache(Config{
		Type:       "redis",
		URL:        "redis://localhost:6379",
		DefaultTTL: time.Minute,
		PoolSize:   5,
	})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Close()
}

func TestNewRedisCache_WithDefaults(t *testing.T) {
	cache, err := NewRedisCache(Config{
		Type:       "redis",
		Password:   "",
		DB:         0,
		DefaultTTL: time.Minute,
		PoolSize:   5,
	})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Close()
}

func TestNewRedisCache_InvalidURL(t *testing.T) {
	_, err := NewRedisCache(Config{
		Type: "redis",
		URL:  "invalid://url",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid redis URL")
}

func TestNewRedisCache_ClusterMode(t *testing.T) {
	cache, err := NewRedisCache(Config{
		Type:         "redis",
		ClusterMode:  true,
		ClusterAddrs: []string{"localhost:7000", "localhost:7001"},
		DefaultTTL:   time.Minute,
		PoolSize:     5,
	})
	require.NoError(t, err)
	require.NotNil(t, cache)
	assert.True(t, cache.isCluster)
	defer cache.Close()
}

func TestRedisCache_PrefixKey(t *testing.T) {
	cache, _ := NewRedisCache(Config{
		Type:   "redis",
		Prefix: "myapp",
	})
	defer cache.Close()

	assert.Equal(t, "myapp:key", cache.prefixKey("key"))
}

func TestRedisCache_PrefixKey_NoPrefix(t *testing.T) {
	cache, _ := NewRedisCache(Config{
		Type:   "redis",
		Prefix: "",
	})
	defer cache.Close()

	assert.Equal(t, "key", cache.prefixKey("key"))
}

func TestRedisCache_StripPrefix(t *testing.T) {
	cache, _ := NewRedisCache(Config{
		Type:   "redis",
		Prefix: "myapp",
	})
	defer cache.Close()

	assert.Equal(t, "key", cache.stripPrefix("myapp:key"))
	assert.Equal(t, "k", cache.stripPrefix("k"))
}

func TestRedisCache_StripPrefix_NoPrefix(t *testing.T) {
	cache, _ := NewRedisCache(Config{
		Type:   "redis",
		Prefix: "",
	})
	defer cache.Close()

	assert.Equal(t, "key", cache.stripPrefix("key"))
}

func TestNew_Redis(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Type = "redis"
	cfg.URL = "redis://localhost:6379"

	cache, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Close()

	_, ok := cache.(*RedisCache)
	assert.True(t, ok)
}
