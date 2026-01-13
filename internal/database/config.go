// Package database provides database connectivity and operations.
package database

import (
	"os"
	"strings"
)

// DatabaseType represents the supported database backends.
type DatabaseType string

const (
	// DatabaseTypePostgres represents PostgreSQL database.
	DatabaseTypePostgres DatabaseType = "postgres"
	// DatabaseTypeMongoDB represents MongoDB database.
	DatabaseTypeMongoDB DatabaseType = "mongodb"
)

// String returns the string representation of DatabaseType.
func (dt DatabaseType) String() string {
	return string(dt)
}

// IsValid returns true if the database type is valid.
func (dt DatabaseType) IsValid() bool {
	switch dt {
	case DatabaseTypePostgres, DatabaseTypeMongoDB:
		return true
	default:
		return false
	}
}

// ParseDatabaseType parses a string into a DatabaseType.
// Returns DatabaseTypePostgres if the input is empty or invalid.
func ParseDatabaseType(s string) DatabaseType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "mongodb", "mongo":
		return DatabaseTypeMongoDB
	case "postgres", "postgresql", "pg":
		return DatabaseTypePostgres
	default:
		return DatabaseTypePostgres // default to postgres for backward compatibility
	}
}

// DatabaseConfig holds unified database configuration for any backend.
type DatabaseConfig struct {
	// Type specifies which database backend to use
	Type DatabaseType

	// PostgreSQL-specific configuration
	Postgres *PostgresConfig

	// MongoDB-specific configuration
	MongoDB *MongoDBConfig
}

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// DefaultPostgresConfig returns a PostgresConfig with sensible defaults.
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:    "localhost",
		Port:    5432,
		SSLMode: "disable",
	}
}

// MongoDBConfig holds MongoDB connection configuration.
type MongoDBConfig struct {
	URI      string
	Database string
}

// DefaultMongoDBConfig returns a MongoDBConfig with sensible defaults.
func DefaultMongoDBConfig() *MongoDBConfig {
	return &MongoDBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "codeai",
	}
}

// DefaultDatabaseConfig returns a DatabaseConfig with sensible defaults.
// Uses PostgreSQL by default for backward compatibility.
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Type:     DatabaseTypePostgres,
		Postgres: DefaultPostgresConfig(),
		MongoDB:  DefaultMongoDBConfig(),
	}
}

// DatabaseConfigFromEnv creates a DatabaseConfig from environment variables.
// Environment variables:
//   - DATABASE_TYPE: "postgres" or "mongodb" (default: "postgres")
//   - For PostgreSQL: DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSL_MODE
//   - For MongoDB: MONGODB_URI, MONGODB_DATABASE
func DatabaseConfigFromEnv() DatabaseConfig {
	cfg := DefaultDatabaseConfig()

	// Read database type from environment
	if dbType := os.Getenv("DATABASE_TYPE"); dbType != "" {
		cfg.Type = ParseDatabaseType(dbType)
	}

	// PostgreSQL environment variables
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Postgres.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		// Parse port, keeping default if invalid
		var p int
		if _, err := parseIntEnv(port, &p); err == nil && p > 0 {
			cfg.Postgres.Port = p
		}
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.Postgres.Database = name
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.Postgres.User = user
	}
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		cfg.Postgres.Password = pass
	}
	if ssl := os.Getenv("DB_SSL_MODE"); ssl != "" {
		cfg.Postgres.SSLMode = ssl
	}

	// MongoDB environment variables
	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		cfg.MongoDB.URI = uri
	}
	if database := os.Getenv("MONGODB_DATABASE"); database != "" {
		cfg.MongoDB.Database = database
	}

	return cfg
}

// parseIntEnv is a helper to parse an integer from a string.
func parseIntEnv(s string, result *int) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil // invalid
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return n, nil
}
