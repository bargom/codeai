# Task: Redis Cache Module

## Overview
Implement Redis-based caching with automatic cache invalidation, TTL support, and integration with CodeAI endpoint caching directives.

## Phase
Phase 2: Core Features

## Priority
Medium - Important for performance optimization.

## Dependencies
- 01-Foundation/01-project-setup.md

## Description
Create a caching module that supports Redis and in-memory caching with a unified interface, automatic key generation, and cache-aside patterns.

## Detailed Requirements

### 1. Cache Module Interface (internal/modules/cache/module.go)

```go
package cache

import (
    "context"
    "time"
)

type CacheModule interface {
    Module

    // Basic operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)

    // Typed operations
    GetJSON(ctx context.Context, key string, dest any) error
    SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error

    // Cache-aside pattern
    GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (any, error)

    // Bulk operations
    MGet(ctx context.Context, keys ...string) ([][]byte, error)
    MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error

    // Key management
    DeletePattern(ctx context.Context, pattern string) error
    Keys(ctx context.Context, pattern string) ([]string, error)

    // Atomic operations
    Incr(ctx context.Context, key string) (int64, error)
    Decr(ctx context.Context, key string) (int64, error)
}

type CacheConfig struct {
    Type       string        // "redis", "memory"
    URL        string        // Redis URL
    DefaultTTL time.Duration
    MaxMemory  int64         // For memory cache
    Prefix     string        // Key prefix
}

var ErrCacheMiss = errors.New("cache miss")
```

### 2. Redis Implementation (internal/modules/cache/redis.go)

```go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "log/slog"
)

type RedisCache struct {
    client     *redis.Client
    config     CacheConfig
    logger     *slog.Logger
}

func NewRedisCache(config CacheConfig) (*RedisCache, error) {
    opts, err := redis.ParseURL(config.URL)
    if err != nil {
        return nil, fmt.Errorf("invalid redis URL: %w", err)
    }

    client := redis.NewClient(opts)

    return &RedisCache{
        client: client,
        config: config,
        logger: slog.Default().With("module", "redis-cache"),
    }, nil
}

func (c *RedisCache) Name() string { return "redis-cache" }

func (c *RedisCache) Initialize(config *Config) error {
    return nil
}

func (c *RedisCache) Start(ctx context.Context) error {
    // Test connection
    if err := c.client.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis connection failed: %w", err)
    }
    c.logger.Info("connected to redis")
    return nil
}

func (c *RedisCache) Stop(ctx context.Context) error {
    return c.client.Close()
}

func (c *RedisCache) Health() HealthStatus {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    if err := c.client.Ping(ctx).Err(); err != nil {
        return HealthStatus{Status: "unhealthy", Error: err.Error()}
    }
    return HealthStatus{Status: "healthy"}
}

func (c *RedisCache) prefixKey(key string) string {
    if c.config.Prefix != "" {
        return c.config.Prefix + ":" + key
    }
    return key
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
    data, err := c.client.Get(ctx, c.prefixKey(key)).Bytes()
    if err == redis.Nil {
        return nil, ErrCacheMiss
    }
    if err != nil {
        return nil, err
    }
    return data, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    if ttl == 0 {
        ttl = c.config.DefaultTTL
    }
    return c.client.Set(ctx, c.prefixKey(key), value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
    return c.client.Del(ctx, c.prefixKey(key)).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
    n, err := c.client.Exists(ctx, c.prefixKey(key)).Result()
    return n > 0, err
}

func (c *RedisCache) GetJSON(ctx context.Context, key string, dest any) error {
    data, err := c.Get(ctx, key)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

func (c *RedisCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.Set(ctx, key, data, ttl)
}

func (c *RedisCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (any, error) {
    // Try to get from cache
    var cached any
    err := c.GetJSON(ctx, key, &cached)
    if err == nil {
        return cached, nil
    }
    if err != ErrCacheMiss {
        c.logger.Warn("cache get error", "key", key, "error", err)
    }

    // Call function to get value
    value, err := fn()
    if err != nil {
        return nil, err
    }

    // Store in cache (don't fail if caching fails)
    if err := c.SetJSON(ctx, key, value, ttl); err != nil {
        c.logger.Warn("cache set error", "key", key, "error", err)
    }

    return value, nil
}

func (c *RedisCache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
    prefixedKeys := make([]string, len(keys))
    for i, k := range keys {
        prefixedKeys[i] = c.prefixKey(k)
    }

    results, err := c.client.MGet(ctx, prefixedKeys...).Result()
    if err != nil {
        return nil, err
    }

    data := make([][]byte, len(results))
    for i, r := range results {
        if r != nil {
            data[i] = []byte(r.(string))
        }
    }
    return data, nil
}

func (c *RedisCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
    pipe := c.client.Pipeline()

    for key, value := range items {
        pipe.Set(ctx, c.prefixKey(key), value, ttl)
    }

    _, err := pipe.Exec(ctx)
    return err
}

func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
    iter := c.client.Scan(ctx, 0, c.prefixKey(pattern), 100).Iterator()
    var keys []string

    for iter.Next(ctx) {
        keys = append(keys, iter.Val())
    }

    if err := iter.Err(); err != nil {
        return err
    }

    if len(keys) > 0 {
        return c.client.Del(ctx, keys...).Err()
    }

    return nil
}

func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
    return c.client.Keys(ctx, c.prefixKey(pattern)).Result()
}

func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
    return c.client.Incr(ctx, c.prefixKey(key)).Result()
}

func (c *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
    return c.client.Decr(ctx, c.prefixKey(key)).Result()
}
```

