# CodeAI - Project Instructions

## Project Overview

CodeAI is a declarative backend framework and DSL (Domain-Specific Language) designed for LLM code generation. LLMs are the primary code generators — they write `.cai` specifications that compile to a production-ready Go runtime.

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

# Server operations
./bin/codeai server start --db-name codeai
./bin/codeai server migrate --db-name codeai

# Code generation
./bin/codeai generate myapp.cai
```

## DSL File Extension

CodeAI source files use the `.cai` extension.

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
