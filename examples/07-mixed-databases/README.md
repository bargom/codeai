# Example 07: Mixed Databases (PostgreSQL + MongoDB)

This example demonstrates both PostgreSQL and MongoDB syntax in CodeAI DSL, showing how each database type can be used for different data requirements.

## Overview

| Aspect | Description |
|--------|-------------|
| **Focus** | Multi-database patterns |
| **Databases** | MongoDB (active) + PostgreSQL (reference) |
| **Key Concepts** | Database selection, syntax differences, data modeling |
| **Difficulty** | Advanced |

## Important Note

The current CodeAI validator requires a single `database_type` in the config. This example demonstrates MongoDB collections with PostgreSQL syntax shown in comments for reference. In production, you would choose the appropriate database type for your use case.

## Architecture Philosophy

This example follows a common production pattern:

- **PostgreSQL** for structured, transactional data (users, deployments, API keys)
- **MongoDB** for flexible, high-volume data (logs, analytics, feature flags)

```
┌─────────────────────────────────────────────────────────────┐
│                     CodeAI Application                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  PostgreSQL (Structured)          MongoDB (Flexible)         │
│  ┌─────────────────────┐         ┌─────────────────────┐    │
│  │ Users               │         │ ActivityLog         │    │
│  │ Organizations       │────────▶│ ExecutionOutput     │    │
│  │ Deployments         │  Events │ AnalyticsEvent      │    │
│  │ Configs             │         │ FeatureFlag         │    │
│  │ ApiKeys             │         │ NotificationTemplate│    │
│  └─────────────────────┘         └─────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## What You'll Learn

- Configuring dual databases in CodeAI
- Choosing the right database for each data type
- PostgreSQL models with relationships
- MongoDB collections for flexible data
- Cross-database event handling
- API endpoints that work with both databases

## Data Models

### PostgreSQL Models (Structured Data)

| Model | Purpose | Why PostgreSQL |
|-------|---------|----------------|
| `User` | User accounts | Strong integrity, relationships |
| `Organization` | Multi-tenancy | Foreign keys, ACID transactions |
| `Deployment` | Deploy tracking | Consistent status updates |
| `Config` | Version control | Checksums, versioning |
| `ApiKey` | Access control | Security-critical, unique keys |

### MongoDB Collections (Flexible Data)

| Collection | Purpose | Why MongoDB |
|------------|---------|-------------|
| `ActivityLog` | Audit trail | High write volume, flexible payload |
| `ExecutionOutput` | Run results | Variable output structure |
| `AnalyticsEvent` | Analytics | Custom properties per event |
| `FeatureFlag` | Feature toggles | Complex targeting rules |
| `NotificationTemplate` | Templates | Multi-locale content variants |

## Quick Start

### 1. Prerequisites

```bash
# Start PostgreSQL
pg_ctl start

# Start MongoDB
mongod --dbpath /data/db

# Or with Docker Compose
docker-compose up -d
```

### 2. Review the DSL

```bash
cat mixed-databases.cai
```

### 3. Validate

```bash
codeai validate mixed-databases.cai
```

### 4. Generate and Run

```bash
codeai generate mixed-databases.cai
codeai migrate up
codeai run
```

## Configuration

### Dual Database Setup

```cai
config {
    app_name: "mixed-databases-example"

    // Primary database (PostgreSQL)
    database_type: "postgres"
    database_host: "localhost"
    database_port: 5432
    database_name: "codeai_mixed"
    database_user: "codeai"
    database_password: "codeai"

    // Secondary database (MongoDB)
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai_mixed"
}
```

## DSL Highlights

### PostgreSQL Model with Relationships

```cai
model User {
    id: uuid, primary, auto
    email: string, required, unique
    username: string, required, unique

    created_at: timestamp, auto
    updated_at: timestamp, auto

    // Relationships to other PostgreSQL tables
    has_many deployments
    has_many api_keys

    indexes {
        unique: [email]
        unique: [username]
    }
}
```

### MongoDB Collection with Flexible Payload

```cai
database mongodb {
    collection ActivityLog {
        _id: objectid, primary, auto

        user_id: string, required
        event_type: string, required

        // Flexible payload - varies by event type
        payload: object, optional

        timestamp: date, auto

        indexes {
            index: [user_id]
            index: [event_type]
            index: [timestamp]
        }
    }
}
```

### Cross-Database Events

When a PostgreSQL record changes, log to MongoDB:

```cai
events {
    // Deployment created in PostgreSQL → log to MongoDB
    on deployment.created {
        action: log_activity
        params: {
            event_type: "deployment.created"
            resource_type: "deployment"
        }
    }

    // User login → update PostgreSQL, log to MongoDB
    on user.login {
        action: log_activity
        action: track_analytics
        params: {
            event_name: "user_login"
            event_category: "authentication"
        }
    }
}
```

### API Endpoints for Both Databases

```cai
api {
    prefix: "/api/v1"

    // PostgreSQL resources
    resource users {
        model: User
        endpoints {
            list: GET /users
            get: GET /users/{id}
            create: POST /users
            update: PUT /users/{id}
            delete: DELETE /users/{id}
        }
    }

    // MongoDB resources
    resource activity {
        collection: ActivityLog
        endpoints {
            list: GET /activity
            get: GET /activity/{id}
            create: POST /activity
        }
    }
}
```

## When to Use Each Database

### Use PostgreSQL For

| Data Type | Reason |
|-----------|--------|
| User accounts | ACID transactions, password security |
| Financial data | Decimal precision, referential integrity |
| Multi-tenant data | Foreign key relationships |
| Configuration | Version tracking, checksums |
| API keys | Unique constraints, secure storage |

### Use MongoDB For

| Data Type | Reason |
|-----------|--------|
| Activity logs | High write throughput |
| Analytics events | Variable properties |
| Feature flags | Complex nested rules |
| Notifications | Multi-locale content |
| Execution outputs | Variable result structures |

## Testing the API

### Create a User (PostgreSQL)

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "username": "admin",
    "password_hash": "hashed_password"
  }'
```

