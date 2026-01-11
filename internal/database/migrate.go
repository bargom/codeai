package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration.
type Migration struct {
	Version   string
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

// Migrator handles database migrations.
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// getAppliedMigrations returns a map of applied migration versions.
func (m *Migrator) getAppliedMigrations() (map[string]time.Time, error) {
	if err := m.ensureMigrationsTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}
	return applied, rows.Err()
}

// loadMigrations loads all migrations from the embedded filesystem.
func (m *Migrator) loadMigrations() ([]Migration, error) {
	migrations := make(map[string]*Migration)

	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}

		filename := filepath.Base(path)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			return nil
		}

		version := parts[0]
		rest := parts[1]

		var direction string
		var name string
		if strings.HasSuffix(rest, ".up.sql") {
			direction = "up"
			name = strings.TrimSuffix(rest, ".up.sql")
		} else if strings.HasSuffix(rest, ".down.sql") {
			direction = "down"
			name = strings.TrimSuffix(rest, ".down.sql")
		} else {
			return nil
		}

		content, err := migrationsFS.ReadFile(path)
		if err != nil {
			return err
		}

		mig, ok := migrations[version]
		if !ok {
			mig = &Migration{Version: version, Name: name}
			migrations[version] = mig
		}

		if direction == "up" {
			mig.UpSQL = string(content)
		} else {
			mig.DownSQL = string(content)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to sorted slice
	result := make([]Migration, 0, len(migrations))
	for _, mig := range migrations {
		result = append(result, *mig)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// MigrateUp runs all pending migrations.
func (m *Migrator) MigrateUp() error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	for _, mig := range migrations {
		if _, ok := applied[mig.Version]; ok {
			continue
		}

		if err := m.runMigration(mig, true); err != nil {
			return fmt.Errorf("running migration %s: %w", mig.Version, err)
		}
	}

	return nil
}

// MigrateDown rolls back the last applied migration.
func (m *Migrator) MigrateDown() error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return nil
	}

	// Find the last applied migration
	var lastVersion string
	for version := range applied {
		if version > lastVersion {
			lastVersion = version
		}
	}

	// Find the migration with that version
	for _, mig := range migrations {
		if mig.Version == lastVersion {
			return m.runMigration(mig, false)
		}
	}

	return fmt.Errorf("migration %s not found", lastVersion)
}

// MigrateReset rolls back all migrations and re-applies them.
func (m *Migrator) MigrateReset() error {
	// Roll back all migrations
	for {
		applied, err := m.getAppliedMigrations()
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			break
		}
		if err := m.MigrateDown(); err != nil {
			return err
		}
	}

	// Re-apply all migrations
	return m.MigrateUp()
}

// runMigration executes a single migration.
func (m *Migrator) runMigration(mig Migration, up bool) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sqlToRun string
	if up {
		sqlToRun = mig.UpSQL
	} else {
		sqlToRun = mig.DownSQL
	}

	if sqlToRun == "" {
		return fmt.Errorf("no %s SQL for migration %s", map[bool]string{true: "up", false: "down"}[up], mig.Version)
	}

	// Execute the migration SQL
	if _, err := tx.Exec(sqlToRun); err != nil {
		return err
	}

	// Update schema_migrations table
	if up {
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", mig.Version); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", mig.Version); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Status returns the current migration status.
func (m *Migrator) Status() ([]Migration, error) {
	migrations, err := m.loadMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	for i := range migrations {
		if t, ok := applied[migrations[i].Version]; ok {
			migrations[i].AppliedAt = &t
		}
	}

	return migrations, nil
}
