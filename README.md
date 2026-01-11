# CodeAI

A Go-based runtime for parsing and executing AI agent specifications written in Markdown.

## Overview

CodeAI provides a structured approach to defining and running AI agents using a Markdown-based domain-specific language.

## Project Structure

```
codeai/
├── cmd/codeai/     # Application entry point
├── internal/       # Private application code
│   ├── parser/     # Markdown to AST parser
│   ├── validator/  # AST validation logic
│   ├── engine/     # Core execution engine
│   └── llm/        # LLM API client interface
├── pkg/types/      # Shared type definitions
├── test/fixtures/  # Test fixtures and data
└── docs/           # Documentation
```

## Requirements

- Go 1.23 or later

## Installation

```bash
go build ./cmd/codeai
```

## Usage

TODO: Add usage instructions

## License

See LICENSE file for details.
