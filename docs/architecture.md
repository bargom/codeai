# CodeAI Architecture Guide

This document provides comprehensive architecture documentation for the CodeAI platform, covering all modules, design patterns, and integration points.

## Table of Contents

1. [System Overview](#system-overview)
2. [Module Reference](#module-reference)
   - [Parser](#parser-module)
   - [Database](#database-module)
   - [HTTP/API](#http-api-module)
   - [Auth](#auth-module)
   - [Workflow](#workflow-module)
   - [Job/Scheduler](#job-scheduler-module)
   - [Event](#event-module)
   - [Integration](#integration-module)
   - [Notification](#notification-module)
   - [Webhook](#webhook-module)
   - [Logging](#logging-module)
   - [Metrics](#metrics-module)
3. [Design Patterns](#design-patterns)
4. [Testing Strategy](#testing-strategy)
5. [Configuration](#configuration)

---

## System Overview

CodeAI is a Go-based platform for AI-powered code operations with a custom DSL for defining deployments, workflows, and configurations.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              HTTP Layer                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Chi Router + Middleware                           │    │
│  │  (RequestID, RealIP, Logger, Recoverer, Timeout, JSON Content-Type) │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌───────────┐   ┌───────────┐   ┌───────────┐
            │   Auth    │   │  Handlers │   │  OpenAPI  │
            │Middleware │   │  (CRUD)   │   │   Docs    │
            └─────┬─────┘   └─────┬─────┘   └───────────┘
                  │               │
                  ▼               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Business Logic Layer                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │  Parser  │  │ Workflow │  │Scheduler │  │  Query   │  │Validator │      │
│  │  (DSL)   │  │(Temporal)│  │ (Asynq)  │  │ Builder  │  │          │      │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Infrastructure Layer                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ Database │  │  Event   │  │  Cache   │  │Integration│  │ Webhook  │      │
│  │(Postgres)│  │   Bus    │  │ (Redis)  │  │ (REST/GQL)│  │ Delivery │      │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Cross-Cutting Concerns                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ Logging  │  │ Metrics  │  │ Tracing  │  │  Health  │  │ Shutdown │      │
│  │ (slog)   │  │(Promethe)│  │(OpenTele)│  │  Checks  │  │ Graceful │      │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │────▶│  Router  │────▶│  Auth    │────▶│ Handler  │
│ Request  │     │          │     │Middleware│     │          │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
                                                         │
                    ┌────────────────────────────────────┘
                    ▼
              ┌──────────┐
              │ Business │
              │  Logic   │
              └──────────┘
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Database │  │  Event   │  │  Cache   │
│          │  │   Bus    │  │          │
└──────────┘  └──────────┘  └──────────┘
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Webhook  │  │  Email   │  │ External │
│ Delivery │  │  Notif   │  │   APIs   │
└──────────┘  └──────────┘  └──────────┘
```

### Directory Structure

```
codeai/
├── cmd/codeai/           # Application entry point
├── internal/             # Private application code
│   ├── api/              # HTTP handlers, router, server
│   ├── ast/              # Abstract Syntax Tree definitions
│   ├── auth/             # JWT validation, middleware
│   ├── cache/            # Caching layer
│   ├── database/         # Database connections, migrations, repositories
│   ├── engine/           # DSL execution engine
│   ├── event/            # Event bus, dispatching, subscribers
│   ├── health/           # Health check endpoints
│   ├── notification/     # Email service, templates
│   ├── openapi/          # OpenAPI specification
│   ├── pagination/       # Pagination utilities
│   ├── parser/           # DSL lexer and parser
│   ├── query/            # Query builder (lex, parse, compile, execute)
│   ├── rbac/             # Role-based access control
│   ├── scheduler/        # Asynq job scheduler
│   ├── shutdown/         # Graceful shutdown handling
│   ├── validation/       # Input validation
│   ├── validator/        # Struct validation
│   ├── webhook/          # Webhook delivery system
│   └── workflow/         # Temporal workflow orchestration
├── pkg/                  # Public libraries
│   ├── integration/      # External service clients (REST, GraphQL, circuit breakers)
│   ├── logging/          # Structured logging
│   ├── metrics/          # Prometheus metrics
│   └── types/            # Shared type definitions
├── config/               # Configuration files
├── docs/                 # Documentation
└── test/                 # Integration tests
```

---

## Module Reference

### Parser Module

**Location**: `internal/parser/`, `internal/ast/`

**Purpose**: Parse CodeAI DSL into an executable Abstract Syntax Tree.

#### Key Components

| File | Purpose |
|------|---------|
| `parser/parser.go` | Participle-based lexer and parser |
| `ast/types.go` | Position tracking, NodeType enum |
| `ast/ast.go` | 16 AST node type definitions |
| `ast/helpers.go` | Walk, Print, Equal, Clone utilities |

#### DSL Language Features

**Declarations**:
```
var x = value
function add(a, b) { return result }
```

**Statements**:
```
x = 42                           # Assignment
if condition { ... } else { ... } # Conditional
for item in items { ... }         # Loop
exec { shell command }            # Shell execution
return value                      # Return
```

**Expressions**:
```
"string"           # String literal
42 or 3.14         # Number literal
true / false       # Boolean literal
[1, 2, 3]          # Array literal
func(arg1, arg2)   # Function call
a + b              # Binary expression
-x, not flag       # Unary expression
```

**Operators**: `+`, `-`, `*`, `/`, `==`, `!=`, `<`, `>`, `<=`, `>=`, `and`, `or`

#### Usage Example

```go
import "github.com/codeai/internal/parser"
import "github.com/codeai/internal/ast"

// Parse DSL string
program, err := parser.Parse(`
    var greeting = "Hello"
    function sayHello(name) {
        return greeting + " " + name
    }
`)

// Walk the AST
ast.Walk(program, func(node ast.Node) bool {
    switch n := node.(type) {
    case *ast.VarDecl:
        fmt.Printf("Variable: %s\n", n.Name)
    case *ast.FunctionDecl:
        fmt.Printf("Function: %s\n", n.Name)
    }
    return true
})

// Pretty print
fmt.Println(ast.Print(program))
```

#### Configuration

The parser uses Participle with:
- **Lookahead**: 3 tokens for disambiguation
- **Elision**: Whitespace and comments ignored
- **Shell State**: Special lexer state for `exec { }` blocks

#### Dependencies
- `github.com/alecthomas/participle/v2`

---

### Database Module

**Location**: `internal/database/`

**Purpose**: PostgreSQL database operations with connection pooling, migrations, and repository pattern.

#### Key Components

| File | Purpose |
|------|---------|
| `database.go` | Connection management, pooling configuration |
| `migrate.go` | Migration engine with embedded SQL |
| `models/models.go` | Domain models (Deployment, Config, Execution) |
| `repository/*.go` | CRUD repositories per entity |
| `testing/helpers.go` | Test utilities with in-memory SQLite |

#### Connection Pooling

```go
type Config struct {
    Host     string
    Port     int
    Database string
    User     string
    Password string
    SSLMode  string
}

// Pool settings (applied automatically)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)
```

#### Migration System

```go
migrator := database.NewMigrator(db)

// Run all pending migrations
err := migrator.MigrateUp()

// Rollback last migration
err := migrator.MigrateDown()

// Check migration status
migrations := migrator.Status()
```

**Migration Naming**: `{VERSION}_{NAME}.{up/down}.sql`
Example: `20260111120000_initial_schema.up.sql`

#### Repository Pattern

```go
// All repositories support transactions
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

deployRepo := repository.NewDeploymentRepository(db).WithTx(tx)
configRepo := repository.NewConfigRepository(db).WithTx(tx)

// Operations in same transaction
deployRepo.Create(ctx, deployment)
configRepo.Create(ctx, config)

tx.Commit()
```

#### Domain Models

**Deployment**:
```go
type Deployment struct {
    ID        string         // UUID
    Name      string         // Unique name
    ConfigID  sql.NullString // FK to configs
    Status    string         // pending|running|stopped|failed|complete
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Config**:
```go
type Config struct {
    ID               string
    Name             string
    Content          string          // DSL content
    ASTJSON          json.RawMessage // Parsed AST
    ValidationErrors json.RawMessage
    CreatedAt        time.Time
}
```

**Execution**:
```go
type Execution struct {
    ID           string
    DeploymentID string
    Command      string
    Output       sql.NullString
    ExitCode     sql.NullInt32
    StartedAt    time.Time
    CompletedAt  sql.NullTime
}
```

#### Dependencies
- `github.com/lib/pq` (PostgreSQL)
- `github.com/google/uuid`
- `modernc.org/sqlite` (testing)

---

### HTTP/API Module

**Location**: `internal/api/`

**Purpose**: REST API with Chi router, middleware stack, and request validation.

#### Key Components

| File | Purpose |
|------|---------|
| `router.go` | Chi router configuration with middleware |
| `server.go` | HTTP server lifecycle management |
| `handlers/*.go` | Request handlers per resource |
| `types/*.go` | Request/Response DTOs |

#### Router Configuration

```go
r := chi.NewRouter()

// Global middleware
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.Timeout(60 * time.Second))

// Routes
r.Route("/deployments", func(r chi.Router) {
    r.Post("/", handler.CreateDeployment)
    r.Get("/", handler.ListDeployments)
    r.Get("/{id}", handler.GetDeployment)
    r.Put("/{id}", handler.UpdateDeployment)
    r.Delete("/{id}", handler.DeleteDeployment)
    r.Post("/{id}/execute", handler.ExecuteDeployment)
})
```

#### Request Validation

Uses `go-playground/validator/v10`:

```go
type CreateDeploymentRequest struct {
    Name     string `json:"name" validate:"required,min=1,max=255"`
    ConfigID string `json:"config_id" validate:"omitempty,uuid"`
}

// Validation tags: required, min, max, uuid, oneof, omitempty
```

#### Handler Pattern

```go
func (h *Handler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
    var req types.CreateDeploymentRequest

    // Decode and validate
    if err := h.decodeAndValidate(r, &req); err != nil {
        h.respondValidationError(w, err)
        return
    }

    // Create entity
    deployment := models.NewDeployment(req.Name)
    if err := h.deployments.Create(r.Context(), deployment); err != nil {
        h.respondError(w, http.StatusInternalServerError, "failed to create")
        return
    }

    h.respondJSON(w, http.StatusCreated, types.DeploymentFromModel(deployment))
}
```

#### Query Builder

Full DSL for database queries:

```
# SELECT queries
SELECT id, name FROM users WHERE status = active ORDER BY created_at DESC

# Aggregate queries
COUNT FROM users WHERE active = true
SUM(amount) FROM orders WHERE status = completed

# UPDATE queries
UPDATE users SET status = inactive WHERE last_login < @date

# Simple filter syntax
status:active priority:1 name:~john
```

**Operators**: `=`, `!=`, `>`, `>=`, `<`, `<=`, `LIKE`, `ILIKE`, `CONTAINS`, `STARTSWITH`, `ENDSWITH`, `IN`, `IS NULL`, `IS NOT NULL`, `BETWEEN`

#### Dependencies
- `github.com/go-chi/chi/v5`
- `github.com/go-playground/validator/v10`

---

### Auth Module

**Location**: `internal/auth/`, `internal/rbac/`

**Purpose**: JWT validation, RBAC, and permission checking.

#### Key Components

| File | Purpose |
|------|---------|
| `auth/jwt.go` | JWT token validation |
| `auth/jwks.go` | JWKS caching and key rotation |
| `auth/middleware.go` | HTTP authentication middleware |
| `rbac/rbac.go` | RBAC engine with permission resolution |
| `rbac/policy.go` | Role and permission definitions |
| `rbac/middleware.go` | HTTP authorization middleware |

#### JWT Configuration

```go
type Config struct {
    Issuer     string // Expected issuer (iss claim)
    Audience   string // Expected audience (aud claim)
    Secret     string // HS256/384/512 symmetric key
    PublicKey  string // PEM-encoded RSA public key
    JWKSURL    string // URL to fetch JWKS from
    RolesClaim string // Claim name for roles (default: "roles")
    PermsClaim string // Claim name for permissions
}
```

**Supported Algorithms**: HS256, HS384, HS512, RS256, RS384, RS512

#### Authentication Middleware

```go
authMW := auth.NewMiddleware(validator)

// Require authentication
r.With(authMW.RequireAuth()).Get("/protected", handler)

// Optional authentication
r.With(authMW.OptionalAuth()).Get("/public", handler)

// Require specific role
r.With(authMW.RequireRole("admin")).Get("/admin", handler)
```

#### RBAC Permission Model

**Permission Format**: `"resource:action"` (e.g., `"configs:read"`)

**Wildcard Support**:
- `"*:read"` - Any resource, read action
- `"users:*"` - Users resource, any action
- `"*:*"` - Full access

**Default Roles**:
```go
admin  → "*:*" (full access)
editor → configs:*, executions:*, deployments:* (inherits viewer)
viewer → configs:read, executions:read, deployments:read, health:read
```

#### Authorization Middleware

```go
rbacMW := rbac.NewMiddleware(engine)

// Single permission
r.With(rbacMW.RequirePermission("configs:read")).Get("/configs", handler)

// Multiple permissions (any)
r.With(rbacMW.RequireAnyPermission("configs:read", "configs:write")).Get("/configs", handler)

// Multiple permissions (all)
r.With(rbacMW.RequireAllPermissions("configs:read", "audit:read")).Get("/configs", handler)
```

#### User Context

```go
// In middleware - add user to context
ctx := auth.ContextWithUser(r.Context(), user)

// In handler - retrieve user
user := auth.UserFromContext(r.Context())
if user.HasRole("admin") {
    // Admin logic
}
```

#### Dependencies
- `github.com/golang-jwt/jwt/v5`

---

### Workflow Module

**Location**: `internal/workflow/`

**Purpose**: Temporal-based workflow orchestration with saga patterns for distributed transactions.

#### Key Components

| Directory | Purpose |
|-----------|---------|
| `engine/` | Temporal client management |
| `patterns/` | Saga workflow implementation |
| `compensation/` | Compensation manager and repository |
| `activities/` | Activity implementations |
| `definitions/` | Workflow type definitions |

#### Workflow Engine

```go
engine := workflow.NewEngine(workflow.DefaultConfig())
engine.Start(ctx)
defer engine.Stop()

// Execute workflow
result, err := engine.ExecuteWorkflow(ctx, "AIAgentPipeline", input)
```

**Default Configuration**:
```go
TemporalHostPort: "localhost:7233"
Namespace: "default"
TaskQueue: "codeai-workflows"
MaxConcurrentWorkflows: 100
MaxConcurrentActivities: 100
DefaultTimeout: 30 * time.Minute
```

#### Saga Pattern

```go
saga := patterns.NewSagaBuilder("data-pipeline").
    AddStep("extract", ExtractActivity, nil, RevertExtract).
    AddStep("transform", TransformActivity, nil, RevertTransform).
    AddStep("load", LoadActivity, nil, RevertLoad).
    Build()

output, err := saga.Execute(ctx)
```

**Saga Features**:
- Automatic compensation on failure (LIFO order)
- Signal support for manual compensation/cancellation
- Non-critical steps that continue on failure
- Full audit trail with compensation logs

#### Workflow Types

**AIAgentPipelineWorkflow**:
```go
Input{
    WorkflowID: "pipeline-123",
    Agents: []AgentConfig{...},
    Timeout: 30 * time.Minute,
    Parallel: false,
}

Output{
    Status: "completed",
    Results: []AgentResult{...},
    TotalDuration: 5 * time.Minute,
    CompensationLogs: []string{...},
}
```

**TestSuiteWorkflow**:
```go
Input{
    SuiteID: "suite-123",
    TestCases: []TestCase{...},
    StopOnFailure: true,
    Parallel: true,
}

Output{
    Results: []TestResult{...},
    TotalCount: 10,
    PassedCount: 8,
    FailedCount: 2,
}
```

#### Dependencies
- `go.temporal.io/sdk`

---

### Job/Scheduler Module

**Location**: `internal/scheduler/`

**Purpose**: Asynq-based job scheduling with Redis backend for async task processing.

#### Key Components

| Directory | Purpose |
|-----------|---------|
| `queue/` | Asynq client/server management |
| `service/` | High-level job service |
| `repository/` | Job persistence |
| `handlers/` | Task handler registry |
| `tasks/` | Task type implementations |

#### Queue Manager

```go
manager := queue.NewManager(queue.Config{
    QueueSize:    1000,
    WorkerCount:  10,
    DrainTimeout: 30 * time.Second,
})

// Priority queues
// critical: 6 workers (webhooks, high-priority)
// default:  3 workers (general tasks)
// low:      1 worker  (cleanup, batch jobs)
```

#### Job Service

```go
service := scheduler.NewService(manager, repository)

// Immediate execution
jobID, err := service.SubmitJob(ctx, "ai_agent_execution", payload)

// Scheduled execution
jobID, err := service.ScheduleJob(ctx, "cleanup", payload, time.Now().Add(24*time.Hour))

// Recurring job (cron)
jobID, err := service.CreateRecurringJob(ctx, "cleanup", payload, "0 2 * * *") // 2 AM daily
```

#### Task Types

| Type | Queue | Timeout | Retries |
|------|-------|---------|---------|
| `ai_agent_execution` | default | 5 min | 3 |
| `test_suite_run` | default | 30 min | 2 |
| `data_processing` | low | 15 min | 3 |
| `webhook` | critical | 30 sec | 5 |
| `cleanup` | low | 10 min | 1 |

#### Task Handler

```go
registry := handlers.NewRegistry()
registry.RegisterHandler("ai_agent_execution", HandleAIAgentTask)
registry.RegisterHandler("webhook", HandleWebhookTask)

// With middleware
registry.WithMiddleware(
    middleware.Logging,
    middleware.Recovery,
    middleware.Timeout,
)
```

#### Dependencies
- `github.com/hibiken/asynq`
- Redis

---

### Event Module

**Location**: `internal/event/`

**Purpose**: In-memory pub/sub event bus with optional persistence.

#### Key Components

| Directory | Purpose |
|-----------|---------|
| `bus/` | Core EventBus implementation |
| `dispatcher/` | Event dispatcher with persistence |
| `repository/` | Event persistence |
| `subscribers/` | Built-in subscribers (webhook, metrics, logging) |
| `handlers/` | Domain event handlers |

#### Event Bus

```go
bus := event.NewEventBus(event.Config{
    WorkerCount: 10,
    BufferSize:  1000,
})
defer bus.Close()

// Subscribe
bus.Subscribe("workflow.completed", func(ctx context.Context, e event.Event) error {
    // Handle event
    return nil
})

// Publish (sync)
bus.Publish(ctx, event)

// Publish (async)
bus.PublishAsync(ctx, event)
```

#### Event Types

```go
// Workflow events
EventWorkflowStarted   = "workflow.started"
EventWorkflowCompleted = "workflow.completed"
EventWorkflowFailed    = "workflow.failed"

// Job events
EventJobEnqueued  = "job.enqueued"
EventJobStarted   = "job.started"
EventJobCompleted = "job.completed"
EventJobFailed    = "job.failed"

// System events
EventAgentExecuted      = "agent.executed"
EventTestSuiteCompleted = "test.suite.completed"
EventWebhookTriggered   = "webhook.triggered"
EventEmailSent          = "email.sent"
```

#### Event Structure

```go
type Event struct {
    ID        string                 // Unique identifier
    Type      EventType              // Event classification
    Source    string                 // System origin
    Timestamp time.Time              // Event time
    Data      map[string]interface{} // Event payload
    Metadata  map[string]string      // Additional context
}
```

#### Built-in Subscribers

- **WebhookSubscriber**: Forward events to external HTTP endpoints
- **MetricsSubscriber**: Track event statistics
- **LoggingSubscriber**: Log all events
- **EmailEventSubscriber**: Queue email notifications

---

### Integration Module

**Location**: `pkg/integration/`

**Purpose**: External service integration with circuit breakers, retry logic, and timeout management.

#### Key Components

| File | Purpose |
|------|---------|
| `circuitbreaker.go` | Circuit breaker pattern |
| `retry.go` | Exponential backoff retry |
| `timeout.go` | Timeout management |
| `config.go` | Configuration builder |
| `rest/client.go` | REST client |
| `graphql/client.go` | GraphQL client |

#### Circuit Breaker

```go
cb := integration.NewCircuitBreaker("github-api", integration.CircuitBreakerConfig{
    FailureThreshold: 5,           // Failures to open
    Timeout:          60 * time.Second, // Time before half-open
    HalfOpenRequests: 3,           // Successes to close
})

err := cb.Execute(ctx, func() error {
    return callExternalAPI()
})
```

**States**: Closed → Open → HalfOpen → Closed

#### Retry Logic

```go
result, err := integration.DoWithResult(ctx, integration.RetryConfig{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    30 * time.Second,
    Multiplier:  2.0,
    Jitter:      0.25,
}, func() (string, error) {
    return callAPI()
})
```

#### REST Client

```go
client := rest.NewClient(integration.NewConfigBuilder("github").
    BaseURL("https://api.github.com").
    BearerAuth(token).
    MaxRetries(3).
    Timeout(30 * time.Second).
    Build())

resp, err := client.Get(ctx, "/repos/owner/repo",
    rest.WithQuery(map[string]string{"page": "1"}),
    rest.WithHeader("Accept", "application/json"),
)

var repo Repository
resp.UnmarshalJSON(&repo)
```

#### GraphQL Client

```go
client := graphql.NewClient(config)

var result struct {
    Repository struct {
        Name string
    }
}

err := client.Query(ctx, `
    query($owner: String!, $name: String!) {
        repository(owner: $owner, name: $name) {
            name
        }
    }
`, map[string]interface{}{
    "owner": "codeai",
    "name":  "platform",
}, &result)
```

#### Middleware Stack

```go
client.Use(
    rest.NewLoggingMiddleware([]string{"password", "token"}),
    rest.NewMetricsMiddleware("github-api"),
    rest.NewRateLimitMiddleware(),
    rest.NewCompressionMiddleware(),
)
```

---

### Notification Module

**Location**: `internal/notification/`

**Purpose**: Email notification service with Brevo integration and templating.

#### Key Components

| File | Purpose |
|------|---------|
| `email/email_service.go` | Email service facade |
| `email/config.go` | Email configuration |
| `email/templates/registry.go` | Template management |
| `email/repository/*.go` | Email log persistence |
| `email/subscriber/*.go` | Event bus integration |

#### Email Service

```go
service := notification.NewEmailService(config, brevoClient, repository)

// Send workflow notification
err := service.SendWorkflowNotification(ctx, workflowID, "completed", recipients)

// Send custom email
err := service.SendCustomEmail(ctx, notification.EmailRequest{
    To:       []string{"user@example.com"},
    Template: "welcome",
    Data:     map[string]interface{}{"name": "John"},
})
```

#### Templates

| Template | Purpose |
|----------|---------|
| `workflow_completed` | Workflow success notification |
| `workflow_failed` | Workflow failure notification |
| `job_completed` | Job success |
| `job_failed` | Job failure |
| `test_results` | Test suite results |
| `welcome` | User onboarding |

#### Configuration

```go
type Config struct {
    BREVOAPIKey             string        // env: BREVO_API_KEY
    SenderName              string        // "CodeAI Platform"
    SenderAddress           string        // "noreply@codeai.io"
    EnableAutoNotifications bool          // true
    BatchSize               int           // 50
    RetryAttempts           int           // 3
    RetryDelaySeconds       int           // 5
}
```

#### Event Integration

```go
subscriber := notification.NewEmailEventSubscriber(service, resolver)
subscriber.RegisterWithBus(eventBus)

// Automatically sends emails on:
// - workflow.completed → workflow_completed template
// - workflow.failed → workflow_failed template
// - job.completed → job_completed template
// - test.suite.completed → test_results template
```

---

### Webhook Module

**Location**: `internal/webhook/`

**Purpose**: Webhook delivery with retry logic, signature verification, and event bus integration.

#### Key Components

| Directory | Purpose |
|-----------|---------|
| `service/` | Webhook orchestration |
| `repository/` | Webhook and delivery persistence |
| `security/` | HMAC-SHA256 signature |
| `queue/` | Async delivery worker pool |
| `retry/` | Automatic retry handler |
| `subscriber/` | Event bus integration |

#### Webhook Service

```go
service := webhook.NewService(client, repository, webhook.Config{
    MaxFailureCount: 10,
    DefaultTimeout:  30 * time.Second,
})

// Register webhook
id, err := service.RegisterWebhook(ctx, webhook.RegisterRequest{
    URL:    "https://example.com/webhook",
    Events: []string{"workflow.completed", "job.failed"},
    Secret: "webhook-secret",
    Headers: map[string]string{"X-Custom": "value"},
})

// Deliver to all webhooks for event
err := service.DeliverWebhooksForEvent(ctx, event)
```

#### Signature Verification

```go
// Signing (server-side)
signature := security.SignPayload(secret, payload)
security.AddSignatureHeaders(headers, secret, payload)

// Verification (receiver-side)
signature := security.ExtractSignature(headers)
valid := security.VerifySignature(secret, payload, signature)
```

**Headers Added**:
- `X-Webhook-Signature`: HMAC-SHA256 signature
- `X-Webhook-Signature-Algorithm`: `sha256`
- `X-Webhook-Timestamp`: Unix timestamp (replay protection)

#### Delivery Queue

```go
queue := webhook.NewDeliveryQueue(webhook.QueueConfig{
    QueueSize:   1000,
    WorkerCount: 10,
})
queue.Start(ctx)
defer queue.Stop()

// Non-blocking enqueue
queue.Enqueue(deliveryItem)
```

#### Retry Handler

```go
handler := webhook.NewRetryHandler(repository, service, webhook.RetryConfig{
    CheckInterval: 1 * time.Minute,
    BatchSize:     100,
    MaxRetries:    5,
})
handler.Start(ctx)
defer handler.Stop()
```

---

### Logging Module

**Location**: `pkg/logging/`

**Purpose**: Structured logging with context propagation and sensitive data redaction.

#### Key Components

| File | Purpose |
|------|---------|
| `config.go` | Logging configuration |
| `logger.go` | Core logger (wraps slog) |
| `middleware.go` | HTTP request logging |
| `redactor.go` | Sensitive data protection |
| `tracing.go` | Distributed tracing context |

#### Configuration

```go
config := logging.Config{
    Level:     "info",           // debug, info, warn, error
    Format:    "json",           // json, text
    Output:    "stdout",         // stdout, stderr, /path/to/file
    AddSource: true,             // Include file:line
    SampleRate: 1.0,             // 1.0 = log all, 0.1 = 10%
    SlowQueryThreshold: 100 * time.Millisecond,
}

logger := logging.New(config)
```

#### Context-Aware Logging

```go
// Add context
logger := logger.WithModule("api").
    WithOperation("CreateDeployment").
    WithEntity("deployment", deploymentID)

// Automatic context extraction
log := logging.LoggerFromContext(ctx, logger)
log.Info("deployment created")
```

#### Sensitive Data Redaction

Protected fields: `password`, `secret`, `token`, `api_key`, `authorization`, `credential`, `credit_card`, `ssn`, `private_key`, `access_token`, `refresh_token`, `bearer`

```go
// Automatic redaction in logs
log.Info("user login", "user", userData) // password fields redacted

// Manual redaction
safe := logging.RedactSensitive(sensitiveMap)
```

#### HTTP Middleware

```go
r.Use(logging.RequestLogger(logger, logging.MiddlewareConfig{
    Verbosity: logging.VerbosityStandard,
    SkipPaths: []string{"/health", "/metrics"},
}))

// Logs: method, path, status, duration, request_id, trace_id
```

#### Distributed Tracing

```go
// Create trace context
tc := logging.NewTraceContext().
    WithRequestID(requestID).
    WithTraceID(traceID).
    WithUserID(userID)

ctx = tc.ToContext(ctx)

// Extract in downstream
tc := logging.FromContext(ctx)
```

---

### Metrics Module

**Location**: `pkg/metrics/`

**Purpose**: Prometheus metrics for HTTP, database, workflow, and integration monitoring.

#### Key Components

| File | Purpose |
|------|---------|
| `config.go` | Metrics configuration |
| `registry.go` | Prometheus registry |
| `http.go` | HTTP request metrics |
| `middleware.go` | HTTP metrics middleware |
| `database.go` | Database query metrics |
| `workflow.go` | Workflow execution metrics |
| `integration.go` | External API metrics |
| `handler.go` | `/metrics` endpoint |

#### Configuration

```go
config := metrics.DefaultConfig().
    WithVersion("v1.0.0").
    WithEnvironment("production").
    WithInstance("pod-123")

registry := metrics.NewRegistry(config)
```

#### HTTP Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `codeai_http_requests_total` | Counter | method, path, status_code |
| `codeai_http_request_duration_seconds` | Histogram | method, path |
| `codeai_http_request_size_bytes` | Histogram | method, path |
| `codeai_http_response_size_bytes` | Histogram | method, path |
| `codeai_http_active_requests` | Gauge | method, path |

```go
r.Use(metrics.HTTPMiddleware(registry))
```

#### Database Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `codeai_db_queries_total` | Counter | operation, table, status |
| `codeai_db_query_duration_seconds` | Histogram | operation, table |
| `codeai_db_query_errors_total` | Counter | operation, table, error_type |
| `codeai_db_connections_active` | Gauge | - |

```go
timer := registry.DB().NewQueryTimer(metrics.OperationSelect, "users")
defer timer.Done(err)

rows, err := db.Query(sql)
```

#### Workflow Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `codeai_workflow_executions_total` | Counter | workflow_name, status |
| `codeai_workflow_execution_duration_seconds` | Histogram | workflow_name |
| `codeai_workflow_active_count` | Gauge | workflow_name |
| `codeai_workflow_step_duration_seconds` | Histogram | workflow_name, step_name |

```go
timer := registry.Workflow().NewExecutionTimer("data-pipeline")
defer timer.Success() // or .Failure(), .Timeout()
```

#### Prometheus Endpoint

```go
registry.RegisterMetricsRoute(router) // Adds /metrics endpoint

// With authentication
handler := registry.HandlerWithAuth(func(r *http.Request) bool {
    return r.Header.Get("Authorization") == "Bearer secret"
})
```

---

## Design Patterns

### Repository Pattern

All database access goes through repository interfaces:

```go
type DeploymentRepository interface {
    Create(ctx context.Context, d *Deployment) error
    GetByID(ctx context.Context, id string) (*Deployment, error)
    List(ctx context.Context, limit, offset int) ([]*Deployment, error)
    Update(ctx context.Context, d *Deployment) error
    Delete(ctx context.Context, id string) error
    WithTx(tx *sql.Tx) *DeploymentRepository
}
```

### Dependency Injection

Services receive dependencies through constructors:

```go
func NewHandler(
    deployments *repository.DeploymentRepository,
    configs *repository.ConfigRepository,
    validator *validator.Validate,
) *Handler {
    return &Handler{
        deployments: deployments,
        configs:     configs,
        validator:   validator,
    }
}
```

### Middleware Chain

HTTP middleware composes in order:

```go
r.Use(middleware.RequestID)     // 1. Add request ID
r.Use(middleware.Logger)        // 2. Log request
r.Use(auth.RequireAuth())       // 3. Authenticate
r.Use(rbac.RequirePermission()) // 4. Authorize
```

### Builder Pattern

Complex configurations use builders:

```go
config := integration.NewConfigBuilder("github").
    BaseURL("https://api.github.com").
    BearerAuth(token).
    MaxRetries(3).
    Timeout(30 * time.Second).
    Build()
```

### Saga Pattern

Distributed transactions with compensation:

```go
saga := NewSagaBuilder("order-process").
    AddStep("reserve-inventory", ReserveActivity, nil, ReleaseInventory).
    AddStep("charge-payment", ChargeActivity, nil, RefundPayment).
    AddStep("ship-order", ShipActivity, nil, CancelShipment).
    Build()
```

### Observer Pattern (Event Bus)

Loose coupling through events:

```go
// Publisher
eventBus.Publish(ctx, Event{Type: "order.created", Data: order})

// Subscribers (decoupled)
eventBus.Subscribe("order.created", webhookSubscriber)
eventBus.Subscribe("order.created", emailSubscriber)
eventBus.Subscribe("order.created", metricsSubscriber)
```

### Circuit Breaker Pattern

Prevent cascading failures:

```go
cb := NewCircuitBreaker("payment-api", config)

err := cb.Execute(ctx, func() error {
    return paymentAPI.Charge(amount)
})
// Returns ErrCircuitOpen if too many recent failures
```

### Decorator Pattern

Add behavior without modifying core:

```go
// CachedStorage decorates Storage
type CachedStorage struct {
    backend Storage
    cache   map[string]*cachedRole
    ttl     time.Duration
}

storage := NewCachedStorage(
    NewMemoryStorage(),
    WithTTL(5 * time.Minute),
)
```

---

## Testing Strategy

### Unit Tests

Located alongside source files (`*_test.go`):

```go
func TestDeploymentRepository_Create(t *testing.T) {
    db := testing.SetupTestDB(t)
    defer testing.TeardownTestDB(t, db)

    repo := repository.NewDeploymentRepository(db)
    deployment := models.NewDeployment("test")

    err := repo.Create(context.Background(), deployment)

    assert.NoError(t, err)
    assert.NotEmpty(t, deployment.ID)
}
```

### Integration Tests

Located in `test/integration/`:

```go
func TestAPI_CreateAndGetDeployment(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()

    // Create
    resp, err := http.Post(server.URL+"/deployments", "application/json", body)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    // Get
    resp, err = http.Get(server.URL+"/deployments/"+id)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### Test Helpers

```go
// In-memory database for tests
db := database.SetupTestDB(t)
defer database.TeardownTestDB(t, db)

// Seed test data
data := database.SeedTestData(t, db)
// data.Config, data.Deployment, data.Execution
```

### Benchmarks

```go
func BenchmarkParser_SmallProgram(b *testing.B) {
    input := `var x = 1`
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(input)
    }
}
```

### Coverage Requirements

- Unit tests: 80%+ coverage
- Integration tests: Critical paths covered
- Run with: `go test -cover ./...`

---

## Configuration

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_HOST` | PostgreSQL host | `localhost` |
| `DATABASE_PORT` | PostgreSQL port | `5432` |
| `DATABASE_NAME` | Database name | `codeai` |
| `DATABASE_USER` | Database user | `postgres` |
| `DATABASE_PASSWORD` | Database password | - |
| `DATABASE_SSLMODE` | SSL mode | `disable` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `TEMPORAL_HOST` | Temporal server | `localhost:7233` |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_FORMAT` | Log format | `json` |
| `BREVO_API_KEY` | Email API key | - |
| `JWT_ISSUER` | JWT issuer | - |
| `JWT_AUDIENCE` | JWT audience | - |
| `JWKS_URL` | JWKS endpoint | - |

### Config Files

Located in `config/`:

```yaml
# config/development.yaml
server:
  port: 8080
  timeout: 60s

database:
  host: localhost
  port: 5432
  name: codeai_dev

logging:
  level: debug
  format: text
```

### Loading Configuration

```go
config := config.Load()

// Or from specific file
config := config.LoadFile("config/production.yaml")

// Or from environment
config := config.LoadEnv()
```

### Secrets Management

Sensitive values should come from:
1. Environment variables (preferred)
2. Secret management service (Vault, AWS Secrets Manager)
3. Kubernetes secrets

Never commit secrets to version control.

---

## Quick Reference

### Starting the Server

```bash
# Build
make build

# Run with defaults
./codeai serve

# Run with custom config
./codeai serve --config config/production.yaml
```

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Integration tests
make test-integration
```

### Database Operations

```bash
# Run migrations
./codeai migrate up

# Rollback
./codeai migrate down

# Status
./codeai migrate status
```

### Health Check

```bash
curl http://localhost:8080/health
```

### Metrics

```bash
curl http://localhost:8080/metrics
```
