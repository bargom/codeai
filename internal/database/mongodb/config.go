// Package mongodb provides MongoDB database connectivity and operations.
package mongodb

import (
	"fmt"
	"time"
)

// Config holds MongoDB connection configuration.
type Config struct {
	// URI is the MongoDB connection string (e.g., mongodb://localhost:27017)
	URI string

	// Database is the name of the database to connect to
	Database string

	// MinPoolSize is the minimum number of connections in the pool
	MinPoolSize uint64

	// MaxPoolSize is the maximum number of connections in the pool
	MaxPoolSize uint64

	// ConnectTimeout is the timeout for establishing a connection
	ConnectTimeout time.Duration

	// SocketTimeout is the timeout for socket reads/writes
	SocketTimeout time.Duration

	// ServerSelectionTimeout is the timeout for server selection
	ServerSelectionTimeout time.Duration

	// RetryWrites enables retryable writes
	RetryWrites bool

	// RetryReads enables retryable reads
	RetryReads bool

	// MaxRetries is the maximum number of retry attempts for transient failures
	MaxRetries int

	// RetryBackoff is the base backoff duration between retries
	RetryBackoff time.Duration

	// MaxRetryBackoff is the maximum backoff duration between retries
	MaxRetryBackoff time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		URI:                    "mongodb://localhost:27017",
		Database:               "codeai",
		MinPoolSize:            5,
		MaxPoolSize:            100,
		ConnectTimeout:         10 * time.Second,
		SocketTimeout:          30 * time.Second,
		ServerSelectionTimeout: 5 * time.Second,
		RetryWrites:            true,
		RetryReads:             true,
		MaxRetries:             3,
		RetryBackoff:           100 * time.Millisecond,
		MaxRetryBackoff:        5 * time.Second,
	}
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("mongodb: URI is required")
	}
	if c.Database == "" {
		return fmt.Errorf("mongodb: Database name is required")
	}
	if c.MinPoolSize > c.MaxPoolSize {
		return fmt.Errorf("mongodb: MinPoolSize (%d) cannot be greater than MaxPoolSize (%d)",
			c.MinPoolSize, c.MaxPoolSize)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("mongodb: MaxRetries cannot be negative")
	}
	return nil
}
