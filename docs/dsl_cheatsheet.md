# CodeAI DSL Syntax Cheatsheet

Quick reference for CodeAI DSL v1.0

---

## Core Language (Implemented)

### Variables & Literals

| Syntax | Example | Description |
|--------|---------|-------------|
| `var name = value` | `var count = 42` | Variable declaration |
| `name = value` | `count = 100` | Assignment (must exist) |
| `"text"` | `"Hello, World!"` | String literal |
| `123` / `3.14` | `var price = 19.99` | Number literal |
| `true` / `false` | `var active = true` | Boolean literal |
| `[a, b, c]` | `var items = [1, 2, 3]` | Array literal |

### Control Flow

| Syntax | Example | Description |
|--------|---------|-------------|
| `if expr { }` | `if active { var x = 1 }` | Conditional |
| `if expr { } else { }` | `if x { } else { }` | If-else |
| `for item in array { }` | `for pod in pods { }` | For-in loop |

### Functions

| Syntax | Example | Description |
|--------|---------|-------------|
| `function name() { }` | `function greet() { }` | No parameters |
| `function name(a, b) { }` | `function add(x, y) { }` | With parameters |
| `name()` | `deploy()` | Function call |
| `name(arg1, arg2)` | `add(1, 2)` | Call with arguments |

### Shell Execution

| Syntax | Example | Description |
|--------|---------|-------------|
| `exec { cmd }` | `exec { kubectl get pods }` | Execute shell command |
| `$variable` | `exec { echo $name }` | Variable interpolation |
| `cmd \| cmd` | `exec { kubectl get pods \| grep Running }` | Pipe commands |

### Comments

| Syntax | Example | Description |
|--------|---------|-------------|
| `// comment` | `// This is a comment` | Single-line |
| `/* comment */` | `/* Multi-line */` | Multi-line |

---

## Business DSL (Planned)

### Entities

| Syntax | Example | Description |
|--------|---------|-------------|
| `entity Name { }` | `entity Product { }` | Entity declaration |
| `field: type` | `name: string` | Field definition |
| `field: type, modifiers` | `id: uuid, primary, auto` | Field with modifiers |
| `ref(Entity)` | `category: ref(Category)` | Foreign key reference |
| `list(type)` | `tags: list(string)` | Array field |
| `enum(a,b,c)` | `status: enum(active,inactive)` | Enumerated values |
| `decimal(p,s)` | `price: decimal(10,2)` | Fixed precision |
| `index: [fields]` | `index: [category, created_at]` | Composite index |

#### Field Types

| Type | Example | Description |
|------|---------|-------------|
| `uuid` | `id: uuid` | Unique identifier |
| `string` | `name: string` | Text (max 255) |
| `text` | `description: text` | Unlimited text |
| `integer` | `count: integer` | 64-bit integer |
| `decimal(p,s)` | `price: decimal(10,2)` | Fixed precision decimal |
| `boolean` | `active: boolean` | True/false |
| `timestamp` | `created_at: timestamp` | Date + time + timezone |
| `date` | `birth_date: date` | Date only |
| `time` | `start_time: time` | Time only |
| `json` | `metadata: json` | Arbitrary JSON |

#### Field Modifiers

| Modifier | Example | Description |
|----------|---------|-------------|
| `primary` | `id: uuid, primary` | Primary key |
| `auto` | `id: uuid, auto` | Auto-generated |
| `required` | `name: string, required` | Cannot be null |
| `optional` | `bio: text, optional` | Can be null |
| `unique` | `email: string, unique` | Must be unique |
| `searchable` | `name: string, searchable` | Full-text index |
| `default(val)` | `count: integer, default(0)` | Default value |
| `auto_update` | `updated_at: timestamp, auto_update` | Auto-update on change |
| `soft_delete` | Entity-level soft deletion |

### Endpoints

| Syntax | Example | Description |
|--------|---------|-------------|
| `endpoint METHOD /path { }` | `endpoint GET /products { }` | HTTP endpoint |
| `auth: required` | `auth: required` | Require authentication |
| `roles: [r1, r2]` | `roles: [admin, user]` | Role-based access |
| `query { field: type }` | `query { page: integer }` | Query parameters |
| `body { field: type }` | `body { name: string }` | Request body |
| `returns: Type` | `returns: Product` | Response type |
| `returns: paginated(T)` | `returns: paginated(Product)` | Paginated response |
| `on_success: emit(Event)` | `on_success: emit(ProductCreated)` | Emit event |

