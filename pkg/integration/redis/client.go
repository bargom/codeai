// Package redis provides a Redis client wrapper for the scheduler.
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection configuration.
type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// Client wraps the go-redis client with additional functionality.
type Client struct {
	rdb    *redis.Client
	config Config
	mu     sync.RWMutex
}

// NewClient creates a new Redis client.
func NewClient(cfg Config) (*Client, error) {
	opts := &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	rdb := redis.NewClient(opts)

	return &Client{
		rdb:    rdb,
		config: cfg,
	}, nil
}

// Ping verifies the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("redis close: %w", err)
	}
	return nil
}

// RedisClient returns the underlying go-redis client for direct access.
func (c *Client) RedisClient() *redis.Client {
	return c.rdb
}

// GetRedisAddr returns the Redis address for Asynq client configuration.
func (c *Client) GetRedisAddr() string {
	return c.config.Addr
}

// GetRedisPassword returns the Redis password for Asynq client configuration.
func (c *Client) GetRedisPassword() string {
	return c.config.Password
}

// GetRedisDB returns the Redis database number for Asynq client configuration.
func (c *Client) GetRedisDB() int {
	return c.config.DB
}

// Health performs a health check on the Redis connection.
func (c *Client) Health(ctx context.Context) error {
	return c.Ping(ctx)
}
