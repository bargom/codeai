// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"
	"net/http"

	"github.com/bargom/codeai/internal/database"
)

// contextKey is a type for context keys used by this package.
type contextKey string

const (
	// connectionKey is the context key for the database connection.
	connectionKey contextKey = "database_connection"
	// databaseTypeKey is the context key for the database type.
	databaseTypeKey contextKey = "database_type"
)

// WithConnection is middleware that injects a database connection into the request context.
// This is useful when you need to access the database type or connection in handlers.
func WithConnection(conn database.Connection) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), connectionKey, conn)
			ctx = context.WithValue(ctx, databaseTypeKey, conn.Type())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ConnectionFromContext retrieves the database connection from the context.
// Returns nil if no connection is present.
func ConnectionFromContext(ctx context.Context) database.Connection {
	if conn, ok := ctx.Value(connectionKey).(database.Connection); ok {
		return conn
	}
	return nil
}

// DatabaseTypeFromContext retrieves the database type from the context.
// Returns an empty string if no type is present.
func DatabaseTypeFromContext(ctx context.Context) database.DatabaseType {
	if dbType, ok := ctx.Value(databaseTypeKey).(database.DatabaseType); ok {
		return dbType
	}
	return ""
}
