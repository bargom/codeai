# Testing Guide

This guide covers testing strategies, patterns, and best practices for the CodeAI project.

## Table of Contents

1. [Testing Philosophy](#testing-philosophy)
2. [Unit Testing](#unit-testing)
3. [Integration Testing](#integration-testing)
4. [Benchmark Testing](#benchmark-testing)
5. [Testing Tools and Utilities](#testing-tools-and-utilities)
6. [Running Tests](#running-tests)

---

## Testing Philosophy

### Coverage Requirements

**Target: 90%+ code coverage** for production code.

| Layer | Minimum Coverage | Target Coverage |
|-------|-----------------|-----------------|
| Parser/Compiler | 95% | 100% |
| Business Logic | 90% | 95% |
| API Handlers | 85% | 90% |
| Infrastructure | 80% | 85% |

### Test Pyramid

Follow the test pyramid principle for optimal test distribution:

```
        /\
       /  \
      / E2E\        <- Few, slow, expensive
     /------\
    /  Integ \      <- Some, moderate speed
   /----------\
  /    Unit    \    <- Many, fast, cheap
 /--------------\
```

| Type | Proportion | Speed | Isolation |
|------|------------|-------|-----------|
| Unit Tests | 70% | < 10ms | Complete |
| Integration Tests | 20% | < 1s | Partial |
| E2E Tests | 10% | < 10s | None |

### What to Test

**Always Test:**
- Public API functions and methods
- Business logic and algorithms
- Error handling and edge cases
- Input validation
- State transitions
- Parser rules and AST generation

**Don't Test:**
- Third-party library internals
- Simple getters/setters with no logic
- Generated code
- Private implementation details (test through public API)

### Testing Principles

1. **Isolation**: Tests should not depend on each other
2. **Determinism**: Same input = same output, always
3. **Speed**: Unit tests should run in milliseconds
4. **Clarity**: Test names should describe the scenario and expectation
5. **Parallelism**: Use `t.Parallel()` for independent tests

---

## Unit Testing

### Test File Structure

Test files are co-located with source files using the `*_test.go` suffix:

```
internal/
├── parser/
│   ├── parser.go           # Source
│   ├── parser_test.go      # Unit tests
│   └── parser_bench_test.go # Benchmarks
├── executor/
│   ├── executor.go
│   └── executor_test.go
```

### Table-Driven Tests

Table-driven tests are the standard pattern in this codebase. They provide:
- Clear test case documentation
- Easy addition of new cases
- Consistent structure
- Parallel execution support

**Example: Testing a Parser Function**

```go
package parser

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseVarDecl(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        input    string
        wantName string
        wantType string
        wantErr  bool
        errMsg   string
    }{
        {
            name:     "string variable",
            input:    `var greeting = "hello"`,
            wantName: "greeting",
            wantType: "string",
            wantErr:  false,
        },
        {
            name:     "integer variable",
            input:    `var count = 42`,
            wantName: "count",
            wantType: "int",
            wantErr:  false,
        },
        {
            name:     "boolean variable",
            input:    `var enabled = true`,
            wantName: "enabled",
            wantType: "bool",
            wantErr:  false,
        },
        {
            name:    "missing variable name",
            input:   `var = "test"`,
            wantErr: true,
            errMsg:  "expected identifier",
        },
        {
            name:    "missing value",
            input:   `var x = `,
            wantErr: true,
            errMsg:  "expected expression",
        },
        {
            name:    "empty input",
            input:   ``,
            wantErr: true,
            errMsg:  "unexpected end of input",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            result, err := ParseVarDecl(tt.input)

            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.wantName, result.Name)
            assert.Equal(t, tt.wantType, result.Type)
        })
    }
}
```

### Mocking Dependencies

Use interface-based mocking for isolating units under test. The codebase uses custom mock structs with function fields for flexibility.

**Example: Testing a Database Repository**

First, define an interface for the dependency:

```go
// internal/repository/interfaces.go
package repository

import (
    "context"
    "database/sql"
)

// DB defines the database operations interface
type DB interface {
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}
```

Create a mock implementation:

```go
// internal/repository/repository_test.go
package repository

import (
    "context"
    "database/sql"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// MockDB implements the DB interface for testing
type MockDB struct {
    QueryFunc    func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowFunc func(ctx context.Context, query string, args ...interface{}) *sql.Row
    ExecFunc     func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    if m.QueryFunc != nil {
        return m.QueryFunc(ctx, query, args...)
    }
    return nil, nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
    if m.QueryRowFunc != nil {
        return m.QueryRowFunc(ctx, query, args...)
    }
    return nil
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    if m.ExecFunc != nil {
        return m.ExecFunc(ctx, query, args...)
    }
    return nil, nil
}

// MockResult implements sql.Result for testing
type MockResult struct {
    LastID       int64
    RowsAffected int64
}

func (m MockResult) LastInsertId() (int64, error) { return m.LastID, nil }
func (m MockResult) RowsAffected() (int64, error) { return m.RowsAffected, nil }

func TestConfigRepository_Create(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        config    *Config
        mockExec  func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
        wantID    int64
        wantErr   bool
    }{
        {
            name: "successful creation",
            config: &Config{
                Name:        "test-config",
                Version:     "1.0.0",
                Environment: "production",
            },
            mockExec: func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
                return MockResult{LastID: 123, RowsAffected: 1}, nil
            },
            wantID:  123,
            wantErr: false,
        },
        {
            name: "database error",
            config: &Config{
                Name: "test-config",
            },
            mockExec: func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
                return nil, sql.ErrConnDone
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            mockDB := &MockDB{ExecFunc: tt.mockExec}
            repo := NewConfigRepository(mockDB)

            id, err := repo.Create(context.Background(), tt.config)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.wantID, id)
        })
    }
}
```

### Testing Error Conditions

Always test both success and error paths:

```go
func TestWorkflowExecutor_Execute(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name        string
        workflow    *Workflow
        setupMocks  func(*MockExecutor)
        wantErr     bool
        wantErrType error
    }{
        {
            name: "successful execution",
            workflow: &Workflow{
                ID:    "wf-123",
                Steps: []Step{{ID: "step-1", Action: "http.get"}},
            },
            setupMocks: func(m *MockExecutor) {
                m.ExecuteStepFunc = func(ctx context.Context, step Step) (Result, error) {
                    return Result{Status: "success"}, nil
                }
            },
            wantErr: false,
        },
        {
            name: "step execution failure",
            workflow: &Workflow{
                ID:    "wf-456",
                Steps: []Step{{ID: "step-1", Action: "http.get"}},
            },
            setupMocks: func(m *MockExecutor) {
                m.ExecuteStepFunc = func(ctx context.Context, step Step) (Result, error) {
                    return Result{}, ErrStepFailed
                }
            },
            wantErr:     true,
            wantErrType: ErrStepFailed,
        },
        {
            name: "context cancellation",
            workflow: &Workflow{
                ID:    "wf-789",
                Steps: []Step{{ID: "step-1", Action: "http.get"}},
            },
            setupMocks: func(m *MockExecutor) {
                m.ExecuteStepFunc = func(ctx context.Context, step Step) (Result, error) {
                    return Result{}, context.Canceled
                }
            },
            wantErr:     true,
            wantErrType: context.Canceled,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            mock := &MockExecutor{}
            tt.setupMocks(mock)
            executor := NewWorkflowExecutor(mock)

            result, err := executor.Execute(context.Background(), tt.workflow)

            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErrType != nil {
                    assert.ErrorIs(t, err, tt.wantErrType)
                }
                return
            }

            require.NoError(t, err)
            assert.Equal(t, "completed", result.Status)
        })
    }
}
```

---

## Integration Testing

### Build Tags

Integration tests use build tags to separate them from unit tests:

```go
//go:build integration

package integration

import (
    "testing"
)

func TestDatabaseIntegration(t *testing.T) {
    // Integration test code
}
```

### Test Database Setup

Use the testing helpers for database setup:

```go
//go:build integration

package integration

import (
    "testing"

    "github.com/stretchr/testify/require"

    dbtest "github.com/codeai/internal/database/testing"
)

func TestUserRepository_CRUD(t *testing.T) {
    // Setup test database (in-memory SQLite)
    db := dbtest.SetupTestDB(t)
    defer dbtest.TeardownTestDB(t, db)

    // Seed initial data if needed
    dbtest.SeedTestData(t, db)

    // Run repository tests
    repo := repository.NewUserRepository(db)

    // Create
    user := &User{Name: "Test User", Email: "test@example.com"}
    id, err := repo.Create(context.Background(), user)
    require.NoError(t, err)
    require.NotZero(t, id)

    // Read
    found, err := repo.GetByID(context.Background(), id)
    require.NoError(t, err)
    assert.Equal(t, user.Name, found.Name)

    // Update
    user.Name = "Updated User"
    err = repo.Update(context.Background(), user)
    require.NoError(t, err)

    // Delete
    err = repo.Delete(context.Background(), id)
    require.NoError(t, err)

    // Verify deletion
    _, err = repo.GetByID(context.Background(), id)
    assert.ErrorIs(t, err, repository.ErrNotFound)
}
```

### HTTP Endpoint Testing

Use `httptest` for API testing:

```go
//go:build integration

package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAPIEndpoints(t *testing.T) {
    // Setup test suite
    suite := SetupTestSuite(t)
    defer suite.Teardown(t)

    t.Run("POST /api/workflows creates workflow", func(t *testing.T) {
        payload := map[string]interface{}{
            "name":        "test-workflow",
            "description": "A test workflow",
            "steps": []map[string]interface{}{
                {"id": "step-1", "action": "http.get", "url": "https://api.example.com"},
            },
        }

        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        rec := httptest.NewRecorder()
        suite.Handler.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusCreated, rec.Code)

        var response map[string]interface{}
        err := json.Unmarshal(rec.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.NotEmpty(t, response["id"])
        assert.Equal(t, "test-workflow", response["name"])
    })

    t.Run("GET /api/workflows/:id returns workflow", func(t *testing.T) {
        // First create a workflow
        wf := suite.CreateWorkflow(t, "fetch-workflow")

        req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wf.ID, nil)
        rec := httptest.NewRecorder()
        suite.Handler.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusOK, rec.Code)

        var response Workflow
        err := json.Unmarshal(rec.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.Equal(t, wf.ID, response.ID)
    })

    t.Run("GET /api/workflows/:id returns 404 for unknown ID", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/workflows/nonexistent", nil)
        rec := httptest.NewRecorder()
        suite.Handler.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusNotFound, rec.Code)
    })
}
```

### Workflow Testing with Temporal Test Server

For Temporal workflows, use the test environment:

```go
//go:build integration

package workflow

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.temporal.io/sdk/testsuite"
)

func TestDeploymentWorkflow(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Register workflow and activities
    env.RegisterWorkflow(DeploymentWorkflow)
    env.RegisterActivity(ValidateConfigActivity)
    env.RegisterActivity(DeployActivity)
    env.RegisterActivity(HealthCheckActivity)

    // Mock activities
    env.OnActivity(ValidateConfigActivity, mock.Anything, mock.Anything).Return(nil)
    env.OnActivity(DeployActivity, mock.Anything, mock.Anything).Return(&DeployResult{
        Version: "1.0.0",
        Status:  "deployed",
    }, nil)
    env.OnActivity(HealthCheckActivity, mock.Anything, mock.Anything).Return(nil)

    // Execute workflow
    input := DeploymentInput{
        ServiceName: "api-server",
        Version:     "1.0.0",
        Environment: "staging",
    }

    env.ExecuteWorkflow(DeploymentWorkflow, input)

    require.True(t, env.IsWorkflowCompleted())
    require.NoError(t, env.GetWorkflowError())

    var result DeploymentResult
    require.NoError(t, env.GetWorkflowResult(&result))
    assert.Equal(t, "deployed", result.Status)
}

func TestDeploymentWorkflow_Rollback(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    env.RegisterWorkflow(DeploymentWorkflow)
    env.RegisterActivity(ValidateConfigActivity)
    env.RegisterActivity(DeployActivity)
    env.RegisterActivity(RollbackActivity)

    // Mock deploy to fail
    env.OnActivity(DeployActivity, mock.Anything, mock.Anything).Return(nil, errors.New("deployment failed"))
    env.OnActivity(RollbackActivity, mock.Anything, mock.Anything).Return(nil)

    input := DeploymentInput{
        ServiceName: "api-server",
        Version:     "1.0.0",
    }

    env.ExecuteWorkflow(DeploymentWorkflow, input)

    require.True(t, env.IsWorkflowCompleted())
    // Workflow should complete but indicate failure
    assert.Error(t, env.GetWorkflowError())
}
```

### Container-Based Testing

Use testcontainers for external dependencies:

```go
//go:build integration

package cache

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisContainer(t *testing.T) (*RedisCache, func()) {
    ctx := context.Background()

    redisContainer, err := redis.Run(ctx, "redis:7-alpine")
    require.NoError(t, err)

    endpoint, err := redisContainer.Endpoint(ctx, "")
    require.NoError(t, err)

    cache, err := NewRedisCache(endpoint)
    require.NoError(t, err)

    cleanup := func() {
        cache.Close()
        _ = redisContainer.Terminate(ctx)
    }

    return cache, cleanup
}

func TestRedisCache_SetGet(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    cache, cleanup := setupRedisContainer(t)
    defer cleanup()

    ctx := context.Background()

    // Test Set
    err := cache.Set(ctx, "key1", "value1", 1*time.Minute)
    require.NoError(t, err)

    // Test Get
    value, err := cache.Get(ctx, "key1")
    require.NoError(t, err)
    assert.Equal(t, "value1", value)

    // Test Get non-existent
    _, err = cache.Get(ctx, "nonexistent")
    assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestRedisCache_Expiration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    cache, cleanup := setupRedisContainer(t)
    defer cleanup()

    ctx := context.Background()

    // Set with short TTL
    err := cache.Set(ctx, "expiring-key", "value", 100*time.Millisecond)
    require.NoError(t, err)

    // Verify it exists
    _, err = cache.Get(ctx, "expiring-key")
    require.NoError(t, err)

    // Wait for expiration
    time.Sleep(200 * time.Millisecond)

    // Verify it's gone
    _, err = cache.Get(ctx, "expiring-key")
    assert.ErrorIs(t, err, ErrKeyNotFound)
}
```

### End-to-End API Test Example

```go
//go:build integration

package e2e

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    apitest "github.com/codeai/internal/api/testing"
    "github.com/codeai/test/integration/testutil"
)

func TestWorkflowExecutionE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    // Setup the full application stack
    suite := SetupTestSuite(t)
    defer suite.Teardown(t)

    // Setup webhook receiver
    webhookServer := testutil.NewWebhookTestServer(t)
    defer webhookServer.Close()

    ctx := context.Background()

    // 1. Create a workflow
    createReq := apitest.NewRequest(t, http.MethodPost, "/api/workflows", map[string]interface{}{
        "name": "e2e-test-workflow",
        "steps": []map[string]interface{}{
            {
                "id":     "fetch",
                "action": "http.get",
                "url":    "https://api.example.com/data",
            },
            {
                "id":       "notify",
                "action":   "webhook",
                "url":      webhookServer.URL,
                "depends":  []string{"fetch"},
            },
        },
        "on_complete": map[string]interface{}{
            "webhook": webhookServer.URL + "/complete",
        },
    })

    resp := suite.MakeRequest(createReq)
    require.Equal(t, http.StatusCreated, resp.StatusCode)

    var workflow map[string]interface{}
    apitest.DecodeJSON(t, resp, &workflow)
    workflowID := workflow["id"].(string)

    // 2. Trigger execution
    triggerReq := apitest.NewRequest(t, http.MethodPost, "/api/workflows/"+workflowID+"/execute", nil)
    resp = suite.MakeRequest(triggerReq)
    require.Equal(t, http.StatusAccepted, resp.StatusCode)

    var execution map[string]interface{}
    apitest.DecodeJSON(t, resp, &execution)
    executionID := execution["id"].(string)

    // 3. Wait for completion
    err := testutil.WaitForCondition(ctx, 30*time.Second, func() bool {
        statusReq := apitest.NewRequest(t, http.MethodGet, "/api/executions/"+executionID, nil)
        resp := suite.MakeRequest(statusReq)
        if resp.StatusCode != http.StatusOK {
            return false
        }
        var status map[string]interface{}
        apitest.DecodeJSON(t, resp, &status)
        return status["status"] == "completed"
    })
    require.NoError(t, err, "workflow execution did not complete in time")

    // 4. Verify webhook was called
    assert.True(t, webhookServer.ReceivedRequest("/complete"), "completion webhook was not called")

    // 5. Verify execution details
    detailsReq := apitest.NewRequest(t, http.MethodGet, "/api/executions/"+executionID, nil)
    resp = suite.MakeRequest(detailsReq)
    require.Equal(t, http.StatusOK, resp.StatusCode)

    var details map[string]interface{}
    apitest.DecodeJSON(t, resp, &details)
    assert.Equal(t, "completed", details["status"])
    assert.NotEmpty(t, details["completed_at"])
}
```

---

## Benchmark Testing

### Writing Benchmark Tests

Benchmark tests measure performance and help detect regressions:

```go
package parser

import (
    "testing"
)

// Small program benchmark
func BenchmarkParseSmallProgram(b *testing.B) {
    program := `
        workflow "test" {
            step "fetch" {
                action = "http.get"
                url = "https://api.example.com"
            }
        }
    `

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := Parse(program)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Medium program benchmark
func BenchmarkParseMediumProgram(b *testing.B) {
    program := generateProgram(100) // 100 statements

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := Parse(program)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Large program benchmark
func BenchmarkParseLargeProgram(b *testing.B) {
    program := generateProgram(1000) // 1000 statements

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := Parse(program)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Benchmark with sub-benchmarks
func BenchmarkParse(b *testing.B) {
    sizes := []struct {
        name  string
        count int
    }{
        {"Small", 10},
        {"Medium", 100},
        {"Large", 1000},
    }

    for _, size := range sizes {
        program := generateProgram(size.count)

        b.Run(size.name, func(b *testing.B) {
            b.ReportAllocs()
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                _, _ = Parse(program)
            }
        })
    }
}

// Helper to generate test programs
func generateProgram(stepCount int) string {
    var b strings.Builder
    b.WriteString(`workflow "benchmark" {` + "\n")

    for i := 0; i < stepCount; i++ {
        fmt.Fprintf(&b, `
    step "step_%d" {
        action = "http.get"
        url = "https://api.example.com/endpoint/%d"
        headers = {
            "Authorization" = "Bearer token"
            "Content-Type" = "application/json"
        }
    }
`, i, i)
    }

    b.WriteString("}\n")
    return b.String()
}
```

### Memory Benchmarks

```go
func BenchmarkExecutorMemory(b *testing.B) {
    workflow := createLargeWorkflow()

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        executor := NewExecutor()
        _, _ = executor.Execute(context.Background(), workflow)
    }
}
```

### Interpreting Results

Run benchmarks with:
```bash
go test -bench=. -benchmem ./internal/parser/
```

Output explanation:
```
BenchmarkParseSmallProgram-8    500000    2340 ns/op    1024 B/op    12 allocs/op
│                          │    │         │             │            │
│                          │    │         │             │            └─ Allocations per op
│                          │    │         │             └─ Bytes allocated per op
│                          │    │         └─ Nanoseconds per operation
│                          │    └─ Number of iterations
│                          └─ GOMAXPROCS
└─ Benchmark name
```

### Performance Regression Detection

Compare benchmarks between versions:

```bash
# Save baseline
go test -bench=. -benchmem ./... > baseline.txt

# After changes
go test -bench=. -benchmem ./... > current.txt

# Compare
benchstat baseline.txt current.txt
```

---

## Testing Tools and Utilities

### testify/assert and testify/require

Use `require` for setup assertions (stops test on failure):
```go
require.NoError(t, err)          // Stop if error
require.NotNil(t, obj)           // Stop if nil
require.True(t, condition)       // Stop if false
```

Use `assert` for test assertions (continues on failure):
```go
assert.Equal(t, expected, actual)
assert.Contains(t, str, substr)
assert.ErrorIs(t, err, targetErr)
assert.JSONEq(t, expectedJSON, actualJSON)
```

### Test Suites with testify/suite

For complex test setups with shared state:

```go
package repository

import (
    "testing"

    "github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
    suite.Suite
    db   *sql.DB
    repo *UserRepository
}

func (s *RepositoryTestSuite) SetupSuite() {
    // Run once before all tests
    s.db = setupTestDatabase(s.T())
}

func (s *RepositoryTestSuite) TearDownSuite() {
    // Run once after all tests
    s.db.Close()
}

func (s *RepositoryTestSuite) SetupTest() {
    // Run before each test
    s.repo = NewUserRepository(s.db)
    truncateTables(s.db)
}

func (s *RepositoryTestSuite) TearDownTest() {
    // Run after each test (cleanup)
}

func (s *RepositoryTestSuite) TestCreate() {
    user := &User{Name: "Test", Email: "test@example.com"}

    id, err := s.repo.Create(context.Background(), user)

    s.Require().NoError(err)
    s.Assert().NotZero(id)
}

func (s *RepositoryTestSuite) TestGetByID() {
    // Create first
    user := &User{Name: "Test", Email: "test@example.com"}
    id, _ := s.repo.Create(context.Background(), user)

    // Fetch
    found, err := s.repo.GetByID(context.Background(), id)

    s.Require().NoError(err)
    s.Assert().Equal(user.Name, found.Name)
}

func TestRepositorySuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}
```

### httptest for HTTP Handlers

```go
func TestHandler(t *testing.T) {
    handler := NewHandler(mockService)

    // Create test server
    server := httptest.NewServer(handler)
    defer server.Close()

    // Make request
    resp, err := http.Get(server.URL + "/api/resource")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### Database Testing Helpers

Located in `internal/database/testing/helpers.go`:

```go
// SetupTestDB creates an in-memory SQLite database with migrations
func SetupTestDB(t *testing.T) *sql.DB

// TeardownTestDB closes the database connection
func TeardownTestDB(t *testing.T, db *sql.DB)

// SeedTestData populates the database with fixture data
func SeedTestData(t *testing.T, db *sql.DB)
```

### API Testing Helpers

Located in `internal/api/testing/helpers.go`:

```go
// TestServer wraps httptest.Server with helpers
type TestServer struct {
    *httptest.Server
    t *testing.T
}

// MakeRequest executes an HTTP request and returns the response
func (s *TestServer) MakeRequest(req *http.Request) *http.Response

// AssertStatus verifies response status code
func AssertStatus(t *testing.T, resp *http.Response, expected int)

// AssertJSON verifies JSON response body
func AssertJSON(t *testing.T, resp *http.Response, expected interface{})

// DecodeJSON decodes response body into target
func DecodeJSON(t *testing.T, resp *http.Response, target interface{})
```

### Integration Test Utilities

Located in `test/integration/testutil/helpers.go`:

```go
// WaitForCondition polls until condition returns true or timeout
func WaitForCondition(ctx context.Context, timeout time.Duration, condition func() bool) error

// AssertEventually retries assertion until success or timeout
func AssertEventually(t *testing.T, timeout time.Duration, assertion func() bool, msg string)

// WebhookTestServer captures incoming webhook requests
type WebhookTestServer struct {
    *httptest.Server
    Requests []WebhookRequest
}

// NewWebhookTestServer creates a test server for capturing webhooks
func NewWebhookTestServer(t *testing.T) *WebhookTestServer
```

---

## Running Tests

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/parser/

# Run specific test
go test -run TestParseVarDecl ./internal/parser/

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./test/integration/...

# Run with verbose output
go test -v -tags=integration ./test/integration/...

# Skip in short mode
go test -short ./...  # Skips tests with testing.Short() check
```

### Benchmark Tests

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkParse ./internal/parser/

# Run with memory profiling
go test -bench=. -benchmem ./...

# Run for specific duration
go test -bench=. -benchtime=5s ./...

# Run multiple times for stability
go test -bench=. -count=5 ./...
```

### Coverage Reports

```bash
# Generate coverage for all packages
go test -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Check coverage threshold (CI)
go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 90" | bc -l) )); then
    echo "Coverage $COVERAGE% is below 90% threshold"
    exit 1
fi
```

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run Unit Tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Run Integration Tests
        run: go test -v -tags=integration ./test/integration/...
        env:
          REDIS_URL: localhost:6379

      - name: Check Coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 90" | bc -l) )); then
            echo "::error::Coverage $COVERAGE% is below 90% threshold"
            exit 1
          fi

      - name: Upload Coverage
        uses: codecov/codecov-action@v4
        with:
          file: coverage.out

  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run Benchmarks
        run: go test -bench=. -benchmem ./... | tee benchmark.txt

      - name: Store Benchmark Results
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: benchmark.txt
          fail-on-alert: true
```

### Makefile Targets

```makefile
.PHONY: test test-unit test-integration test-bench coverage

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	go test -v -race ./...

# Run integration tests
test-integration:
	go test -v -tags=integration ./test/integration/...

# Run benchmarks
test-bench:
	go test -bench=. -benchmem ./...

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Check coverage threshold
coverage-check: coverage
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 90" | bc -l) -eq 1 ]; then \
		echo "Error: Coverage below 90% threshold"; \
		exit 1; \
	fi
```

---

## Quick Reference

### Test Naming Conventions

| Pattern | Example | Use Case |
|---------|---------|----------|
| `Test<Function>` | `TestParse` | Basic function test |
| `Test<Function>_<Scenario>` | `TestParse_InvalidInput` | Specific scenario |
| `Test<Type>_<Method>` | `TestRepository_Create` | Method test |
| `Benchmark<Operation>` | `BenchmarkParse` | Performance test |
| `Example<Function>` | `ExampleParse` | Documentation example |

### Assertion Cheat Sheet

```go
// Equality
assert.Equal(t, expected, actual)
assert.NotEqual(t, unexpected, actual)

// Nil checks
assert.Nil(t, value)
assert.NotNil(t, value)

// Boolean
assert.True(t, condition)
assert.False(t, condition)

// Errors
assert.NoError(t, err)
assert.Error(t, err)
assert.ErrorIs(t, err, targetErr)
assert.ErrorContains(t, err, "message")

// Collections
assert.Contains(t, collection, element)
assert.Len(t, collection, expectedLen)
assert.Empty(t, collection)

// Strings
assert.Contains(t, str, substr)
assert.HasPrefix(t, str, prefix)
assert.HasSuffix(t, str, suffix)

// JSON
assert.JSONEq(t, expectedJSON, actualJSON)
```

### Common Test Patterns

```go
// Setup/Teardown
func TestWithSetup(t *testing.T) {
    // Setup
    resource := setupResource(t)
    t.Cleanup(func() { resource.Close() })

    // Test
    result := resource.DoSomething()
    assert.NotEmpty(t, result)
}

// Parallel tests
func TestParallel(t *testing.T) {
    t.Parallel()
    tests := []struct{ name string }{{name: "case1"}, {name: "case2"}}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test code
        })
    }
}

// Skip conditions
func TestConditional(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping in short mode")
    }
    // long-running test
}
```
