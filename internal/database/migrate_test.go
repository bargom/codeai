package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	return db
}

func TestMigrator_MigrateUp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	t.Run("applies all migrations", func(t *testing.T) {
		err := migrator.MigrateUp()
		require.NoError(t, err)

		// Verify tables exist
		tables := []string{"configs", "deployments", "executions", "schema_migrations"}
		for _, table := range tables {
			var name string
			err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
			require.NoError(t, err, "table %s should exist", table)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		// Running migrations again should not error
		err := migrator.MigrateUp()
		require.NoError(t, err)
	})
}

func TestMigrator_MigrateDown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	t.Run("rolls back last migration", func(t *testing.T) {
		// First apply migrations
		err := migrator.MigrateUp()
		require.NoError(t, err)

		// Then roll back
		err = migrator.MigrateDown()
		require.NoError(t, err)

		// Verify tables are dropped
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('configs', 'deployments', 'executions')").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("does nothing when no migrations applied", func(t *testing.T) {
		db2 := setupTestDB(t)
		defer db2.Close()

		migrator2 := NewMigrator(db2)
		err := migrator2.MigrateDown()
		require.NoError(t, err)
	})
}

func TestMigrator_MigrateReset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	t.Run("resets all migrations", func(t *testing.T) {
		// Apply migrations
		err := migrator.MigrateUp()
		require.NoError(t, err)

		// Insert some data
		_, err = db.Exec("INSERT INTO configs (id, name, content) VALUES ('test-id', 'test', 'content')")
		require.NoError(t, err)

		// Reset
		err = migrator.MigrateReset()
		require.NoError(t, err)

		// Verify tables exist but are empty
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM configs").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestMigrator_Status(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	t.Run("shows pending migrations before applying", func(t *testing.T) {
		migrations, err := migrator.Status()
		require.NoError(t, err)
		require.NotEmpty(t, migrations)

		for _, m := range migrations {
			assert.Nil(t, m.AppliedAt)
		}
	})

	t.Run("shows applied migrations after applying", func(t *testing.T) {
		err := migrator.MigrateUp()
		require.NoError(t, err)

		migrations, err := migrator.Status()
		require.NoError(t, err)
		require.NotEmpty(t, migrations)

		for _, m := range migrations {
			assert.NotNil(t, m.AppliedAt)
		}
	})
}

func TestMigrator_LoadMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	migrations, err := migrator.loadMigrations()
	require.NoError(t, err)
	require.NotEmpty(t, migrations)

	// Verify each migration has both up and down SQL
	for _, m := range migrations {
		assert.NotEmpty(t, m.Version, "migration should have version")
		assert.NotEmpty(t, m.Name, "migration should have name")
		assert.NotEmpty(t, m.UpSQL, "migration should have up SQL")
		assert.NotEmpty(t, m.DownSQL, "migration should have down SQL")
	}
}
