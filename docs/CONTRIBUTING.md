# Contributing to CodeAI

Thank you for your interest in contributing to CodeAI! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Code Style Guidelines](#code-style-guidelines)
- [Development Workflow](#development-workflow)
- [Adding New Features](#adding-new-features)
- [Reporting Issues](#reporting-issues)
- [Community Guidelines](#community-guidelines)

---

## Getting Started

### Prerequisites

- **Go 1.24 or later** - [Download Go](https://go.dev/dl/)
- **Git** - For version control
- **Make** - For running build commands (optional but recommended)
- **Docker** - Required for running integration tests with testcontainers

### Fork and Clone the Repository

1. Fork the repository on GitHub by clicking the "Fork" button

2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/codeai.git
   cd codeai
   ```

3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/bargom/codeai.git
   ```

4. Keep your fork up to date:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

### Development Environment Setup

1. **Install Go dependencies:**
   ```bash
   go mod download
   ```

2. **Verify your setup:**
   ```bash
   go mod tidy
   go vet ./...
   ```

3. **Set up pre-commit hooks (recommended):**
   ```bash
   # Create a pre-commit hook to run formatting and linting
   cat > .git/hooks/pre-commit << 'EOF'
   #!/bin/sh
   go fmt ./...
   go vet ./...
   EOF
   chmod +x .git/hooks/pre-commit
   ```

### Running the Project Locally

**Build the application:**
```bash
# Using Make
make build

# Or directly with Go
go build -o bin/codeai ./cmd/codeai
```

**Run the application:**
```bash
# Using Make
make run

# Or directly
./bin/codeai
```

**Available CLI commands:**
```bash
./bin/codeai --help           # Show available commands
./bin/codeai parse <file>     # Parse a DSL file
./bin/codeai validate <file>  # Validate a DSL file
./bin/codeai server           # Start the API server
```

### Running Tests

**Run all tests:**
```bash
make test
```

**Run specific test categories:**
```bash
# Unit tests only
make test-unit

# Integration tests (requires Docker)
make test-integration

# CLI integration tests
make test-cli

# All tests
make test-all
```

**Run tests with coverage:**
```bash
make test-coverage
# Coverage report will be generated at coverage.html
```

**Run benchmarks:**
```bash
make bench

# Parser benchmarks only
make bench-parse

# Validator benchmarks only
make bench-validate
```

**Run a specific test:**
```bash
go test -v -run TestParseVarDecl ./internal/parser/...
```

---

## Code Style Guidelines

### Go Formatting

All Go code must be formatted using standard Go tools:

```bash
# Format all code
go fmt ./...

# Or use goimports for import organization
goimports -w .
```

**Key formatting rules:**
- Use tabs for indentation (standard Go style)
- Maximum line length: 100 characters (soft limit)
- Run `go fmt` before every commit

### Naming Conventions

**Packages:**
- Use short, lowercase, single-word names
- Avoid underscores or mixedCaps
- Example: `parser`, `validator`, `engine`

**Variables and Functions:**
- Use camelCase for unexported identifiers: `myVariable`, `parseInput`
- Use PascalCase for exported identifiers: `ParseFile`, `ValidateAST`
- Prefer short names for local variables with limited scope: `i`, `n`, `err`
- Use descriptive names for package-level variables

**Constants:**
- Use PascalCase for exported constants: `MaxRetries`, `DefaultTimeout`
- Group related constants using `const` blocks

**Interfaces:**
- Use `-er` suffix for single-method interfaces: `Reader`, `Writer`, `Parser`
- Name interfaces by what they do, not what they are

**Examples:**
```go
// Good
type Parser interface {
    Parse(input string) (*ast.Program, error)
}

func parseStatement(input string) (*Statement, error) {
    // ...
}

// Bad
type IParser interface {  // Don't use I prefix
    Parse(input string) (*ast.Program, error)
}
```

### Package Organization

```
codeai/
├── cmd/codeai/          # Application entry points
│   ├── cmd/             # Cobra commands
│   └── main.go          # Main entry point
├── internal/            # Private application code
│   ├── api/             # HTTP API handlers
│   ├── ast/             # Abstract syntax tree types
│   ├── auth/            # Authentication/authorization
│   ├── cache/           # Caching layer
│   ├── database/        # Database access
│   ├── engine/          # Execution engine
│   ├── llm/             # LLM client interfaces
│   ├── parser/          # DSL parser
│   ├── query/           # Query language
│   ├── validation/      # Input validation
│   └── validator/       # AST validation
├── pkg/                 # Public packages (importable by external projects)
│   └── types/           # Shared type definitions
└── test/                # Integration and E2E tests
    ├── cli/             # CLI tests
    ├── fixtures/        # Test data
    ├── integration/     # Integration tests
    └── performance/     # Benchmarks
```

**Guidelines:**
- `internal/` packages cannot be imported by external projects
- `pkg/` packages can be imported by external projects
- Keep packages focused and cohesive
- Avoid circular dependencies

### Comment and Documentation Standards

**Package comments:**
```go
// Package parser provides a Participle-based parser for the CodeAI DSL.
// It transforms DSL source code into an abstract syntax tree (AST).
package parser
```

**Function comments:**
```go
// Parse parses the input string and returns an AST Program.
// It returns an error if the input contains syntax errors.
func Parse(input string) (*ast.Program, error) {
    // ...
}
```

**Inline comments:**
```go
// Use inline comments sparingly and only when the code isn't self-explanatory
result := complexCalculation(x, y) // Fermat's theorem edge case
```

**Documentation requirements:**
- All exported types, functions, and methods must have doc comments
- Doc comments should be complete sentences starting with the name being documented
- Use examples in `_test.go` files with `Example` prefix

### Error Handling Patterns

**Always handle errors explicitly:**
```go
// Good
result, err := Parse(input)
if err != nil {
    return fmt.Errorf("parsing input: %w", err)
}

// Bad - ignoring errors
result, _ := Parse(input)
```

**Wrap errors with context:**
```go
// Good
if err := db.Save(record); err != nil {
    return fmt.Errorf("saving config %s: %w", record.ID, err)
}

// Avoid generic error messages
if err != nil {
    return fmt.Errorf("operation failed: %w", err) // Too vague
}
```

**Custom error types:**
```go
// Define sentinel errors for expected error conditions
var (
    ErrNotFound     = errors.New("not found")
    ErrInvalidInput = errors.New("invalid input")
)

// Use custom error types for rich error information
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}
```

---

## Development Workflow

### Creating Feature Branches

Always create a new branch for your work:

```bash
# Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name

# For bug fixes
git checkout -b fix/issue-description

# For documentation
git checkout -b docs/description
```

**Branch naming conventions:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or changes

### Commit Message Format

Follow the conventional commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(parser): add support for array literals

Add lexer tokens and grammar rules for array literal expressions.
Arrays can contain mixed types and nested arrays.

Closes #42
```

```
fix(validator): handle nil pointer in type checker

The type checker was panicking when encountering a nil expression
in an if statement condition. Added nil check to prevent crash.

Fixes #123
```

```
docs(api): update REST API endpoint documentation

- Add examples for all endpoints
- Document error response formats
- Add authentication section
```

### Pull Request Process

1. **Before creating a PR:**
   ```bash
   # Ensure tests pass
   make test

   # Run linting
   make lint

   # Format code
   go fmt ./...
   ```

2. **Create the PR:**
   - Use a clear, descriptive title
   - Fill out the PR template completely
   - Link related issues using "Closes #XX" or "Fixes #XX"
   - Add appropriate labels

3. **PR description template:**
   ```markdown
   ## Summary
   Brief description of the changes.

   ## Changes
   - Change 1
   - Change 2

   ## Testing
   How were these changes tested?

   ## Checklist
   - [ ] Tests added/updated
   - [ ] Documentation updated
   - [ ] Code formatted with `go fmt`
   - [ ] No new warnings from `go vet`
   ```

### Code Review Expectations

**As a contributor:**
- Respond to feedback promptly
- Explain your reasoning for design decisions
- Be open to suggestions
- Request re-review after making changes

**Reviewers will check for:**
- Code correctness and test coverage
- Adherence to code style guidelines
- Clear documentation
- No introduced security vulnerabilities
- Performance considerations

### CI/CD Checks That Must Pass

All PRs must pass the following automated checks:

| Check | Command | Description |
|-------|---------|-------------|
| Unit Tests | `make test-unit` | All unit tests must pass |
| Integration Tests | `make test-integration` | Integration tests must pass |
| Linting | `make lint` / `go vet ./...` | No lint errors |
| Build | `make build` | Project must compile |

---

## Adding New Features

### DSL Syntax Additions

When adding new syntax to the CodeAI DSL:

1. **Update the lexer** (`internal/parser/parser.go`):
   - Add new tokens to the lexer definition
   - Consider token precedence

2. **Update grammar structs** (`internal/parser/parser.go`):
   - Add Participle grammar structs for new syntax
   - Follow existing patterns for expression parsing

3. **Update AST types** (`internal/ast/`):
   - Add new AST node types
   - Implement the `Node` interface

4. **Update the transformer** (`internal/parser/parser.go`):
   - Transform Participle structs to AST nodes

5. **Update the validator** (`internal/validator/`):
   - Add validation rules for new syntax
   - Update type checker if needed

6. **Update documentation**:
   - Update `docs/dsl_language_spec.md`
   - Add examples to `docs/dsl_cheatsheet.md`

### New Modules or Packages

When adding a new package:

1. **Choose the right location:**
   - `internal/` for application-specific code
   - `pkg/` for code that should be importable

2. **Create the package structure:**
   ```
   internal/newpkg/
   ├── newpkg.go          # Main implementation
   ├── newpkg_test.go     # Unit tests
   ├── errors.go          # Error definitions (if needed)
   └── doc.go             # Package documentation (optional)
   ```

3. **Add package documentation:**
   ```go
   // Package newpkg provides functionality for X.
   //
   // Example usage:
   //
   //     result, err := newpkg.DoSomething(input)
   //
   package newpkg
   ```

### Testing Requirements

**All new features must include:**

1. **Unit tests** with table-driven approach:
   ```go
   func TestNewFeature(t *testing.T) {
       t.Parallel()

       tests := []struct {
           name    string
           input   string
           want    string
           wantErr bool
       }{
           {
               name:  "valid input",
               input: "test",
               want:  "expected",
           },
           {
               name:    "invalid input",
               input:   "",
               wantErr: true,
           },
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               got, err := NewFeature(tt.input)
               if tt.wantErr {
                   assert.Error(t, err)
                   return
               }
               require.NoError(t, err)
               assert.Equal(t, tt.want, got)
           })
       }
   }
   ```

2. **Coverage requirements:**
   - Minimum 80% coverage for new code
   - Test both success and error paths
   - Test edge cases

3. **Integration tests** (when applicable):
   - Add tests in `test/integration/`
   - Use build tag `// +build integration`

### Documentation Requirements

New features require:

1. **Code documentation:**
   - Doc comments on all exported types and functions
   - Example functions where appropriate

2. **User documentation:**
   - Update relevant docs in `docs/`
   - Add usage examples

3. **API documentation** (for HTTP endpoints):
   - Update `docs/api_reference.md`
   - Include request/response examples

---

## Reporting Issues

### Issue Template

When opening an issue, please use the appropriate template:

#### Bug Report

```markdown
## Bug Description
A clear and concise description of the bug.

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- OS: [e.g., macOS 14.0, Ubuntu 22.04]
- Go version: [e.g., 1.24.0]
- CodeAI version: [e.g., v1.0.0 or commit hash]

## Additional Context
Any other relevant information (logs, screenshots, etc.)
```

#### Feature Request

```markdown
## Feature Description
A clear description of the feature you'd like to see.

## Use Case
Describe the problem this feature would solve.

## Proposed Solution
If you have ideas on how to implement this, describe them here.

## Alternatives Considered
Any alternative solutions or features you've considered.

## Additional Context
Any other relevant information.
```

### Bug Report Format

Good bug reports include:

1. **Minimal reproduction case** - The smallest code that demonstrates the bug
2. **Environment details** - OS, Go version, CodeAI version
3. **Error messages** - Full error output, including stack traces
4. **Expected vs. actual behavior** - Clear description of both

**Example of a good bug report:**

```markdown
## Bug Description
Parser panics when parsing empty array literal `[]`.

## Steps to Reproduce
1. Create a file `test.cai` with content: `var arr = []`
2. Run `./codeai parse test.cai`
3. Observe panic

## Expected Behavior
Should parse successfully and create an empty array AST node.

## Actual Behavior
Panic with stack trace:
```
panic: runtime error: index out of range [0] with length 0
goroutine 1 [running]:
github.com/bargom/codeai/internal/parser.transformArray(...)
```

## Environment
- OS: macOS 14.2
- Go version: 1.24.0
- CodeAI version: commit f1d5332
```

### Feature Request Format

Good feature requests include:

1. **Clear use case** - Why is this feature needed?
2. **Proposed behavior** - How should it work?
3. **Examples** - If relevant, show example syntax or API

### Security Vulnerability Reporting

**Do NOT open public issues for security vulnerabilities.**

Instead:

1. Email security concerns to the maintainers privately
2. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

3. Allow reasonable time for a fix before public disclosure

---

## Community Guidelines

### Code of Conduct

We are committed to providing a welcoming and inclusive environment.

**Expected behavior:**
- Be respectful and considerate
- Use welcoming and inclusive language
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards others

**Unacceptable behavior:**
- Harassment, discrimination, or offensive comments
- Personal attacks
- Trolling or inflammatory remarks
- Publishing others' private information

### Communication Channels

- **GitHub Issues** - Bug reports, feature requests
- **GitHub Discussions** - Questions, ideas, general discussion
- **Pull Requests** - Code contributions

### Response Time Expectations

- **Issues**: Initial response within 1-2 weeks
- **Pull Requests**: Initial review within 1-2 weeks
- **Security issues**: Response within 48 hours

Please be patient - maintainers are often volunteers with limited time.

---

## Examples

### Good Commit Example

```
feat(parser): add support for multi-line string literals

Add triple-quoted string support (""") for multi-line strings.
The parser now correctly handles:
- Escaped characters within multi-line strings
- Preserving internal whitespace
- Proper line number tracking

This enables writing longer text blocks in DSL files without
escape sequences.

Closes #78
```

### Good PR Example

**Title:** `feat(api): add pagination to list endpoints`

**Description:**
```markdown
## Summary
Adds cursor-based pagination to all list endpoints in the REST API.

## Changes
- Add `PageParams` struct for pagination parameters
- Update list handlers to accept `limit` and `cursor` query params
- Add pagination middleware
- Update response format to include pagination metadata

## Testing
- Added unit tests for pagination logic
- Added integration tests for paginated endpoints
- Manual testing with curl against local server

## Checklist
- [x] Tests added/updated
- [x] Documentation updated (api_reference.md)
- [x] Code formatted with `go fmt`
- [x] No new warnings from `go vet`

Closes #92
```

### Good Issue Report Example

**Title:** `[BUG] Parser fails on nested function calls`

**Description:**
```markdown
## Bug Description
Nested function calls like `foo(bar(x))` cause a parse error.

## Steps to Reproduce
```codeai
var result = outer(inner(42))
```
Run: `./codeai parse test.cai`

## Expected Behavior
Should parse successfully with nested `CallExpr` nodes.

## Actual Behavior
Error: `unexpected token "(" at line 1, col 20`

## Environment
- OS: Ubuntu 22.04
- Go version: 1.24.0
- CodeAI version: v0.1.0 (commit abc1234)
```

---

## Thank You!

Thank you for contributing to CodeAI! Your contributions help make this project better for everyone.

If you have questions about contributing, feel free to open a GitHub Discussion or reach out to the maintainers.