### 3. In-Memory Cache (internal/modules/cache/memory.go)

```go
package cache

import (
    "context"
    "encoding/json"
    "sync"
    "time"
)

type MemoryCache struct {
    data       map[string]*cacheEntry
    mu         sync.RWMutex
    maxMemory  int64
    currentMem int64
    defaultTTL time.Duration
}

type cacheEntry struct {
    data      []byte
    expiresAt time.Time
}

func NewMemoryCache(config CacheConfig) *MemoryCache {
    c := &MemoryCache{
        data:       make(map[string]*cacheEntry),
        maxMemory:  config.MaxMemory,
        defaultTTL: config.DefaultTTL,
    }

    // Start cleanup goroutine
    go c.cleanup()

    return c
}

func (c *MemoryCache) Name() string { return "memory-cache" }

func (c *MemoryCache) Initialize(config *Config) error { return nil }

func (c *MemoryCache) Start(ctx context.Context) error { return nil }

func (c *MemoryCache) Stop(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data = make(map[string]*cacheEntry)
    return nil
}

func (c *MemoryCache) Health() HealthStatus {
    return HealthStatus{Status: "healthy"}
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
    c.mu.RLock()
    entry, ok := c.data[key]
    c.mu.RUnlock()

    if !ok {
        return nil, ErrCacheMiss
    }

    if time.Now().After(entry.expiresAt) {
        c.Delete(ctx, key)
        return nil, ErrCacheMiss
    }

    return entry.data, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    if ttl == 0 {
        ttl = c.defaultTTL
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    // Check memory limit
    if c.maxMemory > 0 {
        newSize := c.currentMem + int64(len(value))
        if old, ok := c.data[key]; ok {
            newSize -= int64(len(old.data))
        }

        if newSize > c.maxMemory {
            c.evict(int64(len(value)))
        }
    }

    c.data[key] = &cacheEntry{
        data:      value,
        expiresAt: time.Now().Add(ttl),
    }

    return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if entry, ok := c.data[key]; ok {
        c.currentMem -= int64(len(entry.data))
        delete(c.data, key)
    }

    return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
    c.mu.RLock()
    entry, ok := c.data[key]
    c.mu.RUnlock()

    if !ok {
        return false, nil
    }

    return time.Now().Before(entry.expiresAt), nil
}

func (c *MemoryCache) GetJSON(ctx context.Context, key string, dest any) error {
    data, err := c.Get(ctx, key)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

func (c *MemoryCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.Set(ctx, key, data, ttl)
}

func (c *MemoryCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (any, error) {
    var cached any
    err := c.GetJSON(ctx, key, &cached)
    if err == nil {
        return cached, nil
    }

    value, err := fn()
    if err != nil {
        return nil, err
    }

    c.SetJSON(ctx, key, value, ttl)
    return value, nil
}

func (c *MemoryCache) cleanup() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, entry := range c.data {
            if now.After(entry.expiresAt) {
                c.currentMem -= int64(len(entry.data))
                delete(c.data, key)
            }
        }
        c.mu.Unlock()
    }
}

func (c *MemoryCache) evict(needed int64) {
    // Simple LRU-like eviction: remove oldest entries
    // In production, use a proper LRU implementation
    for key, entry := range c.data {
        c.currentMem -= int64(len(entry.data))
        delete(c.data, key)
        if c.currentMem+needed <= c.maxMemory {
            break
        }
    }
}

// Implement remaining interface methods...
func (c *MemoryCache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
    results := make([][]byte, len(keys))
    for i, key := range keys {
        data, _ := c.Get(ctx, key)
        results[i] = data
    }
    return results, nil
}

func (c *MemoryCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
    for key, value := range items {
        if err := c.Set(ctx, key, value, ttl); err != nil {
            return err
        }
    }
    return nil
}

func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
    // Simple implementation - could use glob matching
    return nil
}

func (c *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var keys []string
    for key := range c.data {
        keys = append(keys, key)
    }
    return keys, nil
}

func (c *MemoryCache) Incr(ctx context.Context, key string) (int64, error) {
    return 0, fmt.Errorf("not implemented for memory cache")
}

func (c *MemoryCache) Decr(ctx context.Context, key string) (int64, error) {
    return 0, fmt.Errorf("not implemented for memory cache")
}
```

