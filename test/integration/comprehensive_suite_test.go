//go:build integration

// Package integration provides comprehensive integration tests for the CodeAI system.
package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/event/bus"
	emailrepo "github.com/bargom/codeai/internal/notification/email/repository"
	schedrepo "github.com/bargom/codeai/internal/scheduler/repository"
	webhookrepo "github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/test/integration/testutil"
	_ "modernc.org/sqlite"
)

// ComprehensiveTestSuite holds all services and repositories for integration testing.
type ComprehensiveTestSuite struct {
	// Database
	DB *sql.DB

	// Repositories
	JobRepository     schedrepo.JobRepository
	WebhookRepository webhookrepo.WebhookRepository
	EmailRepository   emailrepo.EmailRepository

	// Event infrastructure
	EventDispatcher event.Dispatcher
	EventBus        *bus.EventBus

	// Test helpers
	WebhookServer *testutil.WebhookTestServer
	EmailClient   *testutil.MockEmailClient
	Fixtures      *testutil.FixtureBuilder

	// Context
	Ctx    context.Context
	Cancel context.CancelFunc

	// Cleanup functions
	cleanup []func()
}

// NewComprehensiveTestSuite creates a new comprehensive test suite.
func NewComprehensiveTestSuite(t *testing.T) *ComprehensiveTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	suite := &ComprehensiveTestSuite{
		Ctx:      ctx,
		Cancel:   cancel,
		cleanup:  make([]func(), 0),
		Fixtures: testutil.NewFixtureBuilder(),
	}

	// Setup SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	suite.DB = db
	suite.cleanup = append(suite.cleanup, func() { db.Close() })

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run migrations
	migrator := database.NewMigrator(db)
	if err := migrator.MigrateUp(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Setup in-memory repositories
	suite.JobRepository = schedrepo.NewMemoryJobRepository()
	suite.WebhookRepository = webhookrepo.NewMemoryRepository()
	suite.EmailRepository = emailrepo.NewMemoryEmailRepository()

	// Setup event infrastructure
	suite.EventDispatcher = event.NewDispatcher()
	suite.EventBus = bus.NewEventBus(nil)
	suite.cleanup = append(suite.cleanup, func() { suite.EventBus.Close() })

	// Setup mock services
	suite.WebhookServer = testutil.NewWebhookTestServer()
	suite.cleanup = append(suite.cleanup, func() { suite.WebhookServer.Close() })

	suite.EmailClient = testutil.NewMockEmailClient()

	return suite
}

// Teardown cleans up all test resources.
func (s *ComprehensiveTestSuite) Teardown(t *testing.T) {
	t.Helper()

	s.Cancel()

	// Run cleanup functions in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
}

// AddCleanup registers a cleanup function.
func (s *ComprehensiveTestSuite) AddCleanup(fn func()) {
	s.cleanup = append(s.cleanup, fn)
}

// ResetState resets all repositories and mock services.
func (s *ComprehensiveTestSuite) ResetState() {
	s.WebhookServer.Reset()
	s.EmailClient.Reset()

	// Reset in-memory repositories by recreating them
	s.JobRepository = schedrepo.NewMemoryJobRepository()
	s.WebhookRepository = webhookrepo.NewMemoryRepository()
	s.EmailRepository = emailrepo.NewMemoryEmailRepository()
}

// WaitForCondition is a convenience wrapper for testutil.WaitForCondition.
func (s *ComprehensiveTestSuite) WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	return testutil.WaitForCondition(t, timeout, 100*time.Millisecond, condition)
}

// AssertEventually is a convenience wrapper for testutil.AssertEventually.
func (s *ComprehensiveTestSuite) AssertEventually(t *testing.T, timeout time.Duration, assertion func() error) {
	testutil.AssertEventually(t, timeout, assertion)
}

// CreateJob creates a test job in the repository.
func (s *ComprehensiveTestSuite) CreateJob(t *testing.T, taskType string, payload interface{}) *schedrepo.Job {
	t.Helper()

	job := s.Fixtures.CreateTestJob(taskType, payload)
	if err := s.JobRepository.CreateJob(s.Ctx, job); err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}
	return job
}

// CreateWebhook creates a test webhook in the repository.
func (s *ComprehensiveTestSuite) CreateWebhook(t *testing.T, events ...bus.EventType) *webhookrepo.WebhookConfig {
	t.Helper()

	webhook := s.Fixtures.CreateTestWebhook(s.WebhookServer.URL(), events...)
	if err := s.WebhookRepository.CreateWebhook(s.Ctx, webhook); err != nil {
		t.Fatalf("failed to create test webhook: %v", err)
	}
	return webhook
}

// CreateEmail creates a test email log in the repository.
func (s *ComprehensiveTestSuite) CreateEmail(t *testing.T, to []string, subject string) *emailrepo.EmailLog {
	t.Helper()

	email := s.Fixtures.CreateTestEmailLog(to, subject, "sent")
	if err := s.EmailRepository.SaveEmail(s.Ctx, email); err != nil {
		t.Fatalf("failed to create test email: %v", err)
	}
	return email
}

// PublishEvent publishes an event through the event bus.
func (s *ComprehensiveTestSuite) PublishEvent(t *testing.T, event bus.Event) {
	t.Helper()

	if err := s.EventBus.Publish(s.Ctx, event); err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}
}

// MockLogger implements a simple logger for tests.
type MockLogger struct {
	Messages []LogMessage
}

// LogMessage represents a logged message.
type LogMessage struct {
	Level   string
	Message string
	Args    []interface{}
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]LogMessage, 0),
	}
}

func (l *MockLogger) Info(msg string, args ...interface{}) {
	l.Messages = append(l.Messages, LogMessage{Level: "info", Message: msg, Args: args})
}

func (l *MockLogger) Error(msg string, args ...interface{}) {
	l.Messages = append(l.Messages, LogMessage{Level: "error", Message: msg, Args: args})
}

func (l *MockLogger) Debug(msg string, args ...interface{}) {
	l.Messages = append(l.Messages, LogMessage{Level: "debug", Message: msg, Args: args})
}

func (l *MockLogger) Warn(msg string, args ...interface{}) {
	l.Messages = append(l.Messages, LogMessage{Level: "warn", Message: msg, Args: args})
}

// Reset clears all logged messages.
func (l *MockLogger) Reset() {
	l.Messages = make([]LogMessage, 0)
}
