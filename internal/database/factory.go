// Package database provides database connectivity and operations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bargom/codeai/internal/database/mongodb"
)

// Connection represents a database connection that can be either PostgreSQL or MongoDB.
// For connections with repository support, use the setup package instead.
type Connection interface {
	// Type returns the database type.
	Type() DatabaseType
	// Close closes the database connection.
	Close() error
	// Ping verifies the database connection is alive.
	Ping() error
}

// PostgresConnection wraps a PostgreSQL database connection.
type PostgresConnection struct {
	DB *sql.DB
}

// Type returns DatabaseTypePostgres.
func (c *PostgresConnection) Type() DatabaseType {
	return DatabaseTypePostgres
}

// Close closes the PostgreSQL connection.
func (c *PostgresConnection) Close() error {
	return Close(c.DB)
}

// Ping verifies the PostgreSQL connection is alive.
func (c *PostgresConnection) Ping() error {
	return Ping(c.DB)
}

// MongoDBConnection wraps a MongoDB connection.
type MongoDBConnection struct {
	Client *mongodb.Client
}

// Type returns DatabaseTypeMongoDB.
func (c *MongoDBConnection) Type() DatabaseType {
	return DatabaseTypeMongoDB
}

// Close closes the MongoDB connection.
func (c *MongoDBConnection) Close() error {
	if c.Client != nil {
		return c.Client.Close(context.Background())
	}
	return nil
}

// Ping verifies the MongoDB connection is alive.
func (c *MongoDBConnection) Ping() error {
	if c.Client != nil {
		return c.Client.Database().Client().Ping(context.Background(), nil)
	}
	return nil
}

// NewConnection creates a new database connection based on the configuration.
// For PostgreSQL, it returns a PostgresConnection with an active *sql.DB.
// For MongoDB, it returns a MongoDBConnection with an active client.
// Note: For connections with repository initialization, use the setup package.
func NewConnection(cfg DatabaseConfig) (Connection, error) {
	return NewConnectionWithLogger(cfg, nil)
}

// NewConnectionWithLogger creates a new database connection with a custom logger.
func NewConnectionWithLogger(cfg DatabaseConfig, logger *slog.Logger) (Connection, error) {
	if logger == nil {
		logger = slog.Default()
	}

	switch cfg.Type {
	case DatabaseTypePostgres:
		if cfg.Postgres == nil {
			return nil, fmt.Errorf("postgres configuration is required for postgres database type")
		}
		db, err := Connect(Config{
			Host:     cfg.Postgres.Host,
			Port:     cfg.Postgres.Port,
			Database: cfg.Postgres.Database,
			User:     cfg.Postgres.User,
			Password: cfg.Postgres.Password,
			SSLMode:  cfg.Postgres.SSLMode,
		})
		if err != nil {
			return nil, fmt.Errorf("connecting to postgres: %w", err)
		}

		return &PostgresConnection{DB: db}, nil

	case DatabaseTypeMongoDB:
		if cfg.MongoDB == nil {
			return nil, fmt.Errorf("mongodb configuration is required for mongodb database type")
		}

		// Create MongoDB client with configuration
		mongoCfg := mongodb.DefaultConfig()
		mongoCfg.URI = cfg.MongoDB.URI
		mongoCfg.Database = cfg.MongoDB.Database

		client, err := mongodb.New(context.Background(), mongoCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("connecting to mongodb: %w", err)
		}

		return &MongoDBConnection{Client: client}, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// MustNewConnection creates a new database connection and panics on error.
func MustNewConnection(cfg DatabaseConfig) Connection {
	conn, err := NewConnection(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create database connection: %v", err))
	}
	return conn
}

// MustNewConnectionWithLogger creates a new database connection with a logger and panics on error.
func MustNewConnectionWithLogger(cfg DatabaseConfig, logger *slog.Logger) Connection {
	conn, err := NewConnectionWithLogger(cfg, logger)
	if err != nil {
		panic(fmt.Sprintf("failed to create database connection: %v", err))
	}
	return conn
}
