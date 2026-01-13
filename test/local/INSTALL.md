# Installation and Testing Guide

## Quick Start

### 1. Build CodeAI

```bash
cd /Users/bargom/Code/github/codeai
make build
```

Binary will be at: `./bin/codeai`

### 2. Start MongoDB

```bash
# Using Docker (recommended)
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Verify it's running
docker ps | grep mongodb
```

### 3. Test the Application

```bash
# Parse the .cai file
./bin/codeai parse test/local/app.cai

# Validate the .cai file
./bin/codeai validate test/local/app.cai
```

Both commands should succeed without errors.

### 4. Run Automated Tests (Optional)

```bash
cd test/local
./test.sh
```

This will:
- Verify prerequisites
- Parse and validate the .cai file
- Start the server
- Test all API endpoints
- Clean up

## What's Included

### Files Created

- `test/local/app.cai` - Complete test application with MongoDB + endpoints
- `test/local/README.md` - Detailed testing documentation
- `test/local/test.sh` - Automated test script
- `test/local/.gitignore` - Ignore test artifacts

### Features Demonstrated

**Database:**
- MongoDB configuration
- Collection with indexes
- Field modifiers (required, unique, optional, default, auto)
- ObjectID primary key

**API Endpoints (7 total):**
1. `POST /users` - Create user
2. `GET /users/:id` - Get user by ID
3. `GET /users` - Get all users (paginated)
4. `PUT /users/:id` - Update user
5. `PATCH /users/:id` - Partial update
6. `DELETE /users/:id` - Delete user
7. `GET /health` - Health check

**Request/Response Models:**
- CreateUserRequest
- UpdateUserRequest
- SearchParams
- UserList
- Empty
- HealthStatus

## Verification Checklist

✅ CodeAI builds successfully (`make build`)
✅ Parse command works (`./bin/codeai parse test/local/app.cai`)
✅ Validate command works (`./bin/codeai validate test/local/app.cai`)
✅ JSON AST output is well-formed
✅ All 7 endpoints are parsed correctly
✅ MongoDB collections are parsed correctly

## Marketing Website Updated

The codeai-marketing-web has been updated with:
- New "API Endpoints" tab in the examples section
- Complete working example matching `test/local/app.cai`
- Updated curl examples with all CRUD operations

Location: `/Users/bargom/Code/binarydata/ai/codeai-marketing-web/index.html`

## Next Steps

1. **Start the server** (when server command is implemented):
   ```bash
   ./bin/codeai server start --config test/local/app.cai
   ```

2. **Test with curl**:
   ```bash
   # Create user
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"name":"John Doe","email":"john@example.com","age":30}'
   
   # Get users
   curl http://localhost:8080/users
   ```

3. **View OpenAPI docs** (when implemented):
   ```bash
   open http://localhost:8080/docs
   ```

## Troubleshooting

**Parse Error:**
- Check syntax in app.cai
- Ensure proper indentation
- Verify field modifiers

**MongoDB Connection:**
- Ensure MongoDB is running: `docker ps`
- Check port 27017 is available: `lsof -i :27017`
- Restart container: `docker restart mongodb`

**Build Error:**
- Check Go version: `go version` (need 1.24+)
- Clean and rebuild: `make clean && make build`
