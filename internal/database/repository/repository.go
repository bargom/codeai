// Package repository implements the repository pattern for data access.
package repository

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("record not found")

// Querier is an interface that can execute queries.
// Both *sql.DB and *sql.Tx implement this interface.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// baseRepository provides common functionality for all repositories.
type baseRepository struct {
	db Querier
}

// newBaseRepository creates a new baseRepository.
func newBaseRepository(db Querier) baseRepository {
	return baseRepository{db: db}
}
