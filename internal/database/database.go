// Package database provides database connectivity and operations.
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config holds database connection configuration.
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:    "localhost",
		Port:    5432,
		SSLMode: "disable",
	}
}

// Connect establishes a connection to the PostgreSQL database.
func Connect(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}

// ConnectWithDSN connects using a full DSN string.
func ConnectWithDSN(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}

// Ping verifies the database connection is alive.
func Ping(db *sql.DB) error {
	if err := db.Ping(); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}
	return nil
}

// Close closes the database connection.
func Close(db *sql.DB) error {
	if err := db.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	return nil
}

// PoolStats returns current connection pool statistics.
type PoolStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
}

// GetPoolStats returns the current connection pool statistics.
func GetPoolStats(db *sql.DB) PoolStats {
	stats := db.Stats()
	return PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
	}
}
