# CodeAI

## LLM-Native Programming Language

### Complete Implementation Plan

**Version 1.0**  
**January 2026**  
**Binary Data Consulting BV**

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Vision and Design Philosophy](#2-vision-and-design-philosophy)
3. [Language Specification (CodeAI DSL)](#3-language-specification-codeai-dsl)
4. [Go Runtime Architecture](#4-go-runtime-architecture)
5. [Core Modules Specification](#5-core-modules-specification)
6. [Parser Implementation](#6-parser-implementation)
7. [Standard Library](#7-standard-library)
8. [LLM Integration Guidelines](#8-llm-integration-guidelines)
9. [Implementation Phases](#9-implementation-phases)
10. [Testing Strategy](#10-testing-strategy)
11. [Deployment Model](#11-deployment-model)
12. [Performance Considerations](#12-performance-considerations)
13. [Security Model](#13-security-model)
14. [Appendices](#14-appendices)

---

## 1. Executive Summary

### 1.1 Project Overview

CodeAI is a domain-specific programming language designed specifically for Large Language Model (LLM) code generation. Unlike traditional programming languages optimized for human developers, CodeAI prioritizes patterns and syntax that LLMs can generate with high accuracy and consistency.

The language targets backend business applications, providing built-in primitives for databases, APIs, workflows, integrations, and background processing. CodeAI compiles to a high-performance Go runtime that handles all low-level concerns including connection pooling, security, concurrency, and resource management.

### 1.2 Key Objectives

- Maximize LLM code generation accuracy through syntax designed for token prediction patterns
- Eliminate common LLM coding errors: off-by-one, null handling, resource leaks, SQL injection
- Provide high-level business primitives that abstract away infrastructure complexity
- Deploy as single binary with no external dependencies (like Go itself)
- Achieve production-grade performance suitable for enterprise workloads

### 1.3 Target Use Cases

CodeAI is designed for backend business applications including REST and GraphQL APIs, database-driven applications (CRUD operations), background job processing and workflows, third-party service integrations, event-driven architectures, and scheduled tasks and automation.

### 1.4 Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Runtime | Go 1.22+ | Fast compilation, single binary, excellent concurrency |
| Parser | Participle v2 | Native Go, grammar-based, excellent error messages |
| HTTP Server | Chi Router + net/http | Lightweight, idiomatic Go, high performance |
| Database | pgx (Postgres), mongo-driver | Native drivers, connection pooling built-in |
| Background Jobs | Asynq | Redis-backed, reliable, Go-native |
| Configuration | Viper | Multi-format support, environment variables |

---

## 2. Vision and Design Philosophy

### 2.1 Why LLMs Need a New Language

Current programming languages were designed for human cognitive patterns: visual parsing, muscle memory, IDE assistance, and iterative debugging. LLMs operate fundamentally differently, predicting tokens based on statistical patterns learned from training data.

This mismatch causes systematic errors when LLMs generate code in traditional languages:

- **Off-by-one errors** occur because loop boundaries require precise counting
- **Null pointer exceptions** happen because LLMs inconsistently apply null checks
- **Resource leaks** emerge from forgetting cleanup in complex control flows
- **Security vulnerabilities** arise from SQL injection, XSS, and other injection attacks that require consistent escaping

CodeAI addresses these issues by designing syntax and semantics that align with LLM strengths while the runtime handles error-prone details.

### 2.2 Core Design Principles

#### 2.2.1 Declarative Over Imperative

LLMs excel at describing what should happen, not the precise sequence of how. CodeAI uses declarative constructs wherever possible. Instead of writing loop logic, developers declare data transformations. Instead of managing HTTP routing internals, they declare endpoints with constraints.

#### 2.2.2 Fault-Tolerant Syntax

The parser accepts multiple valid forms for the same construct. Trailing commas are always allowed. Keywords can be abbreviated. Whitespace and indentation are flexible. The compiler applies a canonical form, so variations in LLM output still produce identical compiled results.

#### 2.2.3 Semantic Comments

In CodeAI, comments are not just documentation but carry semantic weight. A description on an endpoint affects OpenAPI generation. Comments on entities influence database indexing hints. This aligns with how LLMs naturally produce code with explanatory text.

#### 2.2.4 Business-Domain Primitives

Rather than building from low-level primitives, CodeAI provides first-class constructs for common business patterns. Entity, endpoint, workflow, job, integration, and event are built-in keywords. This reduces the surface area where LLMs can make errors.

#### 2.2.5 Safe by Default

All potentially dangerous operations are handled by the runtime. SQL queries are always parameterized. HTTP responses are automatically escaped. Authentication and authorization are declarative. Resource cleanup is automatic. The LLM cannot generate insecure code because the language does not expose insecure primitives.

---

## 3. Language Specification (CodeAI DSL)

### 3.1 File Structure

CodeAI programs consist of one or more `.codeai` files. Each file contains zero or more top-level declarations. The order of declarations does not matter; the compiler resolves dependencies automatically.

```codeai
# app.codeai
config {
    name: "my-application"
    version: "1.0.0"
    database: postgres
}

entity User { ... }
endpoint GET /users { ... }
workflow OnUserCreated { ... }
```

### 3.2 Configuration Block

Every CodeAI application begins with a config block that defines application metadata and runtime settings.

```codeai
config {
    name: "inventory-service"
    version: "2.1.0"
    
    database: postgres {
        pool_size: 20
        timeout: 30s
    }
    
    cache: redis {
        ttl: 5m
    }
    
    auth: jwt {
        issuer: "https://auth.example.com"
        audience: "api.example.com"
    }
    
    cors: {
        origins: ["https://app.example.com"]
        methods: [GET, POST, PUT, DELETE]
    }
}
```

### 3.3 Entity Declarations

Entities define data models that map to database tables or collections. The runtime handles schema migrations automatically.

```codeai
entity Product {
    description: "Represents a product in the inventory"
    
    id: uuid, primary, auto
    sku: string, unique, required
    name: string, required, searchable
    description: text, optional
    price: decimal(10,2), required
    quantity: integer, default(0)
    category: ref(Category), required
    tags: list(string), optional
    metadata: json, optional
    
    created_at: timestamp, auto
    updated_at: timestamp, auto_update
    deleted_at: timestamp, soft_delete
    
    index: [category, created_at]
    index: [sku] unique
}
```

#### 3.3.1 Field Types

| Type | Description | Database Mapping |
|------|-------------|------------------|
| `uuid` | Universally unique identifier | UUID / VARCHAR(36) |
| `string` | Variable length text (max 255) | VARCHAR(255) |
| `text` | Unlimited length text | TEXT / LONGTEXT |
| `integer` | 64-bit signed integer | BIGINT |
| `decimal(p,s)` | Fixed precision decimal | DECIMAL(p,s) |
| `boolean` | True/false value | BOOLEAN |
| `timestamp` | Date and time with timezone | TIMESTAMPTZ |
| `date` | Date without time | DATE |
| `time` | Time without date | TIME |
| `json` | Arbitrary JSON data | JSONB |
| `list(T)` | Array of type T | ARRAY / JSON |
| `ref(Entity)` | Foreign key reference | UUID + FK constraint |
| `enum(a,b,c)` | Enumerated values | ENUM / VARCHAR + CHECK |

#### 3.3.2 Field Modifiers

| Modifier | Description |
|----------|-------------|
| `primary` | Primary key field |
| `auto` | Auto-generated value (UUID or timestamp) |
| `auto_update` | Automatically updated on modification |
| `required` | Field cannot be null |
| `optional` | Field can be null (default) |
| `unique` | Value must be unique across all records |
| `searchable` | Create full-text search index |
| `default(value)` | Default value if not provided |
| `soft_delete` | Enable soft deletion pattern |

### 3.4 Endpoint Declarations

Endpoints define HTTP API routes. The runtime handles routing, validation, serialization, and error responses.

```codeai
endpoint GET /products {
    description: "List all products with optional filtering"
    auth: required
    
    query {
        category: ref(Category), optional
        min_price: decimal, optional
        max_price: decimal, optional
        search: string, optional
        page: integer, default(1)
        limit: integer, default(20), max(100)
    }
    
    returns: paginated(Product)
    
    filter: {
        category == query.category if provided
        price >= query.min_price if provided
        price <= query.max_price if provided
        name contains query.search if provided
    }
    
    sort: created_at desc
}
```

```codeai
endpoint POST /products {
    description: "Create a new product"
    auth: required
    roles: [admin, inventory_manager]
    
    body {
        sku: string, required
        name: string, required
        description: text, optional
        price: decimal, required, min(0)
        quantity: integer, optional, min(0)
        category: ref(Category), required
        tags: list(string), optional
    }
    
    returns: Product
    
    validate: {
        sku is unique
        category exists
    }
    
    on_success: emit(ProductCreated)
}
```

```codeai
endpoint GET /products/{id} {
    description: "Get a single product by ID"
    auth: optional
    cache: 5m
    
    path {
        id: uuid, required
    }
    
    returns: Product
    
    error 404: "Product not found"
}
```

```codeai
endpoint PUT /products/{id} {
    description: "Update an existing product"
    auth: required
    roles: [admin, inventory_manager]
    
    path {
        id: uuid, required
    }
    
    body {
        name: string, optional
        description: text, optional
        price: decimal, optional, min(0)
        quantity: integer, optional, min(0)
        tags: list(string), optional
    }
    
    returns: Product
    
    on_success: emit(ProductUpdated)
}
```

### 3.5 Workflow Declarations

Workflows define multi-step processes triggered by events. The runtime handles state persistence, retries, and failure recovery.

```codeai
workflow OrderFulfillment {
    description: "Process an order from placement to delivery"
    trigger: OrderPlaced
    
    steps {
        validate_inventory {
            for_each: trigger.order.items
            check: item.product.quantity >= item.quantity
            on_fail: cancel_order("Insufficient inventory")
        }
        
        reserve_inventory {
            for_each: trigger.order.items
            action: decrement(item.product.quantity, item.quantity)
            compensate: increment(item.product.quantity, item.quantity)
        }
        
        process_payment {
            call: PaymentGateway.charge {
                amount: trigger.order.total
                customer: trigger.order.customer
                idempotency_key: trigger.order.id
            }
            timeout: 30s
            retry: 3 times with exponential_backoff
            on_fail: rollback
        }
        
        create_shipment {
            call: ShippingService.create {
                address: trigger.order.shipping_address
                items: trigger.order.items
            }
            on_success: {
                update: trigger.order.status = "shipped"
                emit: OrderShipped
            }
        }
        
        notify_customer {
            send: email(trigger.order.customer.email) {
                template: "order_shipped"
                data: {
                    order_id: trigger.order.id
                    tracking_number: steps.create_shipment.tracking_number
                }
            }
        }
    }
    
    on_complete: update(trigger.order.status = "completed")
    on_fail: {
        update: trigger.order.status = "failed"
        emit: OrderFailed
        alert: ops_team
    }
}
```

### 3.6 Job Declarations

Jobs define scheduled or recurring background tasks.

```codeai
job DailyInventoryReport {
    description: "Generate daily inventory report"
    schedule: "0 6 * * *"  # 6 AM daily
    timezone: "UTC"
    
    steps {
        fetch_data {
            query: select Product where quantity < 10
            as: low_stock_items
        }
        
        generate_report {
            template: "inventory_report"
            data: {
                date: today()
                low_stock: low_stock_items
                total_products: count(Product)
                total_value: sum(Product.price * Product.quantity)
            }
            format: pdf
            as: report_file
        }
        
        distribute {
            send: email(config.reports.recipients) {
                subject: "Daily Inventory Report - {today()}"
                attachment: report_file
            }
        }
    }
    
    retry: 3 times
    timeout: 10m
    on_fail: alert(ops_team)
}
```

```codeai
job CleanupExpiredSessions {
    description: "Remove expired user sessions"
    schedule: every 1h
    
    action: delete Session where expires_at < now()
    
    log: "{count} expired sessions removed"
}
```

### 3.7 Integration Declarations

Integrations define connections to external services with built-in retry logic, circuit breaking, and error handling.

```codeai
integration PaymentGateway {
    description: "Stripe payment processing"
    type: rest
    base_url: env(STRIPE_API_URL, "https://api.stripe.com/v1")
    
    auth: bearer(env(STRIPE_SECRET_KEY))
    
    headers: {
        "Stripe-Version": "2023-10-16"
    }
    
    timeout: 30s
    retry: 3 times with exponential_backoff
    circuit_breaker: {
        threshold: 5 failures in 1m
        reset_after: 30s
    }
    
    operation charge {
        method: POST
        path: "/charges"
        body: {
            amount: integer, required
            currency: string, default("usd")
            source: string, required
            description: string, optional
            idempotency_key: string, optional
        }
        returns: {
            id: string
            status: string
            amount: integer
        }
    }
    
    operation refund {
        method: POST
        path: "/refunds"
        body: {
            charge: string, required
            amount: integer, optional
        }
        returns: {
            id: string
            status: string
        }
    }
}
```

### 3.8 Event Declarations

Events define messages that can be emitted and consumed within the system or published to external message brokers.

```codeai
event ProductCreated {
    description: "Emitted when a new product is added"
    
    payload {
        product_id: uuid
        sku: string
        name: string
        price: decimal
        created_by: ref(User)
        created_at: timestamp
    }
    
    publish_to: [kafka("products"), webhook("inventory-updates")]
}

event LowStockAlert {
    description: "Emitted when product quantity falls below threshold"
    
    payload {
        product_id: uuid
        product_name: string
        current_quantity: integer
        threshold: integer
    }
    
    trigger: when Product.quantity < 10
    
    publish_to: [slack("#inventory-alerts"), email(config.alerts.recipients)]
}
```

### 3.9 Function Declarations

Functions define reusable logic that can be called from endpoints, workflows, or jobs.

```codeai
function calculate_order_total(items: list(OrderItem)) -> decimal {
    description: "Calculate total price including tax and discounts"
    
    let subtotal = sum(items.map(i => i.price * i.quantity))
    let discount = apply_discount(subtotal, items)
    let tax = calculate_tax(subtotal - discount)
    
    return subtotal - discount + tax
}

function apply_discount(subtotal: decimal, items: list(OrderItem)) -> decimal {
    if subtotal >= 100 {
        return subtotal * 0.10  # 10% off orders over $100
    }
    return 0
}

function calculate_tax(amount: decimal) -> decimal {
    return amount * config.tax_rate
}
```

### 3.10 Query Language

CodeAI includes a built-in query language for data operations that is safer and more LLM-friendly than raw SQL.

```codeai
# Select queries
select Product
select Product where category == "electronics"
select Product where price > 100 and quantity > 0
select Product where name contains "phone" order by price desc
select Product where tags includes "featured" limit 10

# Aggregations
count(Product where category == "electronics")
sum(Product.price where in_stock == true)
avg(Product.price) group by category

# Joins (implicit through references)
select Order with customer, items.product
select Product with category where category.name == "Electronics"

# Updates
update Product set quantity = quantity - 1 where id == {product_id}
update Product set price = price * 1.1 where category == "premium"

# Deletes
delete Product where id == {product_id}
delete Session where expires_at < now()
```

### 3.11 Expression Syntax

| Expression | Description | Example |
|------------|-------------|---------|
| Arithmetic | Math operations | `price * quantity + tax` |
| Comparison | Value comparison | `age >= 18 and status == "active"` |
| Logical | Boolean logic | `is_admin or has_permission("edit")` |
| Null coalescing | Default values | `nickname ?? name ?? "Anonymous"` |
| Conditional | Inline conditions | `if quantity > 0 then "In Stock" else "Out of Stock"` |
| List operations | Collection handling | `items.map(i => i.total).sum()` |
| String interpolation | Dynamic strings | `"Hello, {user.name}!"` |
| Date arithmetic | Time calculations | `now() + 7.days` |

---

## 4. Go Runtime Architecture

### 4.1 Architecture Overview

The CodeAI runtime is a Go application that parses CodeAI source files, builds an Abstract Syntax Tree (AST), and executes the program through a module-based engine. The runtime handles all low-level concerns while the CodeAI code focuses purely on business logic.

```
┌──────────────────────────────────────────────────────────────┐
│                     CodeAI Source Files                       │
│                    (.codeai files)                            │
└─────────────────────────┬────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                         Parser                                │
│              (Participle grammar → AST)                       │
└─────────────────────────┬────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                       Validator                               │
│          (Type checking, reference resolution)                │
└─────────────────────────┬────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                    Runtime Engine                             │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌───────────┐  │
│  │  Database  │ │    HTTP    │ │  Workflow  │ │   Event   │  │
│  │   Module   │ │   Module   │ │   Module   │ │   Module  │  │
│  └────────────┘ └────────────┘ └────────────┘ └───────────┘  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌───────────┐  │
│  │    Job     │ │Integration │ │   Cache    │ │   Auth    │  │
│  │   Module   │ │   Module   │ │   Module   │ │   Module  │  │
│  └────────────┘ └────────────┘ └────────────┘ └───────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 Directory Structure

```
codeai/
├── cmd/
│   └── codeai/
│       └── main.go              # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── grammar.go           # Participle grammar definitions
│   │   ├── ast.go               # AST node types
│   │   ├── parser.go            # Main parser logic
│   │   └── lexer.go             # Custom lexer rules
│   ├── validator/
│   │   ├── validator.go         # Type checking and validation
│   │   ├── resolver.go          # Reference resolution
│   │   └── errors.go            # Validation error types
│   ├── runtime/
│   │   ├── engine.go            # Main execution engine
│   │   ├── context.go           # Request/execution context
│   │   ├── evaluator.go         # Expression evaluation
│   │   └── types.go             # Runtime type system
│   ├── modules/
│   │   ├── database/
│   │   │   ├── module.go        # Database module interface
│   │   │   ├── postgres.go      # PostgreSQL implementation
│   │   │   ├── mongodb.go       # MongoDB implementation
│   │   │   ├── migrations.go    # Auto-migration logic
│   │   │   └── query.go         # Query builder
│   │   ├── http/
│   │   │   ├── module.go        # HTTP module interface
│   │   │   ├── server.go        # HTTP server setup
│   │   │   ├── router.go        # Route registration
│   │   │   ├── middleware.go    # Common middleware
│   │   │   └── handlers.go      # Request handlers
│   │   ├── workflow/
│   │   │   ├── module.go        # Workflow module interface
│   │   │   ├── executor.go      # Workflow execution
│   │   │   ├── state.go         # State persistence
│   │   │   └── compensation.go  # Rollback handling
│   │   ├── job/
│   │   │   ├── module.go        # Job module interface
│   │   │   ├── scheduler.go     # Cron scheduling
│   │   │   └── worker.go        # Job execution
│   │   ├── event/
│   │   │   ├── module.go        # Event module interface
│   │   │   ├── emitter.go       # Event emission
│   │   │   ├── subscriber.go    # Event subscription
│   │   │   └── publishers/      # External publishers (Kafka, etc.)
│   │   ├── integration/
│   │   │   ├── module.go        # Integration module interface
│   │   │   ├── client.go        # HTTP client wrapper
│   │   │   ├── circuit.go       # Circuit breaker
│   │   │   └── retry.go         # Retry logic
│   │   ├── cache/
│   │   │   ├── module.go        # Cache module interface
│   │   │   ├── redis.go         # Redis implementation
│   │   │   └── memory.go        # In-memory implementation
│   │   └── auth/
│   │       ├── module.go        # Auth module interface
│   │       ├── jwt.go           # JWT validation
│   │       └── rbac.go          # Role-based access control
│   └── stdlib/
│       ├── functions.go         # Built-in functions
│       ├── datetime.go          # Date/time functions
│       ├── string.go            # String functions
│       └── math.go              # Math functions
├── pkg/
│   └── codeai/
│       └── embed.go             # Embeddable runtime API
├── go.mod
└── go.sum
```

### 4.3 Core Interfaces

The runtime is built on a modular interface system that allows different implementations to be swapped.

```go
// internal/runtime/engine.go
package runtime

type Engine struct {
    config    *Config
    parser    *parser.Parser
    validator *validator.Validator
    modules   map[string]Module
    ctx       context.Context
}

type Module interface {
    Name() string
    Initialize(config *Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}

type Config struct {
    App         AppConfig
    Database    DatabaseConfig
    HTTP        HTTPConfig
    Cache       CacheConfig
    Auth        AuthConfig
    Logging     LoggingConfig
    Metrics     MetricsConfig
}

func NewEngine(sourceDir string) (*Engine, error) {
    e := &Engine{
        modules: make(map[string]Module),
    }
    
    // Parse all .codeai files
    ast, err := e.parser.ParseDirectory(sourceDir)
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }
    
    // Validate AST
    if err := e.validator.Validate(ast); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    
    // Initialize modules based on config
    e.initializeModules(ast.Config)
    
    return e, nil
}

func (e *Engine) Run(ctx context.Context) error {
    // Start all modules
    for _, m := range e.modules {
        if err := m.Start(ctx); err != nil {
            return fmt.Errorf("failed to start %s: %w", m.Name(), err)
        }
    }
    
    // Wait for shutdown signal
    <-ctx.Done()
    
    // Graceful shutdown
    return e.shutdown()
}
```

### 4.4 Execution Context

Every request, workflow step, or job execution operates within a context that provides access to runtime services.

```go
// internal/runtime/context.go
package runtime

type ExecutionContext struct {
    ctx        context.Context
    requestID  string
    traceID    string
    user       *User
    logger     *slog.Logger
    db         DatabaseModule
    cache      CacheModule
    events     EventModule
    variables  map[string]any
}

func (c *ExecutionContext) Query(query string, args ...any) ([]map[string]any, error) {
    // All queries are automatically parameterized
    return c.db.Query(c.ctx, query, args...)
}

func (c *ExecutionContext) GetEntity(entityType string, id string) (map[string]any, error) {
    return c.db.FindByID(c.ctx, entityType, id)
}

func (c *ExecutionContext) SaveEntity(entityType string, data map[string]any) (map[string]any, error) {
    return c.db.Save(c.ctx, entityType, data)
}

func (c *ExecutionContext) Emit(event string, payload map[string]any) error {
    return c.events.Emit(c.ctx, event, payload)
}

func (c *ExecutionContext) Cache(key string, ttl time.Duration, fn func() (any, error)) (any, error) {
    return c.cache.GetOrSet(c.ctx, key, ttl, fn)
}

func (c *ExecutionContext) User() *User {
    return c.user
}

func (c *ExecutionContext) HasRole(role string) bool {
    return c.user != nil && c.user.HasRole(role)
}
```

---

## 5. Core Modules Specification

### 5.1 Database Module

The database module handles all data persistence operations with automatic connection pooling, query parameterization, and migration management.

```go
// internal/modules/database/module.go
package database

type DatabaseModule interface {
    Module
    
    // Query execution
    Query(ctx context.Context, query string, args ...any) ([]Record, error)
    QueryOne(ctx context.Context, query string, args ...any) (Record, error)
    Execute(ctx context.Context, query string, args ...any) (Result, error)
    
    // Entity operations
    FindByID(ctx context.Context, entity string, id string) (Record, error)
    FindAll(ctx context.Context, entity string, opts QueryOptions) ([]Record, error)
    Create(ctx context.Context, entity string, data Record) (Record, error)
    Update(ctx context.Context, entity string, id string, data Record) (Record, error)
    Delete(ctx context.Context, entity string, id string) error
    
    // Transactions
    Transaction(ctx context.Context, fn func(tx Transaction) error) error
    
    // Migrations
    Migrate(ctx context.Context, entities []Entity) error
    MigrationStatus(ctx context.Context) ([]Migration, error)
}

type QueryOptions struct {
    Where   map[string]any
    OrderBy []OrderClause
    Limit   int
    Offset  int
    Include []string  // Related entities to load
}

type Record map[string]any

type Result struct {
    RowsAffected int64
    LastInsertID string
}
```

#### 5.1.1 PostgreSQL Implementation

```go
// internal/modules/database/postgres.go
package database

import (
    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresModule struct {
    pool     *pgxpool.Pool
    entities map[string]*EntityMeta
    logger   *slog.Logger
}

func NewPostgresModule(config DatabaseConfig) (*PostgresModule, error) {
    poolConfig, err := pgxpool.ParseConfig(config.ConnectionString)
    if err != nil {
        return nil, err
    }
    
    poolConfig.MaxConns = int32(config.PoolSize)
    poolConfig.MinConns = int32(config.MinPoolSize)
    poolConfig.MaxConnLifetime = config.MaxConnLifetime
    poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
    
    pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
    if err != nil {
        return nil, err
    }
    
    return &PostgresModule{
        pool:     pool,
        entities: make(map[string]*EntityMeta),
        logger:   slog.Default().With("module", "postgres"),
    }, nil
}

func (m *PostgresModule) Query(ctx context.Context, query string, args ...any) ([]Record, error) {
    rows, err := m.pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    return pgx.CollectRows(rows, pgx.RowToMap)
}
```

### 5.2 HTTP Module

The HTTP module handles all API routing, request validation, response serialization, and middleware.

```go
// internal/modules/http/module.go
package http

type HTTPModule interface {
    Module
    
    // Route registration
    RegisterEndpoint(endpoint *Endpoint) error
    
    // Server control
    ListenAndServe() error
    Shutdown(ctx context.Context) error
    
    // Middleware
    Use(middleware Middleware)
}

type Endpoint struct {
    Method      string
    Path        string
    Description string
    Auth        AuthRequirement
    Roles       []string
    PathParams  []ParamDef
    QueryParams []ParamDef
    Body        *BodyDef
    Returns     *ReturnDef
    Handler     HandlerFunc
}

type HandlerFunc func(ctx *ExecutionContext, req *Request) (*Response, error)

type Request struct {
    PathParams  map[string]string
    QueryParams map[string]any
    Body        map[string]any
    Headers     http.Header
    User        *User
}

type Response struct {
    Status  int
    Body    any
    Headers map[string]string
}
```

```go
// internal/modules/http/server.go
package http

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Server struct {
    router    *chi.Mux
    config    HTTPConfig
    endpoints []*Endpoint
    auth      AuthModule
    logger    *slog.Logger
}

func NewServer(config HTTPConfig, auth AuthModule) *Server {
    r := chi.NewRouter()
    
    // Standard middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(config.RequestTimeout))
    
    // CORS
    if config.CORS.Enabled {
        r.Use(cors.Handler(cors.Options{
            AllowedOrigins:   config.CORS.Origins,
            AllowedMethods:   config.CORS.Methods,
            AllowedHeaders:   config.CORS.Headers,
            AllowCredentials: config.CORS.Credentials,
        }))
    }
    
    return &Server{
        router: r,
        config: config,
        auth:   auth,
        logger: slog.Default().With("module", "http"),
    }
}

func (s *Server) RegisterEndpoint(ep *Endpoint) error {
    handler := s.buildHandler(ep)
    
    switch ep.Method {
    case "GET":
        s.router.Get(ep.Path, handler)
    case "POST":
        s.router.Post(ep.Path, handler)
    case "PUT":
        s.router.Put(ep.Path, handler)
    case "DELETE":
        s.router.Delete(ep.Path, handler)
    case "PATCH":
        s.router.Patch(ep.Path, handler)
    }
    
    s.endpoints = append(s.endpoints, ep)
    return nil
}
```

### 5.3 Workflow Module

The workflow module executes multi-step processes with state persistence, compensation (rollback), and failure handling.

```go
// internal/modules/workflow/module.go
package workflow

type WorkflowModule interface {
    Module
    
    // Registration
    RegisterWorkflow(wf *Workflow) error
    
    // Execution
    Start(ctx context.Context, workflowID string, trigger any) (string, error)
    Resume(ctx context.Context, executionID string) error
    Cancel(ctx context.Context, executionID string) error
    
    // Status
    GetStatus(ctx context.Context, executionID string) (*ExecutionStatus, error)
}

type Workflow struct {
    ID          string
    Description string
    Trigger     string  // Event name that triggers this workflow
    Steps       []*Step
    OnComplete  *Action
    OnFail      *Action
    Timeout     time.Duration
}

type Step struct {
    Name        string
    Type        StepType  // action, condition, loop, parallel
    Action      *Action
    Condition   *Condition
    ForEach     *ForEach
    Timeout     time.Duration
    Retry       *RetryConfig
    Compensate  *Action  // Rollback action
    OnSuccess   *Action
    OnFail      *Action
}

type ExecutionStatus struct {
    ID           string
    WorkflowID   string
    Status       Status  // running, completed, failed, cancelled
    CurrentStep  string
    StepResults  map[string]any
    StartedAt    time.Time
    CompletedAt  *time.Time
    Error        string
}
```

### 5.4 Job Module

```go
// internal/modules/job/module.go
package job

import "github.com/hibiken/asynq"

type JobModule interface {
    Module
    
    // Registration
    RegisterJob(job *Job) error
    
    // Manual execution
    Enqueue(ctx context.Context, jobID string, payload any) error
    EnqueueAt(ctx context.Context, jobID string, payload any, at time.Time) error
    
    // Status
    GetScheduled(ctx context.Context) ([]*ScheduledJob, error)
}

type Job struct {
    ID          string
    Description string
    Schedule    string  // Cron expression or "every 1h"
    Timezone    string
    Steps       []*JobStep
    Timeout     time.Duration
    Retry       int
    OnFail      *Action
}

type AsynqJobModule struct {
    client    *asynq.Client
    server    *asynq.Server
    scheduler *asynq.Scheduler
    jobs      map[string]*Job
}

func (m *AsynqJobModule) Start(ctx context.Context) error {
    // Register all scheduled jobs
    for _, job := range m.jobs {
        if job.Schedule != "" {
            _, err := m.scheduler.Register(job.Schedule, asynq.NewTask(job.ID, nil))
            if err != nil {
                return err
            }
        }
    }
    
    // Start scheduler and worker
    go m.scheduler.Start()
    go m.server.Start(m.buildMux())
    
    return nil
}
```

### 5.5 Event Module

```go
// internal/modules/event/module.go
package event

type EventModule interface {
    Module
    
    // Registration
    RegisterEvent(event *Event) error
    Subscribe(eventID string, handler EventHandler) error
    
    // Emission
    Emit(ctx context.Context, eventID string, payload any) error
}

type Event struct {
    ID          string
    Description string
    Payload     *PayloadDef
    PublishTo   []Publisher
}

type Publisher interface {
    Publish(ctx context.Context, event string, payload any) error
}

type EventHandler func(ctx *ExecutionContext, payload any) error

// Built-in publishers
type KafkaPublisher struct { ... }
type WebhookPublisher struct { ... }
type SlackPublisher struct { ... }
type EmailPublisher struct { ... }
```

### 5.6 Integration Module

```go
// internal/modules/integration/module.go
package integration

type IntegrationModule interface {
    Module
    
    RegisterIntegration(integration *Integration) error
    Call(ctx context.Context, integrationID string, operation string, params any) (any, error)
}

type Integration struct {
    ID             string
    Description    string
    Type           string  // rest, grpc, graphql
    BaseURL        string
    Auth           AuthConfig
    Headers        map[string]string
    Timeout        time.Duration
    Retry          *RetryConfig
    CircuitBreaker *CircuitBreakerConfig
    Operations     map[string]*Operation
}

type Operation struct {
    Method   string
    Path     string
    Body     *BodyDef
    Returns  *ReturnDef
}

// Circuit breaker implementation
type CircuitBreaker struct {
    state       State  // closed, open, half-open
    failures    int
    threshold   int
    resetAfter  time.Duration
    lastFailure time.Time
    mu          sync.RWMutex
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    
    if cb.state == StateClosed {
        return true
    }
    
    if cb.state == StateOpen && time.Since(cb.lastFailure) > cb.resetAfter {
        cb.state = StateHalfOpen
        return true
    }
    
    return cb.state == StateHalfOpen
}
```

### 5.7 Auth Module

```go
// internal/modules/auth/module.go
package auth

type AuthModule interface {
    Module
    
    // Token validation
    ValidateToken(ctx context.Context, token string) (*User, error)
    
    // Authorization
    HasPermission(user *User, permission string) bool
    HasRole(user *User, role string) bool
}

type User struct {
    ID          string
    Email       string
    Roles       []string
    Permissions []string
    Claims      map[string]any
    Token       string
}

type JWTAuthModule struct {
    issuer     string
    audience   string
    publicKeys map[string]*rsa.PublicKey
    jwksURL    string
}

func (m *JWTAuthModule) ValidateToken(ctx context.Context, tokenStr string) (*User, error) {
    token, err := jwt.Parse(tokenStr, m.keyFunc)
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    
    // Validate issuer and audience
    if !claims.VerifyIssuer(m.issuer, true) {
        return nil, ErrInvalidIssuer
    }
    if !claims.VerifyAudience(m.audience, true) {
        return nil, ErrInvalidAudience
    }
    
    return &User{
        ID:          claims["sub"].(string),
        Email:       claims["email"].(string),
        Roles:       extractRoles(claims),
        Permissions: extractPermissions(claims),
        Claims:      claims,
        Token:       tokenStr,
    }, nil
}
```

---

## 6. Parser Implementation

### 6.1 Participle Grammar

The parser uses Participle v2, a Go-native parser generator that uses struct tags to define grammar rules.

```go
// internal/parser/grammar.go
package parser

import "github.com/alecthomas/participle/v2"

type Program struct {
    Declarations []*Declaration `@@*`
}

type Declaration struct {
    Config      *ConfigBlock      `  @@`
    Entity      *EntityDecl       `| @@`
    Endpoint    *EndpointDecl     `| @@`
    Workflow    *WorkflowDecl     `| @@`
    Job         *JobDecl          `| @@`
    Integration *IntegrationDecl  `| @@`
    Event       *EventDecl        `| @@`
    Function    *FunctionDecl     `| @@`
}

type ConfigBlock struct {
    Settings []*ConfigSetting `"config" "{" @@* "}"`
}

type ConfigSetting struct {
    Key   string       `@Ident ":"`
    Value *ConfigValue `@@`
}

type EntityDecl struct {
    Name        string        `"entity" @Ident "{"`
    Description *string       `("description" ":" @String)?`
    Fields      []*FieldDecl  `@@*`
    Indexes     []*IndexDecl  `@@* "}"`
}

type FieldDecl struct {
    Name      string      `@Ident ":"`
    Type      *TypeRef    `@@`
    Modifiers []string    `("," @Ident)*`
}

type TypeRef struct {
    Name   string     `@Ident`
    Params []*TypeRef `("(" @@ ("," @@)* ")")?`
}

type EndpointDecl struct {
    Method      string         `"endpoint" @("GET"|"POST"|"PUT"|"DELETE"|"PATCH")`
    Path        string         `@Path "{"`
    Description *string        `("description" ":" @String)?`
    Auth        *AuthClause    `@@?`
    Roles       []string       `("roles" ":" "[" @Ident ("," @Ident)* "]")?`
    PathParams  *ParamsBlock   `("path" @@)?`
    QueryParams *ParamsBlock   `("query" @@)?`
    Body        *ParamsBlock   `("body" @@)?`
    Returns     *ReturnClause  `@@?`
    Filter      *FilterBlock   `@@?`
    Actions     []*ActionDecl  `@@* "}"`
}
```

### 6.2 AST Node Types

```go
// internal/parser/ast.go
package parser

type AST struct {
    Config       *Config
    Entities     map[string]*Entity
    Endpoints    []*Endpoint
    Workflows    map[string]*Workflow
    Jobs         map[string]*Job
    Integrations map[string]*Integration
    Events       map[string]*Event
    Functions    map[string]*Function
}

type Entity struct {
    Name        string
    Description string
    Fields      []*Field
    Indexes     []*Index
    Position    Position
}

type Field struct {
    Name        string
    Type        Type
    Required    bool
    Unique      bool
    Primary     bool
    Auto        bool
    Default     *Expression
    Reference   *EntityRef
    Position    Position
}

type Type interface {
    TypeName() string
}

type PrimitiveType struct {
    Name      string
    Precision int
    Scale     int
}

type ListType struct {
    ElementType Type
}

type RefType struct {
    EntityName string
}

type EnumType struct {
    Values []string
}
```

### 6.3 Error Handling

The parser provides detailed error messages with line numbers and suggestions for common mistakes.

```go
// internal/parser/errors.go
package parser

type ParseError struct {
    File       string
    Line       int
    Column     int
    Message    string
    Suggestion string
    Context    string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}

func (e *ParseError) Pretty() string {
    var b strings.Builder
    
    b.WriteString(fmt.Sprintf("Error in %s at line %d:\n", e.File, e.Line))
    b.WriteString(fmt.Sprintf("  %s\n", e.Context))
    b.WriteString(fmt.Sprintf("  %s^\n", strings.Repeat(" ", e.Column-1)))
    b.WriteString(fmt.Sprintf("  %s\n", e.Message))
    
    if e.Suggestion != "" {
        b.WriteString(fmt.Sprintf("\n  Suggestion: %s\n", e.Suggestion))
    }
    
    return b.String()
}

// Common error suggestions
var suggestions = map[string]string{
    "unexpected token": "Check for missing commas or brackets",
    "unknown type":     "Valid types: string, text, integer, decimal, boolean, timestamp, uuid, json",
    "undefined entity": "Make sure the entity is declared in a .codeai file",
}
```

---

## 7. Standard Library

### 7.1 Built-in Functions

| Category | Function | Description |
|----------|----------|-------------|
| String | `length(s)` | Returns string length |
| String | `upper(s)` / `lower(s)` | Case conversion |
| String | `trim(s)` / `trim_left(s)` / `trim_right(s)` | Whitespace removal |
| String | `split(s, delim)` | Split string into list |
| String | `join(list, delim)` | Join list into string |
| String | `replace(s, old, new)` | Replace occurrences |
| String | `contains(s, substr)` | Check if contains substring |
| String | `starts_with(s, prefix)` / `ends_with(s, suffix)` | Prefix/suffix check |
| Math | `abs(n)` / `ceil(n)` / `floor(n)` / `round(n)` | Basic math operations |
| Math | `min(a, b)` / `max(a, b)` | Min/max values |
| Math | `sum(list)` / `avg(list)` | Aggregations |
| DateTime | `now()` | Current timestamp |
| DateTime | `today()` | Current date |
| DateTime | `format_date(d, fmt)` | Format date to string |
| DateTime | `parse_date(s, fmt)` | Parse string to date |
| DateTime | `add_days(d, n)` / `add_hours(d, n)` | Date arithmetic |
| DateTime | `diff_days(d1, d2)` | Difference between dates |
| List | `count(list)` | Number of items |
| List | `first(list)` / `last(list)` | First/last item |
| List | `map(list, expr)` | Transform items |
| List | `filter(list, expr)` | Filter items |
| List | `find(list, expr)` | Find first match |
| List | `sort(list, field)` | Sort items |
| List | `unique(list)` | Remove duplicates |
| Util | `uuid()` | Generate UUID |
| Util | `hash(s)` | SHA-256 hash |
| Util | `random(min, max)` | Random integer |
| Util | `env(name, default)` | Environment variable |

### 7.2 Type Coercion Rules

CodeAI performs automatic type coercion in safe, predictable ways to reduce LLM errors.

| From | To | Rule |
|------|-----|------|
| `integer` | `decimal` | Automatic, lossless |
| `integer` | `string` | Automatic via interpolation |
| `decimal` | `string` | Automatic via interpolation |
| `boolean` | `string` | "true" or "false" |
| `timestamp` | `date` | Truncates time portion |
| `date` | `timestamp` | Adds midnight time |
| `string` | `integer` | Explicit `parse_int()` required |
| `string` | `decimal` | Explicit `parse_decimal()` required |
| `any` | `json` | Automatic serialization |
| `json` | `specific` | Explicit casting required |

---

## 8. LLM Integration Guidelines

### 8.1 Prompt Engineering for CodeAI Generation

To maximize LLM code generation accuracy, use structured prompts that clearly specify the desired CodeAI constructs.

```
System Prompt Template:
---
You are a CodeAI developer. CodeAI is a declarative language for backend applications.

Key constructs:
- entity: Data models with typed fields
- endpoint: HTTP API routes
- workflow: Multi-step processes
- job: Scheduled tasks
- integration: External service connections
- event: Message definitions

Rules:
1. Always declare entities before referencing them
2. Use built-in types: string, text, integer, decimal, boolean, timestamp, uuid, json
3. Field modifiers: required, optional, unique, primary, auto, searchable
4. Auth options: required, optional, public
5. Return types: single entity, list(Entity), paginated(Entity)

Generate only valid CodeAI code without explanations.
---
```

### 8.2 Error Recovery Patterns

The parser implements error recovery to handle common LLM mistakes gracefully.

| LLM Mistake | Parser Recovery |
|-------------|-----------------|
| Missing trailing comma | Automatically inserted |
| Extra trailing comma | Ignored |
| Inconsistent indentation | Whitespace normalized |
| "string" vs string type | Quotes stripped from type names |
| camelCase vs snake_case | Both accepted, normalized to snake_case |
| Missing description | Empty string used |
| Typo in keyword (endpint) | Fuzzy matching with suggestions |
| Wrong bracket type { vs [ | Context-aware correction |

### 8.3 Validation Feedback Format

When validation fails, the runtime provides LLM-friendly error messages that can be fed back for correction.

```json
{
    "errors": [
        {
            "type": "undefined_reference",
            "location": "endpoint POST /orders line 15",
            "message": "Entity 'Prodcut' is not defined",
            "suggestion": "Did you mean 'Product'?",
            "fix": "Replace 'Prodcut' with 'Product'"
        }
    ],
    "context": {
        "defined_entities": ["User", "Product", "Order", "Category"],
        "defined_integrations": ["PaymentGateway", "ShippingService"]
    }
}
```

---

## 9. Implementation Phases

### Phase 1: Foundation (Weeks 1-4)

Establish core infrastructure and basic parsing capabilities.

- Set up Go project structure with modules
- Implement Participle grammar for config, entity, and basic endpoint
- Build AST structures and validator
- Create PostgreSQL database module with auto-migration
- Implement basic HTTP server with Chi router
- CLI with run and validate commands

**Deliverable: Working CRUD API generation from CodeAI source.**

### Phase 2: Core Features (Weeks 5-8)

Add authentication, validation, and query capabilities.

- JWT authentication module
- Role-based access control
- Input validation with detailed errors
- Query language parser and executor
- Pagination and filtering support
- Redis cache module

**Deliverable: Production-ready API with auth and caching.**

### Phase 3: Workflows and Jobs (Weeks 9-12)

Implement background processing and event-driven features.

- Workflow engine with state persistence
- Compensation (rollback) handling
- Job scheduler with Asynq
- Event emission and subscription
- Webhook publisher
- Email notifications

**Deliverable: Complete workflow automation capability.**

### Phase 4: Integrations and Polish (Weeks 13-16)

Add external integrations and production hardening.

- Integration module with circuit breaker
- Retry policies with exponential backoff
- OpenAPI specification generation
- Structured logging with slog
- Prometheus metrics
- Health check endpoints
- Graceful shutdown

**Deliverable: Production-ready v1.0 release.**

---

## 10. Testing Strategy

### 10.1 Test Categories

| Category | Scope | Tools |
|----------|-------|-------|
| Unit Tests | Individual functions, parsers, validators | Go testing, testify |
| Integration Tests | Module interactions, database operations | testcontainers-go |
| End-to-End Tests | Full application lifecycle | Custom test harness |
| LLM Accuracy Tests | Code generation correctness | Benchmark suite |
| Performance Tests | Throughput, latency, resource usage | k6, pprof |

### 10.2 LLM Accuracy Benchmarks

A key success metric is LLM code generation accuracy. The benchmark suite tests various LLMs against standardized prompts.

```go
// internal/testing/llm_benchmark.go
type Benchmark struct {
    Name        string
    Prompt      string
    Expected    string  // Expected CodeAI output pattern
    Validators  []Validator
}

var benchmarks = []Benchmark{
    {
        Name: "simple_crud_entity",
        Prompt: "Create a User entity with id, email, name, and created_at",
        Validators: []Validator{
            HasEntity("User"),
            HasField("User", "id", "uuid", "primary", "auto"),
            HasField("User", "email", "string", "required", "unique"),
            HasField("User", "name", "string", "required"),
            HasField("User", "created_at", "timestamp", "auto"),
        },
    },
    {
        Name: "endpoint_with_auth",
        Prompt: "Create a GET endpoint for /users that requires authentication and returns a paginated list",
        Validators: []Validator{
            HasEndpoint("GET", "/users"),
            HasAuth("required"),
            ReturnsType("paginated(User)"),
        },
    },
}
```

---

## 11. Deployment Model

### 11.1 Single Binary Distribution

Like Go itself, CodeAI compiles to a single binary with no external dependencies. Users can download and run immediately.

```bash
# Download
curl -LO https://github.com/codeai/codeai/releases/latest/codeai-linux-amd64
chmod +x codeai-linux-amd64
mv codeai-linux-amd64 /usr/local/bin/codeai

# Run
codeai run ./myapp/
```

### 11.2 CLI Commands

| Command | Description |
|---------|-------------|
| `codeai run <dir>` | Run application from source directory |
| `codeai build <dir>` | Validate and compile (dry run) |
| `codeai validate <dir>` | Check syntax and references only |
| `codeai migrate <dir>` | Run database migrations |
| `codeai openapi <dir>` | Generate OpenAPI specification |
| `codeai version` | Show version information |

### 11.3 Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o codeai ./cmd/codeai

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/codeai /usr/local/bin/
ENTRYPOINT ["codeai", "run"]
CMD ["/app"]
```

```bash
# Usage
docker build -t my-codeai-app .
docker run -v ./myapp:/app -p 8080:8080 my-codeai-app
```

### 11.4 Configuration via Environment

| Variable | Description | Default |
|----------|-------------|---------|
| `CODEAI_PORT` | HTTP server port | 8080 |
| `CODEAI_HOST` | HTTP server host | 0.0.0.0 |
| `CODEAI_LOG_LEVEL` | Logging level (debug/info/warn/error) | info |
| `CODEAI_LOG_FORMAT` | Log format (json/text) | json |
| `DATABASE_URL` | Database connection string | required |
| `REDIS_URL` | Redis connection string | optional |
| `JWT_SECRET` | JWT signing secret | required if auth enabled |
| `JWT_ISSUER` | JWT issuer claim | optional |

---

## 12. Performance Considerations

### 12.1 Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Cold start time | < 500ms | Parse, validate, connect to DB |
| Request latency (p50) | < 5ms | Simple CRUD operations |
| Request latency (p99) | < 50ms | Complex queries with joins |
| Throughput | > 10,000 req/s | Per core, simple endpoints |
| Memory usage | < 100MB | Base runtime, no cache |
| Binary size | < 50MB | Compressed, single binary |

### 12.2 Optimization Strategies

- Connection pooling for database and Redis connections
- Prepared statement caching for frequently executed queries
- Response caching with configurable TTL
- Lazy loading of related entities
- Goroutine pooling for workflow execution
- Zero-allocation JSON serialization where possible

---

## 13. Security Model

### 13.1 Security by Default

CodeAI eliminates entire classes of vulnerabilities by design. The LLM cannot generate insecure code because the language does not expose insecure primitives.

| Vulnerability | Prevention |
|---------------|------------|
| SQL Injection | All queries use parameterized statements; raw SQL not exposed |
| XSS | All output is automatically escaped; no raw HTML output |
| CSRF | Built-in CSRF tokens for state-changing operations |
| Authentication Bypass | Auth is declarative; endpoints are secure by default |
| Sensitive Data Exposure | Automatic field-level redaction in logs |
| Insecure Deserialization | Strict type checking on all inputs |
| Broken Access Control | RBAC enforced at runtime, not application code |

### 13.2 Secret Management

Secrets are never stored in CodeAI source files. The `env()` function retrieves values from environment variables at runtime.

```codeai
# Correct: secrets from environment
config {
    database: postgres {
        connection: env(DATABASE_URL)
    }
    auth: jwt {
        secret: env(JWT_SECRET)
    }
}

# The runtime validates that required env vars are set at startup
```

---

## 14. Appendices

### Appendix A: Complete Grammar (EBNF)

```ebnf
program        = { declaration } ;
declaration    = config | entity | endpoint | workflow | job | integration | event | function ;

config         = "config" "{" { config_setting } "}" ;
config_setting = IDENT ":" config_value ;
config_value   = STRING | NUMBER | BOOL | config_block | config_list ;

entity         = "entity" IDENT "{" [ description ] { field } { index } "}" ;
field          = IDENT ":" type { "," modifier } ;
type           = IDENT [ "(" type_args ")" ] ;
modifier       = IDENT [ "(" expression ")" ] ;

endpoint       = "endpoint" METHOD PATH "{" endpoint_body "}" ;
endpoint_body  = { description | auth | roles | params | returns | filter | action } ;

workflow       = "workflow" IDENT "{" workflow_body "}" ;
workflow_body  = { description | trigger | steps | on_complete | on_fail } ;

job            = "job" IDENT "{" job_body "}" ;
job_body       = { description | schedule | steps | timeout | retry | on_fail } ;
```

### Appendix B: Go Dependencies

```go
module github.com/codeai/codeai

go 1.22

require (
    github.com/alecthomas/participle/v2 v2.1.1
    github.com/go-chi/chi/v5 v5.0.12
    github.com/go-chi/cors v1.2.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/hibiken/asynq v0.24.1
    github.com/jackc/pgx/v5 v5.5.3
    github.com/redis/go-redis/v9 v9.5.1
    github.com/spf13/viper v1.18.2
    github.com/stretchr/testify v1.8.4
    go.mongodb.org/mongo-driver v1.14.0
)
```

### Appendix C: Example Application

```codeai
# inventory-service/app.codeai

config {
    name: "inventory-service"
    version: "1.0.0"
    
    database: postgres {
        pool_size: 20
    }
    
    cache: redis {
        ttl: 5m
    }
    
    auth: jwt {
        issuer: env(JWT_ISSUER)
    }
}

entity Category {
    id: uuid, primary, auto
    name: string, required, unique
    description: text, optional
}

entity Product {
    id: uuid, primary, auto
    sku: string, required, unique
    name: string, required, searchable
    description: text, optional
    price: decimal(10,2), required
    quantity: integer, default(0)
    category: ref(Category), required
    created_at: timestamp, auto
    updated_at: timestamp, auto_update
}

endpoint GET /products {
    auth: optional
    cache: 1m
    
    query {
        category: uuid, optional
        search: string, optional
        page: integer, default(1)
        limit: integer, default(20)
    }
    
    returns: paginated(Product)
}

endpoint POST /products {
    auth: required
    roles: [admin]
    
    body {
        sku: string, required
        name: string, required
        price: decimal, required
        category: uuid, required
    }
    
    returns: Product
    on_success: emit(ProductCreated)
}

event ProductCreated {
    payload {
        product_id: uuid
        sku: string
        name: string
    }
}

job LowStockAlert {
    schedule: every 1h
    
    action: {
        let low_stock = select Product where quantity < 10
        
        if count(low_stock) > 0 {
            send: slack("#inventory") {
                message: "{count(low_stock)} products are low on stock"
            }
        }
    }
}
```

---

*— End of Document —*