### Log Activity (MongoDB)

```bash
curl -X POST http://localhost:8080/api/v1/activity \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid-here",
    "event_type": "page_view",
    "payload": {
      "page": "/dashboard",
      "duration_ms": 1500
    }
  }'
```

### Query Activity by User

```bash
curl "http://localhost:8080/api/v1/activity?user_id=user-uuid-here"
```

### Create a Feature Flag (MongoDB)

```bash
curl -X POST http://localhost:8080/api/v1/features \
  -H "Content-Type: application/json" \
  -d '{
    "key": "new_dashboard",
    "name": "New Dashboard",
    "flag_type": "boolean",
    "default_value": false,
    "rollout_percentage": 10,
    "environments": {
      "development": { "enabled": true },
      "production": { "enabled": false }
    }
  }'
```

## Environment Variables

```bash
# PostgreSQL
export CODEAI_DB_HOST=localhost
export CODEAI_DB_PORT=5432
export CODEAI_DB_NAME=codeai_mixed
export CODEAI_DB_USER=codeai
export CODEAI_DB_PASSWORD=secret

# MongoDB
export CODEAI_MONGODB_URI=mongodb://localhost:27017
export CODEAI_MONGODB_DATABASE=codeai_mixed
```

## Docker Compose Setup

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: codeai_mixed
      POSTGRES_USER: codeai
      POSTGRES_PASSWORD: codeai
    ports:
      - "5432:5432"

  mongodb:
    image: mongo:6
    ports:
      - "27017:27017"

  app:
    build: .
    depends_on:
      - postgres
      - mongodb
    environment:
      CODEAI_DB_HOST: postgres
      CODEAI_DB_NAME: codeai_mixed
      CODEAI_DB_USER: codeai
      CODEAI_DB_PASSWORD: codeai
      CODEAI_MONGODB_URI: mongodb://mongodb:27017
      CODEAI_MONGODB_DATABASE: codeai_mixed
    ports:
      - "8080:8080"
```

## Common Patterns

### Cross-Database References

MongoDB collections can reference PostgreSQL records by ID:

```cai
collection ActivityLog {
    user_id: string, required  // References PostgreSQL User.id
    organization_id: string, optional  // References PostgreSQL Organization.id
}
```

### Denormalization Strategy

Store frequently accessed data in both places:

```cai
// PostgreSQL: source of truth
model User {
    id: uuid, primary, auto
    email: string, required, unique
}

// MongoDB: denormalized for queries
collection ActivityLog {
    user_id: string, required
    user_email: string, optional  // Denormalized for display
}
```

## Related Resources

- [Database Guide](../../docs/03-database.md) - Full database documentation
- [MongoDB Collections Example](../06-mongodb-collections/) - MongoDB-only patterns
- [DSL Cheatsheet](../../docs/dsl_cheatsheet.md) - Quick syntax reference
- [Troubleshooting](../../docs/troubleshooting.md#mongodb-issues) - Common issues
