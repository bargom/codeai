# CodeAI Test Suite

This directory contains the comprehensive test suite for the CodeAI project.

## Directory Structure

```
test/
├── README.md              # This file
├── fixtures/              # DSL example files for testing
│   ├── simple.cai         # Basic variable declarations
│   ├── functions.cai      # Function definitions and calls
│   ├── loops.cai          # For loops and arrays
│   ├── conditionals.cai   # If/else statements
│   ├── exec.cai           # Exec blocks with shell commands
│   ├── complex.cai        # Combination of all features
│   ├── invalid_*.cai      # Programs that should fail validation
├── local/                 # Local manual testing
│   ├── README.md          # Local test instructions
│   ├── app.cai            # MongoDB collections example
│   └── test.sh            # Automated test script
├── integration/           # Integration tests
│   ├── setup_test.go      # Test infrastructure and helpers
│   ├── parser_validator_test.go
│   ├── database_test.go
│   ├── api_test.go
│   └── e2e_test.go
├── cli/                   # CLI integration tests
│   └── cli_test.go
└── performance/           # Benchmark tests
    └── benchmarks_test.go
```

## Running Tests

### Quick Local Test

Test MongoDB collection parsing and validation with the local build:

```bash
cd test/local
./test.sh
```

This runs a quick smoke test to verify parsing and validation work correctly.

### Unit Tests

Run unit tests for all internal packages:

```bash
make test-unit
```

### Integration Tests

Run integration tests (requires the `integration` build tag):

```bash
make test-integration
```

### CLI Tests

Run CLI integration tests:

```bash
make test-cli
```

### All Tests

Run all tests together:

```bash
make test-all
```

### Standard Test Command

Run all tests without build tags (unit tests only):

```bash
make test
```

### With Coverage

Run tests with coverage report:

```bash
make test-coverage
```

This generates `coverage.html` which can be viewed in a browser.

## Benchmarks

### Run All Benchmarks

```bash
make bench
```

### Parser Benchmarks Only

```bash
make bench-parse
```

### Validator Benchmarks Only

```bash
make bench-validate
```

### Full Benchmark Output

```bash
go test -bench=. -benchmem -benchtime=5s ./test/performance/...
```

## Test Categories

### Unit Tests (`./internal/...`)

Unit tests are located alongside the source code in `internal/` packages:

- `internal/parser/parser_test.go` - Parser unit tests
- `internal/validator/validator_test.go` - Validator unit tests
- `internal/database/` - Database layer tests
- `internal/api/` - API handler tests

### Integration Tests (`./test/integration/...`)

Integration tests verify components working together:

1. **Parser + Validator Integration** (`parser_validator_test.go`)
   - Tests parsing DSL and validating the AST
   - Covers valid and invalid programs
   - Verifies error messages

2. **Database Integration** (`database_test.go`)
   - Full CRUD operations
   - Transactions and rollbacks
   - Concurrent access
   - Pagination

3. **API Integration** (`api_test.go`)
   - HTTP endpoint testing
   - Request/response validation
   - Error handling
   - Content type verification

4. **End-to-End Workflows** (`e2e_test.go`)
   - Complete deployment workflow
   - Config creation and updates
   - Multi-step scenarios

### CLI Tests (`./test/cli/...`)

CLI tests verify command-line interface functionality:

- Parse command workflow
- Validate command workflow
- Deploy command (dry-run)
- Config command (dry-run)
- Help output
- Error messages

### Performance Tests (`./test/performance/...`)

Benchmark tests measure performance:

- Parse performance (small/medium/large files)
- Validate performance
- Database operation performance
- Memory allocation reporting

## Writing New Tests

### Adding Integration Tests

1. Use the `integration` build tag:
   ```go
   //go:build integration
   ```

2. Use the `SetupTestSuite` helper:
   ```go
   func TestMyFeature(t *testing.T) {
       suite := SetupTestSuite(t)
       defer suite.Teardown(t)
       
       // Your test code
   }
   ```

3. Use helper methods for creating test data:
   ```go
   cfg := suite.CreateTestConfig(t, "name", "content")
   deploy := suite.CreateTestDeployment(t, "name", &cfg.ID)
   exec := suite.CreateTestExecution(t, deploy.ID, "command")
   ```

### Adding Fixture Files

1. Create `.cai` files in `test/fixtures/`
2. Valid fixtures should parse and validate successfully
3. Invalid fixtures should be prefixed with `invalid_`
4. Add comments describing the test scenario

### Adding Benchmarks

1. Add benchmarks to `test/performance/benchmarks_test.go`
2. Use `b.ResetTimer()` after setup
3. Use `b.ReportAllocs()` for memory benchmarks
4. Follow naming convention: `BenchmarkXxx`

## Test Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| `internal/parser` | ≥90% |
| `internal/validator` | ≥90% |
| `internal/database` | ≥85% |
| `internal/api` | ≥85% |
| Overall | ≥85% |

## CI/CD Integration

Tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Unit Tests
  run: make test-unit

- name: Run Integration Tests
  run: make test-integration

- name: Run Benchmarks
  run: make bench
```

## Troubleshooting

### Tests Fail with Database Errors

Integration tests use in-memory SQLite. Ensure `modernc.org/sqlite` is in go.mod:

```bash
go get modernc.org/sqlite
```

### Build Tag Issues

Integration tests require the `integration` tag. Run with:

```bash
go test -tags integration ./test/integration/...
```

### Fixture Files Not Found

CLI tests reference fixtures using relative paths. Run tests from the project root:

```bash
go test -tags integration ./test/cli/...
```
