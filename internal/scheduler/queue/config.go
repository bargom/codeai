// Package queue provides a job queue manager using Asynq.
package queue

import (
	"time"
)

// Config holds queue configuration.
type Config struct {
	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Server configuration
	Concurrency int
	Queues      map[string]int // queue name -> priority

	// Retry configuration
	MaxRetry       int
	RetryDelayFunc func(n int, err error, task Task) time.Duration

	// Shutdown configuration
	ShutdownTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		RedisAddr:       "localhost:6379",
		RedisDB:         0,
		Concurrency:     10,
		Queues:          map[string]int{"critical": 6, "default": 3, "low": 1},
		MaxRetry:        3,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Queue priority constants.
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)
