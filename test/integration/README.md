# Integration Tests

This directory contains comprehensive integration tests for the CodeAI system, covering all major components from Phase 1 and Phase 2.

## Test Structure

```
test/integration/
├── comprehensive_suite_test.go    # Main test suite with shared resources
├── config/
│   └── test_config.yaml          # Test configuration
├── testutil/
│   ├── helpers.go                # Test utilities and helpers
│   └── fixtures.go               # Test data fixtures
├── scheduler/
│   └── scheduler_integration_test.go   # Job scheduler tests
├── events/
│   └── event_system_integration_test.go # Event system tests
├── email/
│   └── email_integration_test.go        # Email notification tests
├── webhook/
│   └── webhook_integration_test.go      # Webhook delivery tests
├── scenarios/
│   └── e2e_scenarios_test.go            # End-to-end scenario tests
└── performance/
    └── performance_integration_test.go   # Performance/throughput tests
```

## Prerequisites

The integration tests use in-memory implementations for most services, so no external dependencies are required for basic tests.

For full integration tests with external services:

- **Redis**: Running on localhost:6379 (for Asynq job queue)
- **Temporal**: Running on localhost:7233 (for workflow engine)

## Running Tests

### Run All Integration Tests

```bash
go test -tags=integration ./test/integration/... -v
```

### Run Specific Test Suite

```bash
# Scheduler tests
go test -tags=integration ./test/integration/scheduler -v

# Event system tests
go test -tags=integration ./test/integration/events -v

# Email notification tests
go test -tags=integration ./test/integration/email -v

# Webhook tests
go test -tags=integration ./test/integration/webhook -v

# E2E scenario tests
go test -tags=integration ./test/integration/scenarios -v

# Performance tests (skipped in short mode)
go test -tags=integration ./test/integration/performance -v
```

### Run with Coverage

```bash
go test -tags=integration ./test/integration/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Performance Tests

Performance tests are skipped in short mode. To run them:

```bash
go test -tags=integration ./test/integration/performance -v -timeout 10m
```

### Run with Race Detection

```bash
go test -tags=integration ./test/integration/... -race -v
```

## Test Categories

### Scheduler Integration Tests

Tests for the job scheduling system:
- Job CRUD operations
- Job status transitions
- Job listing and filtering
- Retry mechanics
- Scheduled job retrieval
- Concurrent operations

### Event System Tests

Tests for the pub/sub event system:
- Event publishing and subscription
- Multiple subscribers
- Async event publishing
- Error handling and recovery
- Event bus shutdown

### Email Notification Tests

Tests for the email notification service:
- Email log CRUD operations
- Listing and filtering
- Status transitions
- Mock email client

### Webhook Integration Tests

Tests for the webhook delivery system:
- Webhook CRUD operations
- Event-based webhook lookup
- Failure tracking
- Delivery logging
- Retry scheduling

### End-to-End Scenario Tests

Tests for complete workflows:
- Job to webhook flow
- Failure notification flow
- Email notification scenarios
- Concurrent job processing
- Mixed success/failure handling
- Full pipeline scenarios

### Performance Tests

Throughput and latency benchmarks:
- Job creation throughput
- Event publishing throughput
- Webhook query performance
- Concurrent operations
- Benchmarks for critical operations

## Test Utilities

### Test Helpers (`testutil/helpers.go`)

- `WaitForCondition`: Poll until condition is met
- `AssertEventually`: Retry assertion until success
- `WebhookTestServer`: Mock HTTP server for webhooks
- `MockEmailClient`: Mock email client for testing
- `EventCollector`: Collect events for verification

### Fixtures (`testutil/fixtures.go`)

- `CreateTestJob`: Create test job with defaults
- `CreateScheduledJob`: Create scheduled job
- `CreateRecurringJob`: Create cron job
- `CreateTestWebhook`: Create test webhook
- `CreateTestEvent`: Create test event
- `CreateTestEmailLog`: Create email log entry

## Writing New Tests

1. Create a test file with the `//go:build integration` build tag
2. Use the appropriate test suite for your component
3. Use the fixtures for creating test data
4. Use helpers for assertions and waiting

Example:

```go
//go:build integration

package mypackage

import (
    "testing"
    "github.com/bargom/codeai/test/integration/testutil"
)

func TestMyFeature(t *testing.T) {
    fixtures := testutil.NewFixtureBuilder()

    t.Run("test case", func(t *testing.T) {
        job := fixtures.CreateTestJob("my-task", nil)
        // ... test logic
    })
}
```

## CI/CD Integration

The integration tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run Integration Tests
  run: go test -tags=integration ./test/integration/... -v -race -timeout 10m
```

## Notes

- Tests use in-memory repositories by default for speed and isolation
- Each test resets state to ensure independence
- Performance tests are skipped in short mode to speed up regular test runs
- Use `t.Parallel()` for independent tests when appropriate
- Clean up resources in defer statements