### Workflows

| Syntax | Example | Description |
|--------|---------|-------------|
| `workflow Name { }` | `workflow OrderFulfillment { }` | Workflow declaration |
| `trigger: Event` | `trigger: OrderPlaced` | Trigger event |
| `steps { }` | `steps { step1 { } }` | Workflow steps |
| `call: Integration.op { }` | `call: PaymentGateway.charge { }` | External call |
| `timeout: duration` | `timeout: 30s` | Step timeout |
| `retry: N times` | `retry: 3 times` | Retry count |
| `with exponential_backoff` | `retry: 3 times with exponential_backoff` | Backoff strategy |
| `on_fail: rollback` | `on_fail: rollback` | Failure handling |
| `for_each: collection` | `for_each: order.items` | Iterate items |
| `on_complete: action` | `on_complete: emit(Done)` | Completion handler |

### Jobs

| Syntax | Example | Description |
|--------|---------|-------------|
| `job Name { }` | `job DailyReport { }` | Job declaration |
| `schedule: "cron"` | `schedule: "0 6 * * *"` | Cron expression |
| `schedule: every Nd/h/m` | `schedule: every 1h` | Interval schedule |
| `timezone: "Zone"` | `timezone: "UTC"` | Timezone |
| `retry: N times` | `retry: 3 times` | Retry on failure |
| `timeout: duration` | `timeout: 10m` | Job timeout |
| `action: operation` | `action: delete Session where...` | Simple action |

### Integrations

| Syntax | Example | Description |
|--------|---------|-------------|
| `integration Name { }` | `integration PaymentGateway { }` | Integration declaration |
| `type: rest` | `type: rest` | REST API |
| `base_url: url` | `base_url: env(API_URL)` | Base URL |
| `auth: bearer(token)` | `auth: bearer(env(API_KEY))` | Bearer auth |
| `timeout: duration` | `timeout: 30s` | Request timeout |
| `circuit_breaker: { }` | `threshold: 5 failures in 1m` | Circuit breaker |
| `operation name { }` | `operation charge { }` | Define operation |
| `method: HTTP_METHOD` | `method: POST` | HTTP method |
| `path: "/path"` | `path: "/charges"` | Endpoint path |

#### Circuit Breaker Config

| Syntax | Example | Description |
|--------|---------|-------------|
| `threshold: N failures in T` | `threshold: 5 failures in 1m` | Failure threshold |
| `reset_after: duration` | `reset_after: 30s` | Reset interval |

#### Retry Strategies

| Syntax | Example | Description |
|--------|---------|-------------|
| `retry: N times` | `retry: 3 times` | Fixed retries |
| `with exponential_backoff` | `retry: 3 times with exponential_backoff` | Exponential backoff |
| `with linear_backoff` | `retry: 3 times with linear_backoff` | Linear backoff |

### Events

| Syntax | Example | Description |
|--------|---------|-------------|
| `event Name { }` | `event ProductCreated { }` | Event declaration |
| `payload { }` | `payload { id: uuid }` | Event data |
| `publish_to: [targets]` | `publish_to: [kafka("topic")]` | Publish targets |
| `trigger: when condition` | `trigger: when Product.quantity < 10` | Auto-trigger |

#### Publish Targets

| Syntax | Example | Description |
|--------|---------|-------------|
| `kafka("topic")` | `kafka("products")` | Kafka topic |
| `webhook("name")` | `webhook("inventory-updates")` | Webhook |
| `slack("#channel")` | `slack("#alerts")` | Slack channel |

### MongoDB Collections (Implemented)

| Syntax | Example | Description |
|--------|---------|-------------|
| `database mongodb { }` | `database mongodb { }` | MongoDB database block |
| `collection Name { }` | `collection User { }` | Collection declaration |
| `_id: objectid` | `_id: objectid, primary, auto` | ObjectID field |
| `field: embedded { }` | `location: embedded { }` | Embedded document |
| `field: array(type)` | `tags: array(string)` | Array field |
| `indexes { }` | `indexes { index: [email] unique }` | Index definitions |

#### MongoDB Field Types

| Type | Example | Description |
|------|---------|-------------|
| `objectid` | `_id: objectid` | MongoDB ObjectID |
| `string` | `name: string` | Text field |
| `int` | `count: int` | Integer |
| `double` | `price: double` | Floating point |
| `bool` | `active: bool` | Boolean |
| `date` | `created_at: date` | Date/timestamp |
| `array(T)` | `tags: array(string)` | Array of type T |
| `embedded { }` | `address: embedded { }` | Embedded document |
| `object` | `items: array(object)` | Flexible object |

