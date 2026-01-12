package hooks

import (
	"context"

	"github.com/bargom/codeai/internal/shutdown"
)

// CacheCloser defines the interface for a cache connection that can be closed.
type CacheCloser interface {
	Close() error
}

// CacheFlusher defines the interface for a cache that can flush data before closing.
type CacheFlusher interface {
	Flush() error
	Close() error
}

// CacheShutdownFunc creates a shutdown hook for a cache connection.
func CacheShutdownFunc(cache CacheCloser) shutdown.HookFunc {
	return func(ctx context.Context) error {
		return cache.Close()
	}
}

// CacheShutdown creates a shutdown hook for a cache connection.
func CacheShutdown(name string, cache CacheCloser) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityCache,
		Fn:       CacheShutdownFunc(cache),
	}
}

// CacheFlushAndShutdownFunc creates a shutdown hook that flushes then closes a cache.
func CacheFlushAndShutdownFunc(cache CacheFlusher) shutdown.HookFunc {
	return func(ctx context.Context) error {
		// Flush first to ensure all data is written
		if err := cache.Flush(); err != nil {
			// Log but don't fail - still try to close
			// Note: In production, this would use a logger
		}
		return cache.Close()
	}
}

// CacheFlushAndShutdown creates a shutdown hook that flushes then closes a cache.
func CacheFlushAndShutdown(name string, cache CacheFlusher) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityCache,
		Fn:       CacheFlushAndShutdownFunc(cache),
	}
}

// RedisClient defines the interface for a Redis client that can be closed.
type RedisClient interface {
	Close() error
}

// RedisShutdown creates a shutdown hook for a Redis client.
func RedisShutdown(client RedisClient) shutdown.Hook {
	return shutdown.Hook{
		Name:     "redis",
		Priority: shutdown.PriorityCache,
		Fn: func(ctx context.Context) error {
			return client.Close()
		},
	}
}

// MemcachedClient defines the interface for a Memcached client that can be closed.
type MemcachedClient interface {
	Close()
}

// MemcachedShutdown creates a shutdown hook for a Memcached client.
func MemcachedShutdown(client MemcachedClient) shutdown.Hook {
	return shutdown.Hook{
		Name:     "memcached",
		Priority: shutdown.PriorityCache,
		Fn: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	}
}
