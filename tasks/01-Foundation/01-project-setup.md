# Task 001: Project Structure Setup

## Overview
Set up the foundational Go project structure for CodeAI runtime, including module initialization, directory hierarchy, and core configuration files.

## Phase
Phase 1: Foundation

## Priority
Critical - This is the first task that all other tasks depend on.

## Dependencies
None

## Description
Establish the complete Go project structure as specified in the implementation plan. This includes creating the directory hierarchy, initializing Go modules, and setting up the basic configuration for the development environment.

## Detailed Requirements

### 1. Initialize Go Module
```bash
go mod init github.com/codeai/codeai
```

### 2. Create Directory Structure
```
codeai/
├── cmd/
│   └── codeai/
│       └── main.go              # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── grammar.go           # Participle grammar definitions
│   │   ├── ast.go               # AST node types
│   │   ├── parser.go            # Main parser logic
│   │   └── lexer.go             # Custom lexer rules
│   ├── validator/
│   │   ├── validator.go         # Type checking and validation
│   │   ├── resolver.go          # Reference resolution
│   │   └── errors.go            # Validation error types
│   ├── runtime/
│   │   ├── engine.go            # Main execution engine
│   │   ├── context.go           # Request/execution context
│   │   ├── evaluator.go         # Expression evaluation
│   │   └── types.go             # Runtime type system
│   ├── modules/
│   │   ├── database/
│   │   ├── http/
│   │   ├── workflow/
│   │   ├── job/
│   │   ├── event/
│   │   ├── integration/
│   │   ├── cache/
│   │   └── auth/
│   └── stdlib/
│       ├── functions.go         # Built-in functions
│       ├── datetime.go          # Date/time functions
│       ├── string.go            # String functions
│       └── math.go              # Math functions
├── pkg/
│   └── codeai/
│       └── embed.go             # Embeddable runtime API
├── examples/
│   └── inventory-service/       # Example application
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

### 3. Core Dependencies (go.mod)
```go
module github.com/codeai/codeai

go 1.22

require (
    github.com/alecthomas/participle/v2 v2.1.1
    github.com/go-chi/chi/v5 v5.0.12
    github.com/go-chi/cors v1.2.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/hibiken/asynq v0.24.1
    github.com/jackc/pgx/v5 v5.5.3
    github.com/redis/go-redis/v9 v9.5.1
    github.com/spf13/viper v1.18.2
    github.com/stretchr/testify v1.8.4
    go.mongodb.org/mongo-driver v1.14.0
    github.com/spf13/cobra v1.8.0
)
```

### 4. Basic CLI Entry Point (cmd/codeai/main.go)
```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "codeai",
    Short: "CodeAI - LLM-Native Programming Language Runtime",
    Long:  `CodeAI is a domain-specific programming language designed for LLM code generation.`,
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### 5. Makefile for Common Operations
```makefile
.PHONY: build test run clean

VERSION := 1.0.0
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o bin/codeai ./cmd/codeai

test:
	go test -v ./...

run:
	go run ./cmd/codeai $(ARGS)

clean:
	rm -rf bin/

lint:
	golangci-lint run

fmt:
	go fmt ./...
```

### 6. Dockerfile
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o codeai ./cmd/codeai

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/codeai /usr/local/bin/
ENTRYPOINT ["codeai", "run"]
CMD ["/app"]
```

## Acceptance Criteria
- [ ] Go module initialized with correct module path
- [ ] All directories created as per structure
- [ ] go.mod contains all required dependencies
- [ ] Basic main.go compiles without errors
- [ ] `go build ./...` succeeds
- [ ] Makefile works for build, test, and run targets
- [ ] Dockerfile builds successfully
- [ ] README.md with basic project description

## Implementation Steps
1. Create all directories using mkdir -p
2. Initialize go.mod with `go mod init`
3. Create stub files for each package
4. Add dependencies to go.mod
5. Run `go mod tidy`
6. Create Makefile
7. Create Dockerfile
8. Verify build succeeds

## Estimated Complexity
Medium

## Files to Create
- `cmd/codeai/main.go`
- `internal/parser/parser.go` (stub)
- `internal/validator/validator.go` (stub)
- `internal/runtime/engine.go` (stub)
- `internal/modules/module.go` (interface definitions)
- `pkg/codeai/embed.go` (stub)
- `go.mod`
- `Makefile`
- `Dockerfile`
- `README.md`

## Testing Strategy
- Verify `go build ./...` compiles
- Verify `go test ./...` runs (even with no tests)
- Verify Docker image builds
- Verify CLI runs with `--help`

## Notes
- Use Go 1.22+ for latest features
- Follow Go project layout conventions
- Keep internal packages truly internal
- Use pkg/ only for embeddable public API