#### MongoDB Index Types

| Syntax | Example | Description |
|--------|---------|-------------|
| `index: [fields]` | `index: [email]` | Standard index |
| `index: [fields] unique` | `index: [email] unique` | Unique index |
| `index: [fields] text` | `index: [title, body] text` | Text search index |
| `index: [fields] geospatial` | `index: [location] geospatial` | 2dsphere index |

### Configuration

| Syntax | Example | Description |
|--------|---------|-------------|
| `config { }` | `config { name: "service" }` | Config block |
| `database_type: "type"` | `database_type: "mongodb"` | Database type selection |
| `mongodb_uri: "uri"` | `mongodb_uri: "mongodb://localhost:27017"` | MongoDB connection URI |
| `mongodb_database: "name"` | `mongodb_database: "myapp"` | MongoDB database name |
| `database: type { }` | `database: postgres { pool_size: 20 }` | PostgreSQL config |
| `cache: type { }` | `cache: redis { ttl: 5m }` | Cache config |
| `auth: type { }` | `auth: jwt { issuer: env(JWT_ISSUER) }` | Auth config |
| `env(VAR)` | `env(DATABASE_URL)` | Environment variable |

---

## Duration Syntax

| Syntax | Example | Description |
|--------|---------|-------------|
| `Ns` | `30s` | Seconds |
| `Nm` | `5m` | Minutes |
| `Nh` | `1h` | Hours |
| `Nd` | `7d` | Days |

---

## Common Patterns

### 1. Basic CRUD Entity

```codeai
entity User {
    id: uuid, primary, auto
    email: string, required, unique
    name: string, required
    created_at: timestamp, auto
    updated_at: timestamp, auto_update
}
```

### 2. REST Endpoint with Validation

```codeai
endpoint POST /users {
    auth: required
    body {
        email: string, required
        name: string, required
    }
    returns: User
    on_success: emit(UserCreated)
}
```

### 3. External API Integration

```codeai
integration Stripe {
    type: rest
    base_url: env(STRIPE_URL)
    auth: bearer(env(STRIPE_KEY))
    timeout: 30s
    retry: 3 times with exponential_backoff
    circuit_breaker: {
        threshold: 5 failures in 1m
        reset_after: 30s
    }
}
```

### 4. Scheduled Cleanup Job

```codeai
job CleanupSessions {
    schedule: every 1h
    timeout: 5m
    action: delete Session where expires_at < now()
}
```

### 5. Event-Driven Notification

```codeai
event LowStock {
    trigger: when Product.quantity < 10
    payload { product_id: uuid, quantity: integer }
    publish_to: [slack("#inventory")]
}
```

### 6. Multi-Step Workflow

```codeai
workflow ProcessOrder {
    trigger: OrderPlaced
    steps {
        charge_payment {
            call: Stripe.charge { amount: trigger.total }
            timeout: 30s
            retry: 3 times with exponential_backoff
            on_fail: rollback
        }
        send_confirmation {
            send: email(trigger.customer.email) {
                template: "order_confirmed"
            }
        }
    }
    on_complete: update(trigger.status = "completed")
}
```

### 7. Shell Script with Loop

```codeai
var namespaces = ["dev", "staging", "prod"]

for ns in namespaces {
    exec { kubectl get pods -n $ns }
}
```

### 8. MongoDB Collection with Embedded Documents

```codeai
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "ecommerce"
}

database mongodb {
    collection Product {
        description: "E-commerce products with variants"

        _id: objectid, primary, auto
        sku: string, required, unique
        name: string, required
        price: double, required
        categories: array(string), optional

        attributes: embedded {
            color: string, optional
            size: string, optional
            weight: double, optional
        }

        created_at: date, auto
        updated_at: date, auto

        indexes {
            index: [sku] unique
            index: [name] text
            index: [categories]
        }
    }
}
```

### 9. MongoDB Collection with Geospatial Index

```codeai
database mongodb {
    collection Location {
        _id: objectid, primary, auto
        name: string, required

        coordinates: embedded {
            type: string, required, default("Point")
            coordinates: array(double), required
        }

        indexes {
            index: [coordinates] geospatial
        }
    }
}
```
