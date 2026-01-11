package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Cache using Redis as the backend.
type RedisCache struct {
	client     redis.UniversalClient
	config     Config
	hits       int64
	misses     int64
	isCluster  bool
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(cfg Config) (*RedisCache, error) {
	var client redis.UniversalClient

	if cfg.ClusterMode && len(cfg.ClusterAddrs) > 0 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddrs,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			MaxRetries:   cfg.MaxRetries,
		})
	} else if cfg.URL != "" {
		opts, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid redis URL: %w", err)
		}
		if cfg.PoolSize > 0 {
			opts.PoolSize = cfg.PoolSize
		}
		if cfg.MinIdleConns > 0 {
			opts.MinIdleConns = cfg.MinIdleConns
		}
		if cfg.MaxRetries > 0 {
			opts.MaxRetries = cfg.MaxRetries
		}
		client = redis.NewClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         "localhost:6379",
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			MaxRetries:   cfg.MaxRetries,
		})
	}

	return &RedisCache{
		client:    client,
		config:    cfg,
		isCluster: cfg.ClusterMode,
	}, nil
}

func (c *RedisCache) prefixKey(key string) string {
	if c.config.Prefix != "" {
		return c.config.Prefix + ":" + key
	}
	return key
}

func (c *RedisCache) stripPrefix(key string) string {
	if c.config.Prefix != "" && len(key) > len(c.config.Prefix)+1 {
		return key[len(c.config.Prefix)+1:]
	}
	return key
}

// Get retrieves a value from the cache.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, c.prefixKey(key)).Bytes()
	if err == redis.Nil {
		atomic.AddInt64(&c.misses, 1)
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	atomic.AddInt64(&c.hits, 1)
	return data, nil
}

// Set stores a value in the cache with the given TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}
	err := c.client.Set(ctx, c.prefixKey(key), value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// Delete removes a key from the cache.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, c.prefixKey(key)).Err()
	if err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

// Exists checks if a key exists in the cache.
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.prefixKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// GetJSON retrieves and unmarshals a JSON value from the cache.
func (c *RedisCache) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// SetJSON marshals and stores a value as JSON in the cache.
func (c *RedisCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

// GetOrSet implements the cache-aside pattern.
func (c *RedisCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	var cached any
	err := c.GetJSON(ctx, key, &cached)
	if err == nil {
		return cached, nil
	}
	if err != ErrCacheMiss {
		// Log error but continue to fetch fresh data
	}

	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors)
	_ = c.SetJSON(ctx, key, value, ttl)

	return value, nil
}

// MGet retrieves multiple values from the cache.
func (c *RedisCache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	prefixedKeys := make([]string, len(keys))
	for i, k := range keys {
		prefixedKeys[i] = c.prefixKey(k)
	}

	results, err := c.client.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget: %w", err)
	}

	data := make([][]byte, len(results))
	for i, r := range results {
		if r != nil {
			if s, ok := r.(string); ok {
				data[i] = []byte(s)
				atomic.AddInt64(&c.hits, 1)
			}
		} else {
			atomic.AddInt64(&c.misses, 1)
		}
	}
	return data, nil
}

// MSet stores multiple values in the cache.
func (c *RedisCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	pipe := c.client.Pipeline()
	for key, value := range items {
		pipe.Set(ctx, c.prefixKey(key), value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis mset: %w", err)
	}
	return nil
}

// DeletePattern deletes all keys matching the pattern.
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	prefixedPattern := c.prefixKey(pattern)

	var cursor uint64
	var allKeys []string

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, prefixedPattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan: %w", err)
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(allKeys) > 0 {
		if err := c.client.Del(ctx, allKeys...).Err(); err != nil {
			return fmt.Errorf("redis delete pattern: %w", err)
		}
	}

	return nil
}

// Keys returns all keys matching the pattern.
func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := c.client.Keys(ctx, c.prefixKey(pattern)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis keys: %w", err)
	}

	// Strip prefix from keys
	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = c.stripPrefix(k)
	}
	return result, nil
}

// Incr increments a key's value.
func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Incr(ctx, c.prefixKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr: %w", err)
	}
	return val, nil
}

// Decr decrements a key's value.
func (c *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Decr(ctx, c.prefixKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("redis decr: %w", err)
	}
	return val, nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("redis close: %w", err)
	}
	return nil
}

// Health checks if Redis is reachable.
func (c *RedisCache) Health(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check: %w", err)
	}
	return nil
}

// Stats returns cache statistics.
func (c *RedisCache) Stats() Stats {
	ctx := context.Background()

	// Get key count
	var keyCount int64
	if info, err := c.client.Info(ctx, "keyspace").Result(); err == nil {
		// Parse keyspace info for key count
		_ = info // In production, parse this properly
	}

	// Try DBSize for single node
	if size, err := c.client.DBSize(ctx).Result(); err == nil {
		keyCount = size
	}

	return Stats{
		Hits:   atomic.LoadInt64(&c.hits),
		Misses: atomic.LoadInt64(&c.misses),
		Keys:   keyCount,
	}
}
