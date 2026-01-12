# CodeAI Developer Quick Start Guide

This guide will help you get started with CodeAI, a Go-based runtime for parsing and executing AI agent specifications written in a domain-specific language (DSL).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Project Structure](#project-structure)
- [First Application](#first-application)
- [Configuration](#configuration)
- [Development Workflow](#development-workflow)
- [Common Patterns](#common-patterns)

---

## Prerequisites

Before getting started, ensure you have the following installed:

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Main programming language |
| PostgreSQL | 14+ | Database for deployments, configs, and executions |
| Make | Any | Build automation |
| Git | Any | Version control |

### Verify Installation

```bash
# Check Go version
go version
# Expected: go version go1.24.0 darwin/arm64 (or similar)

# Check PostgreSQL
psql --version
# Expected: psql (PostgreSQL) 14.x or higher

# Check Make
make --version
# Expected: GNU Make 3.81 or higher
```

### Optional Tools

- **Redis** - Required for caching features (`internal/cache`)
- **Docker** - For running integration tests with testcontainers

---

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/bargom/codeai.git
cd codeai
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Binary

```bash
# Using Make (recommended)
make build

# Or directly with Go
go build -ldflags "-s -w" -o bin/codeai ./cmd/codeai
```

### 4. Verify the Build

```bash
./bin/codeai version
# Expected output: codeai version X.Y.Z

./bin/codeai --help
# Shows available commands
```

### Available Make Targets

```bash
make help
```

| Target | Description |
|--------|-------------|
| `make build` | Build the application binary |
| `make test` | Run all tests with race detector |
| `make test-unit` | Run unit tests only |
| `make test-integration` | Run integration tests |
| `make test-coverage` | Run tests with coverage report |
| `make bench` | Run benchmarks |
| `make lint` | Run linters (go vet) |
| `make clean` | Clean build artifacts |
| `make tidy` | Tidy go modules |

---

## Project Structure

```
codeai/
├── cmd/codeai/           # Application entry point
│   ├── main.go           # Main entry point
│   └── cmd/              # CLI commands (Cobra)
│       ├── root.go       # Root command and global flags
│       ├── parse.go      # Parse DSL files
│       ├── validate.go   # Validate DSL syntax
│       ├── deploy.go     # Deploy configurations
│       ├── server.go     # HTTP API server
│       ├── config.go     # Configuration management
│       └── completion.go # Shell completions
│
├── internal/             # Private application code
│   ├── api/              # HTTP API (Chi router)
│   │   ├── handlers/     # Request handlers
│   │   ├── types/        # Request/response types
│   │   └── server.go     # Server configuration
│   ├── ast/              # Abstract Syntax Tree types
│   ├── auth/             # JWT authentication & JWKS
│   ├── cache/            # Redis and memory caching
│   ├── database/         # PostgreSQL database layer
│   │   ├── models/       # Database models
│   │   ├── repository/   # Data access repositories
│   │   └── migrate.go    # Database migrations
│   ├── parser/           # Participle-based DSL parser
│   ├── validator/        # AST validation logic
│   ├── query/            # Query language support
│   ├── pagination/       # Pagination utilities
│   ├── rbac/             # Role-based access control
│   ├── validation/       # Request validation middleware
│   ├── webhook/          # Webhook handling
│   ├── workflow/         # Temporal workflow support
│   ├── scheduler/        # Job scheduling
│   ├── event/            # Event system
│   └── notification/     # Notification services
│
├── pkg/                  # Public packages
│   ├── types/            # Shared type definitions
│   ├── integration/      # Integration utilities
│   ├── logging/          # Logging utilities
│   └── metrics/          # Prometheus metrics
│
├── config/               # Configuration files
│   ├── events.yaml       # Event configuration
│   ├── notifications.yaml
│   ├── scheduler.yaml
│   ├── webhooks.yaml
│   └── workflows.yaml
│
├── test/                 # Test files
│   ├── fixtures/         # DSL test fixtures (.cai files)
│   ├── integration/      # Integration tests
│   ├── cli/              # CLI integration tests
│   └── performance/      # Benchmark tests
│
└── docs/                 # Documentation
```

---

## First Application

Let's create a simple CRUD API using CodeAI's DSL.

### Step 1: Create a DSL File

Create a file `myapp.cai` with the following content:

```cai
// myapp.cai - Simple Todo Application

// Configuration variables
var appName = "TodoApp"
var maxItems = 100
var debug = true

// Data structures
var statuses = ["pending", "in_progress", "completed"]
var priorities = ["low", "medium", "high"]

// Helper function for greeting
function greet(name) {
    var message = name
}

// Process each status
for status in statuses {
    var currentStatus = status
}

// Conditional configuration
if debug {
    var logLevel = "verbose"
} else {
    var logLevel = "info"
}

// Execute initialization
exec {
    echo "Initializing TodoApp"
}

var initialized = true
```

### Step 2: Parse the DSL File

```bash
# Parse and display the AST
./bin/codeai parse myapp.cai

# Parse with JSON output
./bin/codeai parse --output json myapp.cai

# Parse with verbose output
./bin/codeai parse --verbose myapp.cai
```

**Expected output (JSON format):**

```json
{
  "Statements": [
    {
      "Name": "appName",
      "Value": {
        "Value": "TodoApp"
      }
    },
    ...
  ]
}
```

### Step 3: Validate the DSL File

```bash
./bin/codeai validate myapp.cai
# Expected: Validation passed (no errors)
```

If there are errors, you'll see output like:

```
Validation errors:
  - Line 5: undefined variable 'unknownVar'
  - Line 10: duplicate function declaration 'greet'
```

### Step 4: Set Up the Database

```bash
# Create the database
createdb codeai

# Run migrations
./bin/codeai server migrate --db-name codeai --db-user postgres

# Or with verbose output
./bin/codeai server migrate -v --db-name codeai --db-user postgres
```

**Expected output:**

```
Running migration 1...
Running migration 2...
Running migration 3...
Migrations completed successfully
```

### Step 5: Start the HTTP Server

```bash
# Start with default settings (localhost:8080)
./bin/codeai server start

# Or with custom settings
./bin/codeai server start --host 0.0.0.0 --port 3000 --db-name codeai

# With verbose output
./bin/codeai server start -v --db-name codeai
```

**Expected output:**

```
Connected to database localhost:5432/codeai
Server listening on localhost:8080
```

### Step 6: Test the API Endpoints

**Health Check:**

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok","timestamp":"2024-01-12T15:00:00Z"}
```

**Create a Configuration:**

```bash
curl -X POST http://localhost:8080/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-todo-app",
    "content": "var appName = \"TodoApp\"\nvar debug = true"
  }'
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-todo-app",
  "content": "var appName = \"TodoApp\"\nvar debug = true",
  "created_at": "2024-01-12T15:00:00Z"
}
```

**List Configurations:**

```bash
curl http://localhost:8080/configs
```

```json
{
  "data": [...],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 1
  }
}
```

**Get a Single Configuration:**

```bash
curl http://localhost:8080/configs/550e8400-e29b-41d4-a716-446655440000
```

**Validate a Configuration:**

```bash
curl -X POST http://localhost:8080/configs/550e8400-e29b-41d4-a716-446655440000/validate
```

**Create a Deployment:**

```bash
curl -X POST http://localhost:8080/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production",
    "config_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

**List Deployments:**

```bash
curl http://localhost:8080/deployments
```

**Execute a Deployment:**

```bash
curl -X POST http://localhost:8080/deployments/550e8400-e29b-41d4-a716-446655440000/execute
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_CONFIG` | `$HOME/.cai.yaml` | Path to config file |
| `CODEAI_DB_HOST` | `localhost` | Database host |
| `CODEAI_DB_PORT` | `5432` | Database port |
| `CODEAI_DB_NAME` | `codeai` | Database name |
| `CODEAI_DB_USER` | `postgres` | Database user |
| `CODEAI_DB_PASSWORD` | (empty) | Database password |
| `CODEAI_DB_SSLMODE` | `disable` | Database SSL mode |
| `CODEAI_SERVER_HOST` | `localhost` | Server bind host |
| `CODEAI_SERVER_PORT` | `8080` | Server bind port |

### Database Setup

**PostgreSQL Configuration:**

```sql
-- Create database and user
CREATE DATABASE codeai;
CREATE USER codeai_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE codeai TO codeai_user;
```

**Connection String Format:**

```
postgresql://user:password@host:port/dbname?sslmode=disable
```

### JWT Authentication (Optional)

For secured endpoints, configure JWT settings:

```bash
# Set JWT secret for HS256 signing
export CODEAI_JWT_SECRET="your-256-bit-secret"

# Or use JWKS for RS256
export CODEAI_JWKS_URL="https://your-auth-provider/.well-known/jwks.json"
```

### Redis Caching (Optional)

```bash
export CODEAI_REDIS_URL="redis://localhost:6379/0"
export CODEAI_CACHE_TTL="5m"
```

---

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests (requires PostgreSQL)
make test-integration

# Run with coverage
make test-coverage
# Open coverage.html in browser
```

### Adding a New Feature

1. **Define the DSL syntax** in `internal/parser/parser.go`
2. **Add AST types** in `internal/ast/types.go`
3. **Implement validation** in `internal/validator/validator.go`
4. **Add API handlers** in `internal/api/handlers/`
5. **Write tests** in the corresponding `*_test.go` files

### Running Benchmarks

```bash
# Run all benchmarks
make bench

# Parser benchmarks only
make bench-parse

# Validator benchmarks only
make bench-validate
```

### Code Quality

```bash
# Run linter
make lint

# Format code (standard go fmt)
go fmt ./...

# Tidy dependencies
make tidy
```

### Building for Production

```bash
# Build optimized binary
make build

# The binary is at bin/codeai with stripped debug info
ls -la bin/codeai
```

---

## Common Patterns

### Authentication Middleware

```go
import "github.com/bargom/codeai/internal/auth"

// Create JWT validator with configuration
validator := auth.NewValidator(auth.Config{
    Issuer:   "your-issuer",
    Audience: "your-audience",
    Secret:   "your-256-bit-secret",     // For HS256
    // Or use public key/JWKS for RS256
    // PublicKey: "-----BEGIN PUBLIC KEY-----...",
    // JWKSURL:   "https://your-auth-provider/.well-known/jwks.json",
})

// Create middleware with the validator
authMiddleware := auth.NewMiddleware(validator)

// Apply to routes with different requirements
router.With(authMiddleware.Authenticate(auth.AuthRequired)).Get("/protected", handler)
router.With(authMiddleware.Authenticate(auth.AuthOptional)).Get("/optional", handler)
// Public routes don't need middleware

// Access user in handler
func handler(w http.ResponseWriter, r *http.Request) {
    user := auth.UserFromContext(r.Context())
    if user != nil {
        fmt.Printf("User ID: %s, Roles: %v\n", user.ID, user.Roles)
    }
}
```

### Request Validation

```go
import "github.com/bargom/codeai/internal/validation"

// Define request struct with validation tags
type CreateConfigRequest struct {
    Name    string `json:"name" validate:"required,min=1,max=255"`
    Content string `json:"content" validate:"required"`
}

// Validate in handler
func (h *Handler) CreateConfig(w http.ResponseWriter, r *http.Request) {
    var req CreateConfigRequest
    if err := h.decodeAndValidate(r, &req); err != nil {
        h.respondValidationError(w, err)
        return
    }
    // Process valid request...
}
```

### Error Handling

```go
import "github.com/bargom/codeai/internal/api/types"

// Standard error response
h.respondError(w, http.StatusNotFound, "resource not found")

// Validation error with details
h.respondJSON(w, http.StatusBadRequest, types.ErrorResponse{
    Error:   "validation failed",
    Details: map[string]string{
        "name": "is required",
    },
})
```

### Database Transactions

```go
import "github.com/bargom/codeai/internal/database"

// Use repository pattern
configRepo := repository.NewConfigRepository(db)

// Create with validation
config, err := configRepo.Create(ctx, &models.Config{
    Name:    "my-config",
    Content: "var x = 1",
})
if err != nil {
    return fmt.Errorf("failed to create config: %w", err)
}
```

### Caching

```go
import "github.com/bargom/codeai/internal/cache"

// Create Redis cache with configuration
redisCache, err := cache.NewRedisCache(cache.Config{
    URL: "redis://localhost:6379/0",
    // Or individual settings:
    // Password:     "",
    // DB:           0,
    // PoolSize:     10,
    // MinIdleConns: 5,
    // MaxRetries:   3,
})
if err != nil {
    log.Fatal(err)
}
defer redisCache.Close()

// Or use memory cache for development
memCache := cache.NewMemoryCache(cache.Config{
    DefaultTTL: 5 * time.Minute,
    MaxSize:    1000,
})

// Basic cache operations
err = redisCache.Set(ctx, "key", []byte("value"), 5*time.Minute)
data, err := redisCache.Get(ctx, "key")

// JSON operations (automatic serialization)
err = redisCache.SetJSON(ctx, "user:123", user, 10*time.Minute)
err = redisCache.GetJSON(ctx, "user:123", &user)

// Cache-aside pattern (recommended)
result, err := redisCache.GetOrSet(ctx, "expensive:key", 5*time.Minute, func() (any, error) {
    // This function is called only on cache miss
    return expensiveOperation()
})

// Bulk operations
values, err := redisCache.MGet(ctx, "key1", "key2", "key3")
err = redisCache.MSet(ctx, map[string][]byte{
    "key1": []byte("value1"),
    "key2": []byte("value2"),
}, 5*time.Minute)

// Check cache stats
stats := redisCache.Stats()
fmt.Printf("Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
```

### Pagination

```go
import "github.com/bargom/codeai/internal/pagination"

// Extract from request
params := pagination.FromRequest(r)
// params.Limit, params.Offset

// Apply to query
results, total, err := repo.List(ctx, params.Limit, params.Offset)

// Return paginated response
h.respondJSON(w, http.StatusOK, types.PaginatedResponse{
    Data: results,
    Pagination: types.Pagination{
        Limit:  params.Limit,
        Offset: params.Offset,
        Total:  total,
    },
})
```

---

## Next Steps

- Explore the [test fixtures](../test/fixtures/) for more DSL examples
- Read the [implementation status](./implementation_status.md) for feature roadmap
- Check the [CodeAI Implementation Plan](../CodeAI_Implementation_Plan.md) for architecture details

## Getting Help

If you encounter issues:

1. Check existing tests for usage examples
2. Run with `-v` flag for verbose output
3. Check the logs for error details
4. Open an issue on GitHub

---

*This guide covers CodeAI version 0.1.x. For the latest updates, check the repository.*
