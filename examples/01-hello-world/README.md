# Hello World API Example

A simple example demonstrating basic CodeAI DSL features with a Task/Todo API.

## Overview

This example shows how to:
- Define a basic entity with common field types
- Create REST endpoints for CRUD operations
- Use field modifiers (required, optional, default)
- Set up basic authentication
- Emit events on data changes

## File Structure

```
01-hello-world/
├── hello-world.cai    # Main DSL file
├── README.md             # This file
└── test.sh               # Sample curl commands for testing
```

## Entity: Task

The `Task` entity represents a simple todo item:

| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Primary key (auto-generated) |
| `title` | string | Task title (required, searchable) |
| `description` | text | Detailed description (optional) |
| `status` | enum | pending, in_progress, completed |
| `priority` | integer | Priority level (default: 1) |
| `due_date` | date | Optional due date |
| `completed` | boolean | Completion flag |
| `completed_at` | timestamp | When completed |
| `created_at` | timestamp | Auto-set on creation |
| `updated_at` | timestamp | Auto-updated on changes |

## Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/tasks` | List tasks with filtering | Optional |
| GET | `/tasks/{id}` | Get task by ID | Optional |
| POST | `/tasks` | Create new task | Required |
| PUT | `/tasks/{id}` | Update task | Required |
| DELETE | `/tasks/{id}` | Delete task | Required |
| POST | `/tasks/{id}/complete` | Mark complete | Required |

## Step-by-Step Instructions

### 1. Generate the API

```bash
# From the project root
codeai generate examples/01-hello-world/hello-world.cai

# This generates:
# - Database migrations
# - Go models and handlers
# - OpenAPI specification
# - Tests
```

### 2. Set Environment Variables

```bash
export DATABASE_URL="postgres://localhost:5432/hello_world"
export JWT_ISSUER="hello-world-api"
export JWT_SECRET="your-secret-key-here"
```

### 3. Run Migrations

```bash
codeai migrate up
```

### 4. Start the Server

```bash
codeai run
# Server starts on http://localhost:8080
```

### 5. Test the API

See the **Sample Requests** section below.

## Sample Requests

### Create a Task

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "title": "Learn CodeAI",
    "description": "Complete the hello world tutorial",
    "priority": 1,
    "due_date": "2026-01-20"
  }'
```

**Expected Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Learn CodeAI",
  "description": "Complete the hello world tutorial",
  "status": "pending",
  "priority": 1,
  "due_date": "2026-01-20",
  "completed": false,
  "completed_at": null,
  "created_at": "2026-01-12T10:00:00Z",
  "updated_at": "2026-01-12T10:00:00Z"
}
```

### List Tasks

```bash
# List all tasks
curl http://localhost:8080/tasks

# Filter by status
curl "http://localhost:8080/tasks?status=pending"

# With pagination
curl "http://localhost:8080/tasks?page=1&limit=10"

# Search by title
curl "http://localhost:8080/tasks?search=CodeAI"
```

**Expected Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Learn CodeAI",
      "status": "pending",
      "priority": 1,
      "created_at": "2026-01-12T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### Get a Single Task

```bash
curl http://localhost:8080/tasks/550e8400-e29b-41d4-a716-446655440000
```

### Update a Task

```bash
curl -X PUT http://localhost:8080/tasks/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "status": "in_progress",
    "priority": 2
  }'
```

### Mark Task Complete

```bash
curl -X POST http://localhost:8080/tasks/550e8400-e29b-41d4-a716-446655440000/complete \
  -H "Authorization: Bearer <token>"
```

**Expected Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Learn CodeAI",
  "status": "completed",
  "completed": true,
  "completed_at": "2026-01-12T11:00:00Z"
}
```

### Delete a Task

```bash
curl -X DELETE http://localhost:8080/tasks/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

## Events

The API emits events on data changes:

| Event | Trigger | Payload |
|-------|---------|---------|
| `TaskCreated` | POST /tasks | task_id, title, created_at |
| `TaskUpdated` | PUT /tasks/{id} | task_id, changes, updated_at |
| `TaskDeleted` | DELETE /tasks/{id} | task_id, deleted_at |
| `TaskCompleted` | POST /tasks/{id}/complete | task_id, completed_at |

Subscribe to these events for webhooks or integrations.

## Key Concepts Demonstrated

1. **Entity Definition**: Basic entity with various field types
2. **Field Modifiers**: `required`, `optional`, `default`, `auto`, `searchable`
3. **Enum Type**: `status: enum(pending, in_progress, completed)`
4. **Indexes**: For query optimization
5. **REST Endpoints**: Full CRUD with path and query parameters
6. **Authentication**: JWT-based auth with `required`/`optional`
7. **Events**: Event emission on data changes
8. **Configuration**: Database and auth configuration

## Next Steps

- Try the [Blog API](../02-blog-api/) example for entity relationships
- See [E-commerce](../03-ecommerce/) for workflows and sagas
- Check [Integrations](../04-integrations/) for external API patterns
