# CodeAI - Project Instructions

## Project Overview

CodeAI is a declarative backend framework and DSL (Domain-Specific Language) designed for LLM code generation. LLMs are the primary code generators — they write `.cai` specifications that compile to a production-ready Go runtime.

**✅ Fully Implemented Features:**
- **MongoDB Collections**: Define collections with embedded documents, arrays, indexes
- **PostgreSQL Models**: Define tables with relationships and constraints
- **REST API Endpoints**: All HTTP methods (GET, POST, PUT, PATCH, DELETE)
- **Handler Logic**: Execute `do` blocks with validate, insert, query, update, delete operations
- **Middleware**: Authentication, authorization, rate limiting, logging
- **Runtime Server**: Fully functional HTTP server with route registration
- **Code Generation**: Generates handlers from DSL and registers routes dynamically

## Tech Stack

- **Language**: Go 1.24+
- **Parser**: Participle v2
- **Router**: Chi v5
- **Database**: PostgreSQL 14+
- **Cache**: Redis 6+
- **Workflows**: Temporal SDK
- **Jobs**: Asynq (distributed queue)
- **Metrics**: Prometheus
- **Logging**: slog (structured)

## Project Structure

```
cmd/codeai/          # CLI entry point
internal/
  api/               # HTTP handlers, router, middleware
  parser/            # DSL lexer and parser
  database/          # PostgreSQL, migrations, repositories
  auth/              # JWT, JWKS, RBAC
  cache/             # Redis and memory caching
  workflow/          # Temporal workflow engine
  scheduler/         # Asynq job scheduling
  event/             # Event bus and dispatching
  validator/         # AST validation
  openapi/           # OpenAPI 3.0 spec generation
  integration/       # External API clients with circuit breakers
  health/            # Health check endpoints
  shutdown/          # Graceful shutdown
  pagination/        # Cursor/offset pagination
  query/             # Query builder and executor
docs/                # Documentation (14 files)
examples/            # 5 progressive examples
validation_reports/  # Architecture validation reports
```

## Common Commands

```bash
# Build
make build

# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-cli

# Run benchmarks
make bench

# Lint
make lint

# Clean
make clean
```

## CLI Usage

```bash
# Parse and validate DSL
./bin/codeai parse myapp.cai
./bin/codeai validate myapp.cai

# Server operations (FULLY WORKING)
./bin/codeai server start --file myapp.cai --port 8080
./bin/codeai server start --db-name codeai
./bin/codeai server migrate --db-name codeai

# The server will automatically:
# - Generate handlers from endpoint definitions
# - Register routes with Chi router
# - Connect to database (MongoDB or PostgreSQL)
# - Start HTTP server and respond to requests
```

## Local Testing

A comprehensive test setup is available in `test/local/`:

```bash
# Quick parse/validate test
cd test/local
./test.sh

# Full server test with MongoDB and curl
./test-server.sh

# This will:
# 1. Start MongoDB in Docker
# 2. Parse and validate app.cai
# 3. Start the HTTP server
# 4. Test all CRUD endpoints
# 5. Verify responses
```

See `test/local/README.md` and `test/local/STATUS.md` for details.

## DSL File Extension

CodeAI source files use the `.cai` extension.

## Endpoint Syntax (Fully Implemented)

```cai
// Define database schema
database mongodb {
    collection User {
        _id: objectid, primary, auto
        name: string, required
        email: string, required, unique
        age: int, optional
        created_at: date, auto
    }
}

// Define REST endpoints
endpoint POST "/users" {
    request CreateUserRequest from body
    response User status 201

    do {
        validate(request)
        insert("users", request)
    }
}

endpoint GET "/users/:id" {
    request User from path
    response User status 200

    do {
        validate(id)
        query("users", id)
    }
}

endpoint PUT "/users/:id" {
    request UpdateUserRequest from body
    response User status 200

    do {
        validate(id)
        validate(request)
        update("users", id, request)
    }
}

endpoint DELETE "/users/:id" {
    response Empty status 204

    do {
        validate(id)
        delete("users", id)
    }
}

// Middleware can be added to endpoints
middleware auth {
    type authentication
    config {
        method: jwt
    }
}

endpoint GET "/admin/users" {
    middleware auth
    response UserList status 200
}
```

## Running the Server (Fully Working)

```bash
# Start server with a .cai file
./bin/codeai server start --file app.cai --port 8080

# The server will:
# 1. Parse and validate the DSL
# 2. Connect to MongoDB/PostgreSQL
# 3. Generate HTTP handlers from endpoints
# 4. Register routes with Chi router
# 5. Start listening on specified port

# Test endpoints
curl http://localhost:8080/users
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

## Key Design Principles

1. **LLM-Friendly Syntax**: Declarative constructs that align with LLM token prediction patterns
2. **Fault-Tolerant Parsing**: Flexible whitespace handling, lenient syntax variations
3. **Safe by Default**: No exposed unsafe primitives; runtime handles security
4. **Business-Domain Focus**: First-class constructs for common backend patterns

## Testing Strategy

- Unit tests: `internal/` packages
- Integration tests: `test/integration/`
- CLI tests: `test/cli/`
- Performance benchmarks: `test/performance/`
- Use testcontainers-go for database tests

## Code Style

- Follow standard Go conventions
- Use structured logging (slog)
- All public functions must have documentation
- Error messages should be actionable
