// Package testing provides test helpers for database tests.
package testing

import (
	"database/sql"
	"testing"

	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SetupTestDB creates an in-memory SQLite database for testing
// and runs all migrations.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Enable foreign keys for SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run migrations
	migrator := database.NewMigrator(db)
	if err := migrator.MigrateUp(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// TeardownTestDB closes the test database connection.
func TeardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := db.Close(); err != nil {
		t.Errorf("failed to close test database: %v", err)
	}
}

// SeedTestData inserts sample test data into the database.
func SeedTestData(t *testing.T, db *sql.DB) *TestData {
	t.Helper()

	data := &TestData{}

	// Create test config
	data.Config = &models.Config{
		ID:      uuid.New().String(),
		Name:    "test-config",
		Content: "deployment test { task run { exec 'echo hello' } }",
	}
	_, err := db.Exec(
		"INSERT INTO configs (id, name, content) VALUES (?, ?, ?)",
		data.Config.ID, data.Config.Name, data.Config.Content,
	)
	if err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	// Create test deployment
	data.Deployment = &models.Deployment{
		ID:     uuid.New().String(),
		Name:   "test-deployment",
		Status: string(models.DeploymentStatusPending),
		ConfigID: sql.NullString{
			String: data.Config.ID,
			Valid:  true,
		},
	}
	_, err = db.Exec(
		"INSERT INTO deployments (id, name, config_id, status) VALUES (?, ?, ?, ?)",
		data.Deployment.ID, data.Deployment.Name, data.Deployment.ConfigID, data.Deployment.Status,
	)
	if err != nil {
		t.Fatalf("failed to seed deployment: %v", err)
	}

	// Create test execution
	data.Execution = &models.Execution{
		ID:           uuid.New().String(),
		DeploymentID: data.Deployment.ID,
		Command:      "echo hello",
	}
	_, err = db.Exec(
		"INSERT INTO executions (id, deployment_id, command) VALUES (?, ?, ?)",
		data.Execution.ID, data.Execution.DeploymentID, data.Execution.Command,
	)
	if err != nil {
		t.Fatalf("failed to seed execution: %v", err)
	}

	return data
}

// TestData holds seeded test data for use in tests.
type TestData struct {
	Config     *models.Config
	Deployment *models.Deployment
	Execution  *models.Execution
}

// GenerateUUID returns a new UUID string for testing.
func GenerateUUID() string {
	return uuid.New().String()
}
