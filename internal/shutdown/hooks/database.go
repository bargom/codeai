package hooks

import (
	"context"

	"github.com/bargom/codeai/internal/shutdown"
)

// DatabaseCloser defines the interface for a database connection that can be closed.
type DatabaseCloser interface {
	Close() error
}

// DatabaseContextCloser defines the interface for a database connection that can be closed with context.
type DatabaseContextCloser interface {
	Close(ctx context.Context) error
}

// DatabaseShutdownFunc creates a shutdown hook for a database connection.
func DatabaseShutdownFunc(db DatabaseCloser) shutdown.HookFunc {
	return func(ctx context.Context) error {
		return db.Close()
	}
}

// DatabaseShutdown creates a shutdown hook for a database connection.
func DatabaseShutdown(name string, db DatabaseCloser) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityDatabase,
		Fn:       DatabaseShutdownFunc(db),
	}
}

// DatabaseContextShutdownFunc creates a shutdown hook for a database with context close.
func DatabaseContextShutdownFunc(db DatabaseContextCloser) shutdown.HookFunc {
	return func(ctx context.Context) error {
		return db.Close(ctx)
	}
}

// DatabaseContextShutdown creates a shutdown hook for a database with context close.
func DatabaseContextShutdown(name string, db DatabaseContextCloser) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityDatabase,
		Fn:       DatabaseContextShutdownFunc(db),
	}
}

// SQLDBShutdown creates a shutdown hook for a *sql.DB compatible database.
type SQLDBCloser interface {
	Close() error
}

// SQLDBShutdown creates a shutdown hook for a SQL database.
func SQLDBShutdown(db SQLDBCloser) shutdown.Hook {
	return shutdown.Hook{
		Name:     "database",
		Priority: shutdown.PriorityDatabase,
		Fn: func(ctx context.Context) error {
			return db.Close()
		},
	}
}

// MongoDBShutdown creates a shutdown hook for a MongoDB client.
// MongoDB client uses Disconnect(ctx) instead of Close().
type MongoDBClient interface {
	Disconnect(ctx context.Context) error
}

// MongoDBShutdown creates a shutdown hook for a MongoDB client.
func MongoDBShutdown(client MongoDBClient) shutdown.Hook {
	return shutdown.Hook{
		Name:     "mongodb",
		Priority: shutdown.PriorityDatabase,
		Fn: func(ctx context.Context) error {
			return client.Disconnect(ctx)
		},
	}
}
