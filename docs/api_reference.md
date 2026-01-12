# CodeAI API Reference

This document provides comprehensive documentation for the CodeAI HTTP REST API endpoints.

## Table of Contents

- [API Overview](#api-overview)
- [Authentication](#authentication)
- [Common Headers](#common-headers)
- [Error Handling](#error-handling)
- [Pagination and Filtering](#pagination-and-filtering)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Deployments](#deployments)
  - [Configs](#configs)
  - [Executions](#executions)
  - [Workflows](#workflows)
  - [Jobs](#jobs)
  - [Events](#events)
  - [Webhooks](#webhooks)
- [OpenAPI Specification](#openapi-specification)
- [Code Generation](#code-generation)

---

## API Overview

### Base URL Structure

The CodeAI API follows RESTful conventions. The base URL structure is:

```
http://<host>:<port>/<resource>
```

Default server configuration:
- **Development**: `http://localhost:8080`
- **Production**: Configure via environment variable `CODEAI_API_URL`

### Server Configuration

The HTTP server is configured with the following timeouts:

| Setting | Value | Description |
|---------|-------|-------------|
| Read Timeout | 15 seconds | Maximum time to read the entire request |
| Write Timeout | 15 seconds | Maximum time to write the response |
| Idle Timeout | 60 seconds | Maximum time to keep idle connections |
| Request Timeout | 60 seconds | Maximum time for the entire request |

### API Version

The current API version is **v1**. All endpoints are accessible without version prefix, though `/api/v1/` prefix is supported for workflow-related endpoints.

---

## Authentication

### JWT Token Authentication

CodeAI uses JSON Web Tokens (JWT) for API authentication. The authentication scheme uses HTTP Bearer tokens.

#### Obtaining a Token

Tokens are obtained through your identity provider or the authentication service configured for your CodeAI deployment.

#### Using the Token

Include the JWT token in the `Authorization` header:

```http
Authorization: Bearer <your-jwt-token>
```

#### Token Format

Tokens should follow the standard JWT format:
- **Header**: Algorithm and token type
- **Payload**: Claims including user ID, roles, and expiration
- **Signature**: Verification signature

#### Token Refresh

When a token is about to expire, obtain a new token from your authentication provider before the current token expires. The API does not provide a refresh endpoint directly.

#### Security Schemes

The OpenAPI specification supports these security schemes:

| Scheme | Type | Description |
|--------|------|-------------|
| Bearer Auth | HTTP Bearer | JWT token in Authorization header |
| API Key | API Key | X-API-Key header authentication |

---

## Common Headers

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes (for POST/PUT) | Must be `application/json` |
| `Authorization` | Depends on endpoint | Bearer token for authenticated endpoints |
| `X-Request-ID` | No | Client-provided request ID for tracing |

### Response Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | Always `application/json` |
| `X-Request-ID` | Request identifier (auto-generated if not provided) |

---

## Error Handling

### Error Response Format

All errors return a consistent JSON structure:

```json
{
  "error": "Human-readable error message",
  "details": {
    "field_name": "Validation error for this field"
  }
}
```

### HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request succeeded |
| 201 | Created | Resource created successfully |
| 202 | Accepted | Request accepted for processing |
| 204 | No Content | Request succeeded, no content to return |
| 400 | Bad Request | Invalid request syntax or validation error |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 500 | Internal Server Error | Server-side error |
| 503 | Service Unavailable | Service temporarily unavailable |

### Validation Errors

Validation errors return HTTP 400 with detailed field-level information:

```json
{
  "error": "validation failed",
  "details": {
    "name": "is required",
    "config_id": "must be a valid UUID"
  }
}
```

Common validation messages:
- `is required` - Field must be provided
- `must be at least X characters` - Minimum length requirement
- `must be at most X characters` - Maximum length requirement
- `must be a valid UUID` - UUID format required
- `must be one of: value1 value2` - Must match allowed values

---

## Pagination and Filtering

### Pagination Parameters

All list endpoints support pagination via query parameters:

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `limit` | integer | 20 | 100 | Number of items per page |
| `offset` | integer | 0 | - | Number of items to skip |

### Paginated Response Format

```json
{
  "data": [...],
  "limit": 20,
  "offset": 0,
  "total": 150
}
```

### Example

```http
GET /deployments?limit=10&offset=20
```

Returns items 21-30 from the deployments list.

---

## Endpoints

### Health Check

Health check endpoints for monitoring and Kubernetes probes.

#### GET /health

Returns overall system health status.

**Response**

```json
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Connected"
    },
    "temporal": {
      "status": "healthy",
      "message": "Connected"
    }
  }
}
```

| Status Code | Description |
|-------------|-------------|
| 200 | System is healthy |
| 503 | System is unhealthy |

#### GET /health/live

Kubernetes liveness probe endpoint.

**Response**: Returns 200 unless the process is broken.

#### GET /health/ready

Kubernetes readiness probe endpoint.

**Response**: Returns 503 if critical dependencies are unavailable.

---

### Deployments

Manage CodeAI deployments.

#### POST /deployments

Create a new deployment.

**Request Body**

```json
{
  "name": "my-deployment",
  "config_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Deployment name (1-255 characters) |
| `config_id` | string | No | UUID of associated config |

**Response** (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "my-deployment",
  "config_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### GET /deployments

List all deployments with pagination.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Items per page (default: 20, max: 100) |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "my-deployment",
      "config_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "running",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:35:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

#### GET /deployments/{id}

Get a specific deployment by ID.

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Deployment UUID |

**Response** (200 OK)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "my-deployment",
  "config_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

#### PUT /deployments/{id}

Update an existing deployment.

**Request Body**

```json
{
  "name": "updated-deployment",
  "config_id": "550e8400-e29b-41d4-a716-446655440002",
  "status": "running"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | New deployment name |
| `config_id` | string | No | New config UUID |
| `status` | string | No | New status (pending, running, stopped, failed, complete) |

**Response** (200 OK): Updated deployment object

#### DELETE /deployments/{id}

Delete a deployment.

**Response** (204 No Content)

#### POST /deployments/{id}/execute

Execute a deployment.

**Request Body** (optional)

```json
{
  "variables": {
    "env": "production",
    "debug": false
  }
}
```

**Response** (202 Accepted)

```json
{
  "id": "exec-550e8400-e29b-41d4-a716-446655440001",
  "deployment_id": "550e8400-e29b-41d4-a716-446655440001",
  "command": "execute",
  "started_at": "2024-01-15T10:40:00Z"
}
```

#### GET /deployments/{id}/executions

List executions for a specific deployment.

**Response** (200 OK): Paginated list of executions

---

### Configs

Manage CodeAI DSL configurations.

#### POST /configs

Create a new configuration.

**Request Body**

```json
{
  "name": "my-config",
  "content": "workflow MyWorkflow {\n  step1: Agent(\"gpt-4\")\n}"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Config name (1-255 characters) |
| `content` | string | Yes | CodeAI DSL content |

**Response** (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440010",
  "name": "my-config",
  "content": "workflow MyWorkflow {\n  step1: Agent(\"gpt-4\")\n}",
  "ast_json": {...},
  "validation_errors": null,
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### GET /configs

List all configurations with pagination.

**Response** (200 OK): Paginated list of configs

#### GET /configs/{id}

Get a specific configuration.

**Response** (200 OK): Config object

#### PUT /configs/{id}

Update a configuration.

**Request Body**

```json
{
  "name": "updated-config",
  "content": "workflow UpdatedWorkflow {\n  step1: Agent(\"gpt-4-turbo\")\n}"
}
```

**Response** (200 OK): Updated config object

#### DELETE /configs/{id}

Delete a configuration.

**Response** (204 No Content)

#### POST /configs/{id}/validate

Validate a configuration's DSL content.

**Request Body** (optional)

```json
{
  "content": "workflow TestWorkflow { ... }"
}
```

If no content is provided, validates the stored config content.

**Response** (200 OK)

```json
{
  "valid": true,
  "errors": []
}
```

Or with errors:

```json
{
  "valid": false,
  "errors": [
    "parse error: unexpected token at line 5",
    "validation error: undefined agent type"
  ]
}
```

---

### Executions

View execution history and status.

#### GET /executions

List all executions with pagination.

**Response** (200 OK)

```json
{
  "data": [
    {
      "id": "exec-001",
      "deployment_id": "deploy-001",
      "command": "execute",
      "output": "Completed successfully",
      "exit_code": 0,
      "started_at": "2024-01-15T10:40:00Z",
      "completed_at": "2024-01-15T10:45:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

#### GET /executions/{id}

Get a specific execution.

**Response** (200 OK)

```json
{
  "id": "exec-001",
  "deployment_id": "deploy-001",
  "command": "execute",
  "output": "Completed successfully",
  "exit_code": 0,
  "started_at": "2024-01-15T10:40:00Z",
  "completed_at": "2024-01-15T10:45:00Z"
}
```

---

### Workflows

Manage Temporal workflow executions.

#### POST /workflows

Start a new workflow.

**Request Body**

```json
{
  "workflowId": "wf-unique-id-001",
  "workflowType": "ai-pipeline",
  "input": {
    "model": "gpt-4",
    "prompt": "Analyze this data",
    "maxTokens": 1000
  },
  "metadata": {
    "project": "my-project",
    "environment": "production"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflowId` | string | Yes | Unique workflow identifier |
| `workflowType` | string | Yes | Type: `ai-pipeline` or `test-suite` |
| `input` | object | Yes | Workflow-specific input data |
| `metadata` | object | No | Custom metadata key-value pairs |

**Response** (202 Accepted)

```json
{
  "workflowId": "wf-unique-id-001",
  "runId": "run-abc123",
  "status": "running"
}
```

#### GET /workflows

List workflows with filtering.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `workflowType` | string | Filter by workflow type |
| `status` | string | Filter by status (pending, running, completed, failed, canceled) |
| `limit` | integer | Items per page |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "workflows": [
    {
      "workflowId": "wf-001",
      "workflowType": "ai-pipeline",
      "runId": "run-abc",
      "status": "completed",
      "startedAt": "2024-01-15T10:00:00Z",
      "completedAt": "2024-01-15T10:05:00Z"
    }
  ],
  "total": 50,
  "limit": 20,
  "offset": 0
}
```

#### GET /workflows/{id}

Get workflow status.

**Response** (200 OK)

```json
{
  "workflowId": "wf-001",
  "workflowType": "ai-pipeline",
  "runId": "run-abc",
  "status": "completed",
  "input": {...},
  "output": {...},
  "startedAt": "2024-01-15T10:00:00Z",
  "completedAt": "2024-01-15T10:05:00Z",
  "metadata": {
    "project": "my-project"
  }
}
```

#### POST /workflows/{id}/cancel

Cancel a running workflow.

**Request Body** (optional)

```json
{
  "reason": "User requested cancellation"
}
```

**Response** (200 OK)

```json
{
  "workflowId": "wf-001",
  "status": "canceled"
}
```

#### GET /workflows/{id}/history

Get workflow execution history.

**Response** (200 OK)

```json
{
  "workflowId": "wf-001",
  "runId": "run-abc",
  "events": [
    {
      "eventId": 1,
      "eventType": "WorkflowExecutionStarted",
      "timestamp": "2024-01-15T10:00:00Z"
    },
    {
      "eventId": 2,
      "eventType": "ActivityTaskScheduled",
      "timestamp": "2024-01-15T10:00:01Z"
    }
  ]
}
```

---

### Jobs

Manage scheduled and recurring jobs.

#### POST /jobs

Submit a job for immediate processing.

**Request Body**

```json
{
  "task_type": "data-processing",
  "payload": {
    "source": "s3://bucket/data.csv",
    "operations": ["clean", "transform"]
  },
  "queue": "high-priority",
  "max_retries": 3,
  "timeout": "30m",
  "metadata": {
    "project": "analytics"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_type` | string | Yes | Type of task to execute |
| `payload` | any | No | Task-specific payload |
| `queue` | string | No | Target queue name |
| `max_retries` | integer | No | Maximum retry attempts |
| `timeout` | string | No | Timeout duration (e.g., "5m", "1h") |
| `metadata` | object | No | Custom metadata |

**Response** (201 Created)

```json
{
  "id": "job-550e8400-e29b-41d4-a716-446655440001"
}
```

#### POST /jobs/schedule

Schedule a job for future execution.

**Request Body**

```json
{
  "task_type": "report-generation",
  "payload": {...},
  "schedule_at": "2024-01-20T09:00:00Z"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schedule_at` | string | Yes | RFC3339 timestamp for execution |
| (other fields) | - | - | Same as POST /jobs |

**Response** (201 Created): Job ID

#### POST /jobs/recurring

Create a recurring job with cron schedule.

**Request Body**

```json
{
  "task_type": "daily-cleanup",
  "payload": {...},
  "cron_spec": "0 2 * * *"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `cron_spec` | string | Yes | Cron expression (e.g., "0 2 * * *" for daily at 2 AM) |
| (other fields) | - | - | Same as POST /jobs |

**Response** (201 Created): Job ID

#### GET /jobs

List jobs with filtering.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status (pending, running, completed, failed) |
| `task_type` | string | Filter by task type |
| `queue` | string | Filter by queue |
| `limit` | integer | Items per page |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "jobs": [...],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

#### GET /jobs/{id}

Get job status.

**Response** (200 OK): Job object with status

#### DELETE /jobs/{id}

Cancel a job.

**Response** (204 No Content)

#### GET /jobs/stats

Get queue statistics.

**Response** (200 OK)

```json
{
  "queues": {
    "default": {
      "pending": 10,
      "running": 5,
      "completed": 1000,
      "failed": 3
    },
    "high-priority": {
      "pending": 2,
      "running": 3,
      "completed": 500,
      "failed": 1
    }
  }
}
```

---

### Events

Query system events and statistics.

#### GET /events

List events with filtering.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Filter by event type |
| `source` | string | Filter by event source |
| `start_time` | string | Filter events after this time (RFC3339) |
| `end_time` | string | Filter events before this time (RFC3339) |
| `limit` | integer | Items per page |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "data": [
    {
      "id": "evt-001",
      "type": "workflow.started",
      "source": "workflow-engine",
      "timestamp": "2024-01-15T10:00:00Z",
      "data": {
        "workflowId": "wf-001"
      },
      "metadata": {
        "environment": "production"
      }
    }
  ],
  "limit": 20,
  "offset": 0
}
```

#### GET /events/{id}

Get a specific event.

**Response** (200 OK): Event object

#### GET /events/types

List all available event types.

**Response** (200 OK)

```json
[
  {
    "type": "workflow.started",
    "description": "Workflow execution has started",
    "category": "workflow"
  },
  {
    "type": "workflow.completed",
    "description": "Workflow execution completed successfully",
    "category": "workflow"
  },
  {
    "type": "workflow.failed",
    "description": "Workflow execution failed",
    "category": "workflow"
  },
  {
    "type": "job.enqueued",
    "description": "Job has been added to the queue",
    "category": "job"
  },
  {
    "type": "job.started",
    "description": "Job processing has started",
    "category": "job"
  },
  {
    "type": "job.completed",
    "description": "Job completed successfully",
    "category": "job"
  },
  {
    "type": "job.failed",
    "description": "Job processing failed",
    "category": "job"
  },
  {
    "type": "agent.executed",
    "description": "Agent has executed an action",
    "category": "agent"
  },
  {
    "type": "test.suite.completed",
    "description": "Test suite execution completed",
    "category": "test"
  },
  {
    "type": "webhook.triggered",
    "description": "Webhook was triggered",
    "category": "webhook"
  },
  {
    "type": "email.sent",
    "description": "Email was sent",
    "category": "notification"
  }
]
```

#### GET /events/stats

Get event statistics.

**Response** (200 OK)

```json
{
  "total_events": 15000,
  "by_type": {
    "workflow.started": 500,
    "workflow.completed": 480,
    "workflow.failed": 20
  },
  "by_source": {
    "workflow-engine": 1000,
    "job-scheduler": 500
  }
}
```

---

### Webhooks

Manage webhook subscriptions for event notifications.

#### POST /webhooks

Create a new webhook subscription.

**Request Body**

```json
{
  "url": "https://your-server.com/webhook",
  "events": ["workflow.completed", "workflow.failed"],
  "secret": "your-webhook-secret",
  "headers": {
    "X-Custom-Header": "custom-value"
  },
  "metadata": {
    "project": "my-project"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Webhook endpoint URL (must be valid URL) |
| `events` | array | No | Event types to subscribe (empty = all events) |
| `secret` | string | No | Secret for HMAC signature verification |
| `headers` | object | No | Custom headers to include in webhook requests |
| `metadata` | object | No | Custom metadata |

**Response** (201 Created)

```json
{
  "id": "wh-550e8400-e29b-41d4-a716-446655440001"
}
```

#### GET /webhooks

List all webhooks.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `active` | boolean | Filter by active status |
| `limit` | integer | Items per page |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "webhooks": [
    {
      "id": "wh-001",
      "url": "https://your-server.com/webhook",
      "events": ["workflow.completed"],
      "headers": {},
      "active": true,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z",
      "last_delivery": "2024-01-15T12:00:00Z",
      "failure_count": 0
    }
  ],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

#### GET /webhooks/{id}

Get a specific webhook.

**Response** (200 OK): Webhook object

#### PUT /webhooks/{id}

Update a webhook.

**Request Body**

```json
{
  "url": "https://new-server.com/webhook",
  "events": ["workflow.completed", "job.failed"],
  "active": true
}
```

All fields are optional.

**Response** (200 OK): Updated webhook object

#### DELETE /webhooks/{id}

Delete a webhook.

**Response** (204 No Content)

#### POST /webhooks/{id}/test

Send a test webhook to verify configuration.

**Response** (200 OK)

```json
{
  "id": "delivery-001",
  "webhook_id": "wh-001",
  "event_id": "test-event",
  "event_type": "webhook.test",
  "url": "https://your-server.com/webhook",
  "status_code": 200,
  "duration_ms": 150,
  "attempts": 1,
  "success": true,
  "delivered_at": "2024-01-15T12:00:00Z"
}
```

#### GET /webhooks/{id}/deliveries

List delivery history for a webhook.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `success` | boolean | Filter by delivery success |
| `limit` | integer | Items per page |
| `offset` | integer | Items to skip |

**Response** (200 OK)

```json
{
  "deliveries": [
    {
      "id": "delivery-001",
      "webhook_id": "wh-001",
      "event_id": "evt-001",
      "event_type": "workflow.completed",
      "url": "https://your-server.com/webhook",
      "status_code": 200,
      "duration_ms": 150,
      "attempts": 1,
      "success": true,
      "delivered_at": "2024-01-15T12:00:00Z"
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

#### POST /webhooks/deliveries/{id}/retry

Retry a failed webhook delivery.

**Response** (200 OK)

```json
{
  "status": "retry initiated"
}
```

---

## OpenAPI Specification

CodeAI can generate OpenAPI 3.0 specifications from its DSL definitions.

### Accessing the OpenAPI Spec

The OpenAPI specification can be generated programmatically using the OpenAPI generator:

```go
import "github.com/bargom/codeai/internal/openapi"

// Create generator with custom config
config := &openapi.Config{
    Title:       "My API",
    Version:     "1.0.0",
    Description: "API generated from CodeAI DSL",
    Servers: []openapi.ServerConfig{
        {URL: "https://api.example.com", Description: "Production"},
    },
}

gen := openapi.NewGenerator(config)

// Generate from CodeAI DSL file
spec, err := gen.GenerateFromFile("my-api.cai")

// Write to file
err = gen.WriteToFile("openapi.yaml", spec)
```

### Output Formats

The generator supports two output formats:

| Format | Extension | Usage |
|--------|-----------|-------|
| YAML | `.yaml`, `.yml` | Human-readable, good for version control |
| JSON | `.json` | Machine-readable, good for tooling |

### Using with Swagger UI

1. Generate the OpenAPI spec file
2. Serve the spec file at a known URL
3. Configure Swagger UI to load from that URL

Example with Docker:

```bash
docker run -p 8081:8080 -e SWAGGER_JSON=/spec/openapi.yaml \
  -v $(pwd)/openapi.yaml:/spec/openapi.yaml swaggerapi/swagger-ui
```

### Using with Postman

1. Generate the OpenAPI spec (JSON or YAML)
2. In Postman, go to File > Import
3. Select the generated spec file
4. Postman will create a collection with all endpoints

### CodeAI-Specific Extensions

The OpenAPI specification includes CodeAI-specific extensions:

| Extension | Description |
|-----------|-------------|
| `x-codeai-handler` | Internal handler reference |
| `x-codeai-middleware` | Applied middleware list |
| `x-codeai-source` | Source DSL reference |

---

## Code Generation

### Generating Client SDKs

Use the OpenAPI specification with standard code generators:

```bash
# Generate TypeScript client
npx @openapitools/openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-axios \
  -o ./generated/ts-client

# Generate Go client
openapi-generator generate \
  -i openapi.yaml \
  -g go \
  -o ./generated/go-client

# Generate Python client
openapi-generator generate \
  -i openapi.yaml \
  -g python \
  -o ./generated/python-client
```

### Generating Server Stubs

```bash
# Generate Go server
openapi-generator generate \
  -i openapi.yaml \
  -g go-server \
  -o ./generated/go-server
```

---

## Rate Limiting

The API does not currently implement rate limiting. For production deployments, consider adding rate limiting at the load balancer or API gateway level.

---

## Changelog

### v1.0.0
- Initial API release
- Deployments, Configs, and Executions endpoints
- Workflow management with Temporal integration
- Job scheduling with cron support
- Event streaming and webhook notifications
- OpenAPI 3.0 specification generation
