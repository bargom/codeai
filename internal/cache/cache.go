// Package cache provides caching functionality with Redis and in-memory backends.
package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when a key is not found in the cache.
var ErrCacheMiss = errors.New("cache miss")

// Cache defines the interface for cache operations.
type Cache interface {
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

	// Lifecycle
	Close() error
	Health(ctx context.Context) error

	// Stats
	Stats() Stats
}

// Stats holds cache statistics.
type Stats struct {
	Hits       int64
	Misses     int64
	Keys       int64
	MemoryUsed int64
}

// Config holds cache configuration.
type Config struct {
	// Type is the cache backend type: "redis" or "memory"
	Type string

	// Redis configuration
	URL      string // Redis URL (redis://localhost:6379)
	Password string
	DB       int

	// Cluster configuration
	ClusterAddrs []string
	ClusterMode  bool

	// Connection pool settings
	PoolSize     int
	MinIdleConns int
	MaxRetries   int

	// General settings
	DefaultTTL time.Duration
	Prefix     string

	// Memory cache settings
	MaxMemory int64 // Maximum memory in bytes (0 = unlimited)
	MaxItems  int   // Maximum number of items (0 = unlimited)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Type:         "memory",
		DefaultTTL:   5 * time.Minute,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		MaxMemory:    100 * 1024 * 1024, // 100MB
		MaxItems:     10000,
	}
}

// New creates a new cache instance based on configuration.
func New(cfg Config) (Cache, error) {
	switch cfg.Type {
	case "redis":
		return NewRedisCache(cfg)
	case "memory", "":
		return NewMemoryCache(cfg), nil
	default:
		return nil, errors.New("unsupported cache type: " + cfg.Type)
	}
}