### 4. HTTP Caching Middleware

```go
// internal/modules/cache/middleware.go
package cache

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    "time"
)

type CachingMiddleware struct {
    cache CacheModule
}

func NewCachingMiddleware(cache CacheModule) *CachingMiddleware {
    return &CachingMiddleware{cache: cache}
}

func (m *CachingMiddleware) Handler(ttl time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only cache GET requests
            if r.Method != http.MethodGet {
                next.ServeHTTP(w, r)
                return
            }

            // Generate cache key
            key := m.generateKey(r)

            // Try to get from cache
            cached, err := m.cache.Get(r.Context(), key)
            if err == nil {
                w.Header().Set("X-Cache", "HIT")
                w.Header().Set("Content-Type", "application/json")
                w.Write(cached)
                return
            }

            // Capture response
            rec := &responseRecorder{
                ResponseWriter: w,
                body:          &bytes.Buffer{},
                status:        200,
            }

            next.ServeHTTP(rec, r)

            // Cache successful responses
            if rec.status >= 200 && rec.status < 300 {
                m.cache.Set(r.Context(), key, rec.body.Bytes(), ttl)
            }

            w.Header().Set("X-Cache", "MISS")
        })
    }
}

func (m *CachingMiddleware) generateKey(r *http.Request) string {
    h := sha256.New()
    h.Write([]byte(r.URL.String()))
    return "http:" + hex.EncodeToString(h.Sum(nil))[:16]
}

type responseRecorder struct {
    http.ResponseWriter
    body   *bytes.Buffer
    status int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
    r.body.Write(b)
    return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(status int) {
    r.status = status
    r.ResponseWriter.WriteHeader(status)
}
```

## Acceptance Criteria
- [ ] Redis connection and basic operations
- [ ] In-memory cache fallback
- [ ] JSON serialization helpers
- [ ] GetOrSet cache-aside pattern
- [ ] Key prefix support
- [ ] Pattern-based deletion
- [ ] HTTP caching middleware
- [ ] TTL support

## Testing Strategy
- Unit tests with mock Redis
- Integration tests with Redis container
- Performance benchmarks

## Files to Create
- `internal/modules/cache/module.go`
- `internal/modules/cache/redis.go`
- `internal/modules/cache/memory.go`
- `internal/modules/cache/middleware.go`
- `internal/modules/cache/cache_test.go`
