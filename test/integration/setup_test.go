//go:build integration

// Package integration provides integration tests for the CodeAI system.
package integration

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bargom/codeai/internal/api"
	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	_ "modernc.org/sqlite"
)

// TestSuite holds shared resources for integration tests.
type TestSuite struct {
	DB           *sql.DB
	Server       *httptest.Server
	Handler      *handlers.Handler
	ConfigRepo   *repository.ConfigRepository
	DeployRepo   *repository.DeploymentRepository
	ExecRepo     *repository.ExecutionRepository
}

// SetupTestSuite creates a new test suite with all resources initialized.
func SetupTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	suite := &TestSuite{}

	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	suite.DB = db

	// Enable foreign keys for SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run migrations
	migrator := database.NewMigrator(db)
	if err := migrator.MigrateUp(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Setup repositories
	suite.ConfigRepo = repository.NewConfigRepository(db)
	suite.DeployRepo = repository.NewDeploymentRepository(db)
	suite.ExecRepo = repository.NewExecutionRepository(db)

	// Setup API handler
	suite.Handler = handlers.NewHandler(suite.DeployRepo, suite.ConfigRepo, suite.ExecRepo)

	// Setup API router and test server
	router := api.NewRouter(suite.Handler)
	suite.Server = httptest.NewServer(router)

	return suite
}

// Teardown cleans up all test resources.
func (s *TestSuite) Teardown(t *testing.T) {
	t.Helper()

	if s.Server != nil {
		s.Server.Close()
	}

	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	}
}

// TestMain provides setup and teardown for all integration tests.
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// Helper functions for tests

// CreateTestConfig creates a test config in the database.
func (s *TestSuite) CreateTestConfig(t *testing.T, name, content string) *models.Config {
	t.Helper()

	cfg := models.NewConfig(name, content)
	if err := s.ConfigRepo.Create(context.Background(), cfg); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	return cfg
}

// CreateTestDeployment creates a test deployment in the database.
func (s *TestSuite) CreateTestDeployment(t *testing.T, name string, configID *string) *models.Deployment {
	t.Helper()

	deploy := models.NewDeployment(name)
	if configID != nil {
		deploy.ConfigID = sql.NullString{String: *configID, Valid: true}
	}
	if err := s.DeployRepo.Create(context.Background(), deploy); err != nil {
		t.Fatalf("failed to create test deployment: %v", err)
	}
	return deploy
}

// CreateTestExecution creates a test execution in the database.
func (s *TestSuite) CreateTestExecution(t *testing.T, deploymentID, command string) *models.Execution {
	t.Helper()

	exec := models.NewExecution(deploymentID, command)
	if err := s.ExecRepo.Create(context.Background(), exec); err != nil {
		t.Fatalf("failed to create test execution: %v", err)
	}
	return exec
}
