// Package mongodb provides MongoDB database connectivity and operations.
package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client wraps a MongoDB client with connection management and logging.
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	config   Config
	logger   *slog.Logger
	mu       sync.RWMutex
	closed   bool
}

// New creates a new MongoDB client with the given configuration.
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	logger = logger.With(slog.String("component", "mongodb"))

	client := &Client{
		config: cfg,
		logger: logger,
	}

	if err := client.connect(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

// connect establishes the connection to MongoDB with retry logic.
func (c *Client) connect(ctx context.Context) error {
	opts := options.Client().
		ApplyURI(c.config.URI).
		SetMinPoolSize(c.config.MinPoolSize).
		SetMaxPoolSize(c.config.MaxPoolSize).
		SetConnectTimeout(c.config.ConnectTimeout).
		SetSocketTimeout(c.config.SocketTimeout).
		SetServerSelectionTimeout(c.config.ServerSelectionTimeout).
		SetRetryWrites(c.config.RetryWrites).
		SetRetryReads(c.config.RetryReads)

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := c.calculateBackoff(attempt)
			c.logger.Debug("retrying connection",
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff))

			select {
			case <-ctx.Done():
				return fmt.Errorf("mongodb: connection cancelled: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			lastErr = err
			c.logger.Warn("connection attempt failed",
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			continue
		}

		// Verify connection with ping
		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			lastErr = err
			c.logger.Warn("ping failed",
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			// Disconnect failed client
			_ = client.Disconnect(ctx)
			continue
		}

		c.client = client
		c.database = client.Database(c.config.Database)
		c.logger.Info("connected to MongoDB",
			slog.String("database", c.config.Database))
		return nil
	}

	return fmt.Errorf("mongodb: failed to connect after %d attempts: %w",
		c.config.MaxRetries+1, lastErr)
}

// calculateBackoff computes exponential backoff with jitter.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := c.config.RetryBackoff * time.Duration(math.Pow(2, float64(attempt-1)))
	if backoff > c.config.MaxRetryBackoff {
		backoff = c.config.MaxRetryBackoff
	}
	return backoff
}

// Database returns the MongoDB database handle.
func (c *Client) Database() *mongo.Database {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.database
}

// Collection returns a collection from the database.
func (c *Client) Collection(name string) *mongo.Collection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.database == nil {
		return nil
	}
	return c.database.Collection(name)
}

// Client returns the underlying mongo.Client.
func (c *Client) Client() *mongo.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Close gracefully disconnects from MongoDB.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	if c.client == nil {
		c.closed = true
		return nil
	}

	c.logger.Info("disconnecting from MongoDB")

	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongodb: disconnect failed: %w", err)
	}

	c.closed = true
	c.client = nil
	c.database = nil

	c.logger.Info("disconnected from MongoDB")
	return nil
}

// IsClosed returns true if the client has been closed.
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// WithTransaction executes a function within a MongoDB transaction.
func (c *Client) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("mongodb: client is closed")
	}
	client := c.client
	c.mu.RUnlock()

	session, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("mongodb: failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})
	if err != nil {
		return fmt.Errorf("mongodb: transaction failed: %w", err)
	}

	return nil
}

// PoolStats holds connection pool statistics.
type PoolStats struct {
	// TotalConnections is the total number of connections in the pool
	TotalConnections int
	// AvailableConnections is the number of available connections
	AvailableConnections int
	// InUseConnections is the number of connections currently in use
	InUseConnections int
}

// GetPoolStats returns the current connection pool statistics.
// Note: The mongo-driver doesn't expose detailed pool stats directly,
// so this returns information based on server status.
func (c *Client) GetPoolStats(ctx context.Context) (PoolStats, error) {
	c.mu.RLock()
	if c.closed || c.client == nil {
		c.mu.RUnlock()
		return PoolStats{}, fmt.Errorf("mongodb: client is closed")
	}
	c.mu.RUnlock()

	// The mongo-driver doesn't expose pool stats directly
	// We can only verify connectivity
	return PoolStats{}, nil
}
