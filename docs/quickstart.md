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
| PostgreSQL | 14+ | Relational database (default) |
| MongoDB | 4.2+ | Document database (optional) |
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

- **MongoDB** - Alternative document database option (`internal/database/mongodb`)
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

Let's create a complete Blog API with REST endpoints using CodeAI's DSL. This example demonstrates:

- **Database Schema**: PostgreSQL tables with relationships and constraints
- **REST API Endpoints**: Full CRUD operations (GET, POST, PUT, DELETE)
- **Request/Response Types**: Type-safe request and response handling
- **Middleware**: Authentication, authorization, and rate limiting
- **Handler Logic**: `do` blocks with validate, create, find, update, delete operations
- **Nested Resources**: User posts accessed via `/users/:userId/posts`
- **Event Emission**: Trigger events like `user_created` and `post_created`

### Step 1: Create a DSL File

Create a file `myapp.cai` with the following content:

```cai
// myapp.cai - Blog API with REST Endpoints

// =============================================================================
// Configuration
// =============================================================================
config {
    name: "blog-api"
    version: "1.0.0"
    database_type: "postgres"
    api_prefix: "/api/v1"
}

// =============================================================================
// Database Schema
// =============================================================================
database postgres {
    // Users table
    users {
        id         uuid primary_key default "gen_random_uuid()"
        email      varchar(255) unique not_null
        username   varchar(100) unique not_null
        name       varchar(255) not_null
        role       varchar(50) default "'user'"
        created_at timestamp default "now()"
        updated_at timestamp default "now()"
    }

    // Posts table
    posts {
        id         uuid primary_key default "gen_random_uuid()"
        user_id    uuid references "users(id)"
        title      varchar(255) not_null
        slug       varchar(255) unique not_null
        content    text not_null
        status     varchar(50) default "'draft'"
        created_at timestamp default "now()"
        updated_at timestamp default "now()"
    }
}

// =============================================================================
// REST API Endpoints
// =============================================================================

// List all users with pagination
endpoint GET "/users" {
    request SearchParams from query
    response UserList status 200
    do {
        users = find(User) where "deleted_at IS NULL"
    }
}

// Get a single user by ID
endpoint GET "/users/:id" {
    request UserID from path
    response User status 200
    do {
        validate(request)
        user = find(User, id)
    }
}

// Create a new user
endpoint POST "/users" {
    middleware rate_limit
    request CreateUser from body
    response User status 201
    do {
        validate(request)
        user = create(User, request)
        emit(user_created, user)
    }
}

// Update an existing user
endpoint PUT "/users/:id" {
    middleware auth
    request UpdateUser from body
    response User status 200
    do {
        validate(request)
        authorize(request, "admin")
        user = find(User, id)
        updated = update(user, request)
    }
}

// Delete a user
endpoint DELETE "/users/:id" {
    middleware auth
    middleware admin
    response Empty status 204
    do {
        authorize(request, "admin")
        user = find(User, id)
        delete(user)
    }
}

// =============================================================================
// Post Endpoints
// =============================================================================

// List all posts
endpoint GET "/posts" {
    request PostSearchParams from query
    response PostList status 200
    do {
        posts = find(Post)
    }
}

// Create a new post
endpoint POST "/posts" {
    middleware auth
    request CreatePost from body
    response Post status 201
    do {
        validate(request)
        post = create(Post, request)
        emit(post_created, post)
    }
}

// Get posts for a specific user
endpoint GET "/users/:userId/posts" {
    request UserPostsParams from query
    response PostList status 200
    do {
        validate(request)
        posts = find(Post) where "user_id = :userId"
    }
}
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

**Create a User:**

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "username": "alice",
    "name": "Alice Johnson",
    "role": "author"
  }'
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "username": "alice",
  "name": "Alice Johnson",
  "role": "author",
  "created_at": "2024-01-12T15:00:00Z",
  "updated_at": "2024-01-12T15:00:00Z"
}
```

**List All Users:**

```bash
curl http://localhost:8080/api/v1/users
```

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "alice@example.com",
      "username": "alice",
      "name": "Alice Johnson",
      "role": "author"
    }
  ],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 1
  }
}
```

**Get a Single User:**

```bash
curl http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "username": "alice",
  "name": "Alice Johnson",
  "role": "author"
}
```

**Create a Blog Post:**

```bash
curl -X POST http://localhost:8080/api/v1/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "Getting Started with CodeAI",
    "slug": "getting-started-with-codeai",
    "content": "CodeAI is a declarative backend framework...",
    "status": "published",
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440111",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Getting Started with CodeAI",
  "slug": "getting-started-with-codeai",
  "content": "CodeAI is a declarative backend framework...",
  "status": "published",
  "created_at": "2024-01-12T15:05:00Z"
}
```

**List All Posts:**

```bash
curl http://localhost:8080/api/v1/posts
```

**Get User's Posts (Nested Resource):**

```bash
curl http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000/posts
```

**Update a User:**

```bash
curl -X PUT http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "Alice Smith",
    "role": "editor"
  }'
```

**Delete a User (Admin Only):**

```bash
curl -X DELETE http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
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
| `CODEAI_DATABASE_TYPE` | `postgres` | Database type (postgres or mongodb) |
| `CODEAI_MONGODB_URI` | (empty) | MongoDB connection URI |
| `CODEAI_MONGODB_DATABASE` | (empty) | MongoDB database name |
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

**MongoDB Setup (Optional):**

```bash
# Start MongoDB locally
mongod --replSet rs0

# Initialize replica set (required for transactions)
mongosh --eval "rs.initiate()"

# Or use Docker
docker run -d --name mongodb \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password \
  mongo:7 --replSet rs0
```

**MongoDB Environment Variables:**

```bash
export CODEAI_DATABASE_TYPE=mongodb
export CODEAI_MONGODB_URI=mongodb://localhost:27017
export CODEAI_MONGODB_DATABASE=codeai
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
