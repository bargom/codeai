# CodeAI Examples

This directory contains practical example projects demonstrating CodeAI DSL features and patterns.

## Examples Overview

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | [Hello World](./01-hello-world/) | Simple Task API | Basic entities, CRUD endpoints, events |
| 02 | [Blog API](./02-blog-api/) | Multi-entity blog platform | Relationships, RBAC, soft delete, search |
| 03 | [E-commerce](./03-ecommerce/) | Order processing backend | Workflows, saga pattern, integrations |
| 04 | [Integrations](./04-integrations/) | External API patterns | Circuit breaker, retry, webhooks, GraphQL |
| 05 | [Scheduled Jobs](./05-scheduled-jobs/) | Background job processing | Cron jobs, queues, retries, monitoring |
| 06 | [MongoDB Collections](./06-mongodb-collections/) | MongoDB document modeling | Collections, embedded docs, arrays, indexes |
| 07 | [Mixed Databases](./07-mixed-databases/) | PostgreSQL + MongoDB together | Dual databases, cross-database events |

## Quick Start

Each example is self-contained and can be run independently.

### 1. Choose an Example

```bash
cd examples/01-hello-world
```

### 2. Review the DSL File

```bash
cat hello-world.cai
```

### 3. Generate the API

```bash
codeai generate hello-world.cai
```

### 4. Run Migrations

```bash
codeai migrate up
```

### 5. Start the Server

```bash
codeai run
```

### 6. Test the API

```bash
./test.sh
```

## Example Structure

Each example directory contains:

```
examples/XX-name/
├── name.cai      # Main DSL file
├── README.md        # Documentation with explanations
└── test.sh          # Curl commands for testing
```

## Progression Path

We recommend exploring the examples in order:

### 1. Hello World (Beginner)

Start here to understand basic concepts:
- Entity definition with field types and modifiers
- REST endpoints for CRUD operations
- Simple events for data changes

**What you'll learn:**
- `entity` declarations
- Field types: `uuid`, `string`, `text`, `boolean`, `integer`, `enum`
- Field modifiers: `required`, `optional`, `default`, `auto`, `searchable`
- `endpoint` declarations with `auth` and `returns`

### 2. Blog API (Intermediate)

Build on basics with multiple entities:
- Foreign key relationships with `ref(Entity)`
- Role-based access control (RBAC)
- Soft delete for data preservation
- Full-text search

**What you'll learn:**
- Entity relationships
- `roles` directive for authorization
- `soft_delete` modifier
- Self-referencing entities (comments, categories)

### 3. E-commerce (Advanced)

Full-featured backend with workflows:
- Order processing workflow with saga pattern
- Payment integration with circuit breaker
- Inventory management with reservations
- Event-driven architecture

**What you'll learn:**
- `workflow` declarations
- `integration` definitions
- Saga pattern with compensation
- Complex business logic

### 4. Integrations (Specialist)

Deep dive into external service patterns:
- REST and GraphQL integrations
- Circuit breaker configuration
- Retry strategies
- Webhook delivery with reliability

**What you'll learn:**
- Advanced `integration` patterns
- `circuit_breaker` configuration
- `retry` strategies
- `webhook` delivery

### 5. Scheduled Jobs (Specialist)

Background processing patterns:
- Cron-based scheduling
- Job queues with priorities
- Report generation
- Data maintenance

**What you'll learn:**
- `job` declarations
- Cron expressions
- Queue configuration
- Multi-step job logic

### 6. MongoDB Collections (Intermediate)

Document database modeling with MongoDB:
- Collection definitions with `database mongodb { }`
- Embedded documents for denormalized data
- Array fields for flexible structures
- MongoDB-specific index types

**What you'll learn:**
- `collection` declarations
- MongoDB field types: `objectid`, `embedded`, `array`
- Index types: standard, unique, text, geospatial
- When to use MongoDB vs PostgreSQL

### 7. Mixed Databases (Advanced)

Using PostgreSQL and MongoDB together:
- Dual database configuration
- Choosing the right database for each data type
- Cross-database event handling
- API endpoints for both databases

**What you'll learn:**
- Multi-database architecture patterns
- PostgreSQL for structured, transactional data
- MongoDB for flexible, high-volume data
- Event-driven cross-database operations

## Key Concepts by Example

