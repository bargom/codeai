# CodeAI Local Test

Complete test suite for CodeAI with MongoDB and REST API endpoints.

## Quick Start

```bash
# Fast test (parse & validate only)
./test.sh

# Full test (includes server + MongoDB + curl tests)
./test.sh --server
```

## What's Tested

✅ **DSL Parsing**
- MongoDB collection definitions
- Embedded documents and arrays
- Index definitions (unique, text, compound)
- REST API endpoint declarations
- Request/Response mappings
- Handler logic (`do` blocks)

✅ **Validation**
- Schema validation
- Field type checking
- Endpoint syntax

✅ **Server Runtime** (with --server flag)
- HTTP server startup
- MongoDB connection
- Route registration
- CRUD operations (GET, POST, PUT, DELETE)
- Response validation

## What's in app.cai

Complete application with:
- **4 MongoDB Collections**: User, UserProfile, Post, Comment
- **7 REST Endpoints**: Health check + full CRUD for users
- **Handler Logic**: validate, insert, query, update, delete, paginate

Features demonstrated:
- Field types: objectid, string, int, bool, date, arrays
- Embedded documents (address in UserProfile)
- Indexes (unique, text, compound)
- Request/Response models
- All HTTP methods (GET, POST, PUT, PATCH, DELETE)

## Manual Testing

```bash
# From project root
cd /Users/bargom/Code/github/codeai

# Parse
./bin/codeai parse test/local/app.cai

# Validate
./bin/codeai validate test/local/app.cai

# Start server
./bin/codeai server start --file test/local/app.cai --port 8080

# Test endpoints (in another terminal)
curl http://localhost:8080/health
curl http://localhost:8080/users
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","age":30}'
```

## Prerequisites

**Required:**
- Go 1.24+
- CodeAI built (`make build`)

**Optional (for --server test):**
- Docker (for MongoDB)
- curl
- jq (for pretty JSON output)

## Test Output

```bash
$ ./test.sh
✓ CodeAI binary ready
✓ Parse successful
  Found 7 endpoints
✓ Validation successful
✓ All tests passed!

$ ./test.sh --server
✓ MongoDB started
✓ Server ready
Testing Endpoints:
  GET /health ... ✓
  GET /users ... ✓
  POST /users ... ✓
✓ Server test complete!
```

## Files

- **app.cai** - Complete MongoDB + REST API application
- **test.sh** - Automated test script
- **README.md** - This file
- **STATUS.md** - Implementation status
