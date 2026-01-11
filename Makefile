# CodeAI Makefile

# Variables
BINARY_NAME=codeai
BUILD_DIR=bin
CMD_DIR=./cmd/codeai

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build test test-unit test-integration test-cli test-all lint run clean tidy bench help

all: build

## build: Build the application binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-unit: Run unit tests only (internal packages)
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage-unit.out ./internal/...
	@echo "Unit tests complete"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags integration -coverprofile=coverage-integration.out ./test/integration/...
	@echo "Integration tests complete"

## test-cli: Run CLI integration tests
test-cli:
	@echo "Running CLI tests..."
	$(GOTEST) -v -tags integration ./test/cli/...
	@echo "CLI tests complete"

## test-all: Run all tests (unit + integration + cli)
test-all: test-unit test-integration test-cli
	@echo "All tests complete"

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./test/performance/...
	@echo "Benchmarks complete"

## bench-parse: Run parser benchmarks only
bench-parse:
	@echo "Running parser benchmarks..."
	$(GOTEST) -bench=BenchmarkParse -benchmem ./test/performance/...

## bench-validate: Run validator benchmarks only
bench-validate:
	@echo "Running validator benchmarks..."
	$(GOTEST) -bench=BenchmarkValidate -benchmem ./test/performance/...

## lint: Run linters (go vet)
lint:
	@echo "Running linters..."
	$(GOVET) ./...
	@echo "Lint complete"

## run: Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy
	@echo "Tidy complete"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