| Concept | Hello World | Blog | E-commerce | Integrations | Jobs | MongoDB | Mixed DB |
|---------|:-----------:|:----:|:----------:|:------------:|:----:|:-------:|:--------:|
| Entities | ✓ | ✓ | ✓ | ✓ | ✓ | | ✓ |
| Endpoints | ✓ | ✓ | ✓ | ✓ | ✓ | | ✓ |
| Events | ✓ | ✓ | ✓ | ✓ | ✓ | | ✓ |
| Relationships | | ✓ | ✓ | | | | ✓ |
| RBAC | | ✓ | ✓ | ✓ | ✓ | | |
| Soft Delete | | ✓ | | | | | |
| Integrations | | | ✓ | ✓ | ✓ | | |
| Circuit Breaker | | | ✓ | ✓ | | | |
| Workflows | | | ✓ | | | | |
| Saga Pattern | | | ✓ | | | | |
| Webhooks | | | | ✓ | | | |
| GraphQL | | | | ✓ | | | |
| Scheduled Jobs | | | | | ✓ | | |
| Job Queues | | | | | ✓ | | |
| MongoDB Collections | | | | | | ✓ | ✓ |
| Embedded Documents | | | | | | ✓ | ✓ |
| Text Search | | | | | | ✓ | |
| Geospatial Indexes | | | | | | ✓ | |
| Dual Databases | | | | | | | ✓ |

## Common DSL Patterns

### Entity with Common Fields

```codeai
entity Example {
    id: uuid, primary, auto
    name: string, required
    description: text, optional
    status: enum(active, inactive), default(active)
    created_at: timestamp, auto
    updated_at: timestamp, auto_update
}
```

### REST Endpoint with Auth

```codeai
endpoint GET /items {
    description: "List all items"
    auth: required
    roles: [admin, user]

    query {
        status: string, optional
        page: integer, default(1)
        limit: integer, default(20)
    }

    returns: paginated(Item)
}
```

### Integration with Resilience

```codeai
integration ExternalAPI {
    type: rest
    base_url: env(API_URL)
    auth: bearer(env(API_TOKEN))

    timeout: 30s
    retry: 3 times with exponential_backoff
    circuit_breaker: {
        threshold: 5 failures in 1m
        reset_after: 30s
    }

    operation get_data {
        method: GET
        path: "/data"
        returns: json
    }
}
```

### Workflow with Steps

```codeai
workflow ProcessOrder {
    trigger: OrderPlaced

    steps {
        validate {
            check: trigger.order.items.length > 0
            on_fail: abort("Empty order")
        }

        process {
            call: PaymentGateway.charge {
                amount: trigger.order.total
            }
            timeout: 30s
            retry: 3 times
            on_fail: rollback
        }

        notify {
            send: email(trigger.customer.email) {
                template: "order_confirmed"
            }
        }
    }

    on_complete: emit(OrderConfirmed)
}
```

### Scheduled Job

```codeai
job DailyReport {
    schedule: "0 6 * * *"
    timezone: "UTC"

    queue: default
    timeout: 30m
    retry: 3 times

    steps {
        fetch_data {
            query: select Order where created_at >= yesterday()
            as: orders
        }

        generate_report {
            template: "daily_report"
            data: { orders: orders }
            format: pdf
            as: report
        }

        distribute {
            send: email(config.recipients) {
                attachment: report
            }
        }
    }
}
```

### MongoDB Collection with Embedded Documents

```cai
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "myapp"
}

database mongodb {
    collection Product {
        _id: objectid, primary, auto
        sku: string, required, unique
        name: string, required
        price: double, required

        categories: array(string), optional
        tags: array(string), optional

        attributes: embedded {
            color: string, optional
            size: string, optional
            weight: double, optional
        }

        created_at: date, auto

        indexes {
            index: [sku] unique
            index: [name] text
            index: [categories]
        }
    }
}
```

## Environment Variables

Each example requires certain environment variables. See the individual README files for details.

**Common Variables:**

```bash
# PostgreSQL Database
export DATABASE_URL="postgres://localhost:5432/dbname"

# MongoDB Database
export CODEAI_MONGODB_URI="mongodb://localhost:27017"
export CODEAI_MONGODB_DATABASE="myapp"

# Cache (Redis)
export REDIS_URL="redis://localhost:6379"

# Authentication
export JWT_ISSUER="your-issuer"
export JWT_SECRET="your-secret-key"
```

## Getting Help

- See [DSL Language Spec](../docs/dsl_language_spec.md) for full syntax reference
- See [DSL Cheatsheet](../docs/dsl_cheatsheet.md) for quick reference
- See [Database Guide](../docs/03-database.md) for PostgreSQL and MongoDB configuration
- See [Integration Patterns](../docs/integration_patterns.md) for resilience patterns
- See [Workflows and Jobs](../docs/workflows_and_jobs.md) for async processing
- See [Troubleshooting](../docs/troubleshooting.md) for common issues and solutions

## Contributing

To add a new example:

1. Create a new directory: `examples/XX-name/`
2. Add the DSL file: `name.cai`
3. Add documentation: `README.md`
4. Add test script: `test.sh`
5. Update this README with the new example

Make sure each example:
- Is self-contained
- Has clear documentation
- Includes working test commands
- Demonstrates specific CodeAI features
