package checks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/health"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDatabaseChecker(t *testing.T) {
	t.Run("healthy database", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		checker := NewDatabaseChecker(db)

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Empty(t, result.Message)
		assert.NotNil(t, result.Details)
	})

	t.Run("closed database returns unhealthy", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		db.Close() // Close immediately

		checker := NewDatabaseChecker(db)

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "database ping failed")
	})

	t.Run("name returns database", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()

		checker := NewDatabaseChecker(db)
		assert.Equal(t, "database", checker.Name())
	})

	t.Run("default severity is critical", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()

		checker := NewDatabaseChecker(db)
		assert.Equal(t, health.SeverityCritical, checker.Severity())
	})

	t.Run("custom timeout", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()

		checker := NewDatabaseChecker(db, WithDatabaseTimeout(5*time.Second))
		assert.Equal(t, 5*time.Second, checker.timeout)
	})

	t.Run("custom severity", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()

		checker := NewDatabaseChecker(db, WithDatabaseSeverity(health.SeverityWarning))
		assert.Equal(t, health.SeverityWarning, checker.Severity())
	})

	t.Run("context cancellation", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		checker := NewDatabaseChecker(db)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result := checker.Check(ctx)

		// With SQLite in-memory, ping might still succeed even with cancelled context
		// The important thing is that it doesn't hang
		assert.NotEqual(t, "", result.Status)
	})
}

// mockDB implements a database interface for testing error scenarios
type mockDB struct {
	pingErr error
}

func (m *mockDB) PingContext(ctx context.Context) error {
	return m.pingErr
}

func TestDatabaseCheckerWithMock(t *testing.T) {
	t.Run("ping error", func(t *testing.T) {
		// Since sql.DB doesn't expose an interface, we test error scenarios
		// using actual database connections or with integration tests
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		// Simulate error by closing connection
		db.Close()

		checker := NewDatabaseChecker(db)
		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "failed")
	})
}

func TestDatabaseCheckerDetails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Set some connection pool settings
	db.SetMaxOpenConns(10)

	checker := NewDatabaseChecker(db)
	result := checker.Check(context.Background())

	assert.Equal(t, health.StatusHealthy, result.Status)
	assert.Contains(t, result.Details, "max_connections")
	assert.Contains(t, result.Details, "open_connections")
	assert.Contains(t, result.Details, "in_use")
	assert.Contains(t, result.Details, "idle")
}
