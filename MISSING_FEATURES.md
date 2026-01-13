# Missing Features Implementation Prompt

Run this prompt to implement the missing features in CodeAI.

---

## Prompt

Implement the missing endpoint and REST API features for CodeAI. The marketing website promises functionality that doesn't exist yet. Here's what needs to be built:

### 1. Endpoint DSL Syntax

Add endpoint definitions to the parser. The syntax should be:

```
endpoint GET /users {
    description: "List all users"
    auth: optional
    query {
        page: integer, default(1)
        limit: integer, default(20)
    }
    returns: paginated(User)
}

endpoint POST /users {
    description: "Create a new user"
    auth: required
    body {
        name: string, required
        email: string, required
    }
    returns: User
}

endpoint GET /users/:id {
    description: "Get user by ID"
    auth: required
    path { id: objectid, required }
    returns: User
}

endpoint PUT /users/:id {
    description: "Update user"
    auth: required
    path { id: objectid, required }
    body {
        name: string, optional
        email: string, optional
    }
    returns: User
}

endpoint DELETE /users/:id {
    description: "Delete user"
    auth: required
    roles: [admin]
    path { id: objectid, required }
    returns: void
}
```

### 2. Implementation Tasks

#### Task 1: AST Types (internal/ast/ast.go)
Add these types:
- `EndpointDecl` - endpoint definition with method, path, auth, body, query, path params, returns
- `ParamDecl` - parameter with name, type, modifiers (required, optional, default)
- `AuthLevel` - enum: none, optional, required
- `ReturnType` - single, paginated, void

#### Task 2: Parser (internal/parser/parser.go)
- Add lexer tokens for: `endpoint`, `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `auth`, `body`, `query`, `path`, `returns`, `paginated`, `roles`
- Parse endpoint blocks with all nested structures
- Handle path parameters like `/users/:id`

#### Task 3: Validator (internal/validator/validator.go)
- Validate endpoint references existing collections/models
- Validate return types match defined schemas
- Validate auth levels are valid
- Validate roles are strings

#### Task 4: Code Generator (NEW: internal/codegen/)
Create a code generator that:
- Generates Chi router handlers from endpoint definitions
- Generates request/response structs
- Generates validation middleware
- Wires up to MongoDB/PostgreSQL repositories

#### Task 5: Runtime Server (internal/api/)
Update `server start` to:
- Parse the .cai file
- Generate handlers dynamically
- Register routes based on endpoint definitions
- Connect to database and create repositories
- Serve the API

### 3. File Structure

```
internal/
  ast/
    ast.go          # Add EndpointDecl, ParamDecl, etc.
  parser/
    parser.go       # Add endpoint parsing
    parser_test.go  # Add endpoint tests
  validator/
    validator.go    # Add endpoint validation
  codegen/          # NEW
    generator.go    # Main code generator
    handlers.go     # Generate HTTP handlers
    routes.go       # Generate router setup
    types.go        # Generate request/response types
  api/
    dynamic.go      # NEW: Dynamic route registration
```

### 4. Example .cai File That Should Work

```
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "myapp"
}

database mongodb {
    collection User {
        _id: objectid, primary, auto
        name: string, required
        email: string, required, unique
        created_at: date, auto
        updated_at: date, auto
    }
}

endpoint GET /users {
    description: "List all users"
    auth: optional
    query {
        page: integer, default(1)
        limit: integer, default(20)
    }
    returns: paginated(User)
}

endpoint POST /users {
    description: "Create a new user"
    auth: optional
    body {
        name: string, required
        email: string, required
    }
    returns: User
}

endpoint GET /users/:id {
    description: "Get user by ID"
    path { id: string, required }
    returns: User
}

endpoint DELETE /users/:id {
    description: "Delete user"
    auth: required
    path { id: string, required }
    returns: void
}
```

### 5. Expected Behavior

After implementation, running:
```bash
codeai server start
```

Should:
1. Parse app.cai
2. Connect to MongoDB
3. Generate CRUD handlers for User collection
4. Start HTTP server on :8080
5. Serve these endpoints:
   - GET /users - list users with pagination
   - POST /users - create user
   - GET /users/:id - get user by ID
   - DELETE /users/:id - delete user

### 6. Testing

Create tests in:
- `internal/parser/parser_test.go` - TestParseEndpoint*
- `internal/validator/validator_test.go` - TestValidateEndpoint*
- `test/integration/endpoint_test.go` - Full integration tests

### 7. Priority Order

1. AST types (foundation)
2. Parser (can parse the syntax)
3. Validator (ensures correctness)
4. Code generator (generates handlers)
5. Runtime integration (makes it work)

Start with Task 1 and work through sequentially. Each task should have tests before moving to the next.
