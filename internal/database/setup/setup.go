// Package setup provides database connection setup with repository initialization.
// This package bridges the database and repository packages to avoid import cycles.
package setup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/mongodb"
	"github.com/bargom/codeai/internal/database/repository"
)

// Connection represents a database connection with repositories.
type Connection interface {
	// Type returns the database type.
	Type() database.DatabaseType
	// Close closes the database connection.
	Close() error
	// Ping verifies the database connection is alive.
	Ping() error
	// Repositories returns the repository interfaces for this connection.
	Repositories() *repository.Repositories
	// DB returns the raw sql.DB for PostgreSQL connections (nil for MongoDB).
	DB() *sql.DB
	// MongoClient returns the MongoDB client for MongoDB connections (nil for PostgreSQL).
	MongoClient() *mongodb.Client
}

// PostgresConnection wraps a PostgreSQL database connection with repositories.
type PostgresConnection struct {
	db    *sql.DB
	repos *repository.Repositories
}

// Type returns DatabaseTypePostgres.
func (c *PostgresConnection) Type() database.DatabaseType {
	return database.DatabaseTypePostgres
}

// Close closes the PostgreSQL connection.
func (c *PostgresConnection) Close() error {
	return database.Close(c.db)
}

// Ping verifies the PostgreSQL connection is alive.
func (c *PostgresConnection) Ping() error {
	return database.Ping(c.db)
}

// Repositories returns the repository interfaces for PostgreSQL.
func (c *PostgresConnection) Repositories() *repository.Repositories {
	return c.repos
}

// DB returns the raw sql.DB connection.
func (c *PostgresConnection) DB() *sql.DB {
	return c.db
}

// MongoClient returns nil for PostgreSQL connections.
func (c *PostgresConnection) MongoClient() *mongodb.Client {
	return nil
}

// MongoDBConnection wraps a MongoDB connection with repository adapters.
type MongoDBConnection struct {
	client *mongodb.Client
	repos  *repository.Repositories
}

// Type returns DatabaseTypeMongoDB.
func (c *MongoDBConnection) Type() database.DatabaseType {
	return database.DatabaseTypeMongoDB
}

// Close closes the MongoDB connection.
func (c *MongoDBConnection) Close() error {
	if c.client != nil {
		return c.client.Close(context.Background())
	}
	return nil
}

// Ping verifies the MongoDB connection is alive.
func (c *MongoDBConnection) Ping() error {
	if c.client != nil {
		return c.client.Database().Client().Ping(context.Background(), nil)
	}
	return nil
}

// Repositories returns the repository interfaces for MongoDB.
func (c *MongoDBConnection) Repositories() *repository.Repositories {
	return c.repos
}

// DB returns nil for MongoDB connections.
func (c *MongoDBConnection) DB() *sql.DB {
	return nil
}

// MongoClient returns the MongoDB client.
func (c *MongoDBConnection) MongoClient() *mongodb.Client {
	return c.client
}

// NewConnection creates a new database connection based on the configuration.
// For PostgreSQL, it returns a PostgresConnection with an active *sql.DB.
// For MongoDB, it returns a MongoDBConnection with an active client.
func NewConnection(cfg database.DatabaseConfig) (Connection, error) {
	return NewConnectionWithLogger(cfg, nil)
}

// NewConnectionWithLogger creates a new database connection with a custom logger.
func NewConnectionWithLogger(cfg database.DatabaseConfig, logger *slog.Logger) (Connection, error) {
	if logger == nil {
		logger = slog.Default()
	}

	switch cfg.Type {
	case database.DatabaseTypePostgres:
		if cfg.Postgres == nil {
			return nil, fmt.Errorf("postgres configuration is required for postgres database type")
		}
		db, err := database.Connect(database.Config{
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

		// Create PostgreSQL repositories
		deployments := repository.NewDeploymentRepository(db)
		configs := repository.NewConfigRepository(db)
		executions := repository.NewExecutionRepository(db)

		return &PostgresConnection{
			db:    db,
			repos: repository.NewRepositories(deployments, configs, executions),
		}, nil

	case database.DatabaseTypeMongoDB:
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

		// Create MongoDB repository adapters
		deployments := mongodb.NewDeploymentAdapter(client, logger)
		configs := mongodb.NewConfigAdapter(client, logger)
		executions := mongodb.NewExecutionAdapter(client, logger)

		return &MongoDBConnection{
			client: client,
			repos: &repository.Repositories{
				Deployments: deployments,
				Configs:     configs,
				Executions:  executions,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// MustNewConnection creates a new database connection and panics on error.
func MustNewConnection(cfg database.DatabaseConfig) Connection {
	conn, err := NewConnection(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create database connection: %v", err))
	}
	return conn
}

// MustNewConnectionWithLogger creates a new database connection with a logger and panics on error.
func MustNewConnectionWithLogger(cfg database.DatabaseConfig, logger *slog.Logger) Connection {
	conn, err := NewConnectionWithLogger(cfg, logger)
	if err != nil {
		panic(fmt.Sprintf("failed to create database connection: %v", err))
	}
	return conn
}
