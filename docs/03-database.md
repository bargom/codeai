# CodeAI Database Guide

This guide covers database configuration and usage in CodeAI, including PostgreSQL and MongoDB support.

---

## Table of Contents

- [Overview](#overview)
- [PostgreSQL](#postgresql)
- [MongoDB](#mongodb)
- [MongoDB vs PostgreSQL](#mongodb-vs-postgresql)
- [Configuration](#configuration)
- [Collection Definition Syntax](#collection-definition-syntax)
- [Index Types](#index-types)
- [Transaction Support](#transaction-support)
- [Migration Strategy](#migration-strategy)
- [Performance Tuning](#performance-tuning)

---

## Overview

CodeAI supports two database backends:

| Feature | PostgreSQL | MongoDB |
|---------|-----------|---------|
| Data Model | Relational tables | Document collections |
| Schema | Strict, predefined | Flexible, schema-optional |
| Relationships | Foreign keys, JOINs | Embedded documents, references |
| Transactions | Full ACID | ACID (with replica set) |
| Query Language | SQL-based DSL | Document-based DSL |
| Best For | Structured data, complex queries | Flexible schemas, rapid iteration |

---

## PostgreSQL

PostgreSQL is the default database for CodeAI. It's ideal for:

- Structured, relational data
- Complex queries with JOINs
- Strong data integrity requirements
- ACID transactions
- Financial or regulated data

### PostgreSQL Configuration

```cai
config {
    database_type: "postgres"
    database_host: "localhost"
    database_port: 5432
    database_name: "codeai"
    database_user: "codeai"
    database_password: "secret"
    database_ssl_mode: "disable"
}
```

### PostgreSQL Model Definition

```cai
model User {
    description: "User accounts"

    id: uuid, primary, auto
    email: string, required, unique
    username: string, required, unique
    password_hash: string, required

    created_at: timestamp, auto
    updated_at: timestamp, auto_update

    has_many posts
    has_many comments

    indexes {
        unique: [email]
        unique: [username]
        index: [created_at]
    }
}
```

### PostgreSQL Field Types

| Type | Description | Example |
|------|-------------|---------|
| `uuid` | UUID v4 | `id: uuid, primary, auto` |
| `string` | VARCHAR(255) | `name: string, required` |
| `text` | Unlimited text | `content: text, optional` |
| `integer` | 64-bit integer | `count: integer, default(0)` |
| `decimal(p,s)` | Fixed precision | `price: decimal(10,2)` |
| `boolean` | True/false | `active: boolean, default(true)` |
| `timestamp` | Date + time + TZ | `created_at: timestamp, auto` |
| `date` | Date only | `birth_date: date` |
| `time` | Time only | `start_time: time` |
| `json` | JSON data | `metadata: json, optional` |

---

## MongoDB

MongoDB support enables document-based data modeling in CodeAI. It's ideal for:

- Flexible or evolving schemas
- Embedded documents and arrays
- High-throughput writes (logs, events, analytics)
- Geospatial data
- Content management systems

### MongoDB Configuration

```cai
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai"
}
```

#### Connection URI Formats

| Environment | URI Format |
|-------------|-----------|
| Local | `mongodb://localhost:27017` |
| With Auth | `mongodb://user:pass@localhost:27017` |
| Replica Set | `mongodb://host1:27017,host2:27017,host3:27017/?replicaSet=rs0` |
| Atlas | `mongodb+srv://user:pass@cluster.mongodb.net/dbname` |

### MongoDB Collection Definition

```cai
database mongodb {
    collection Product {
        description: "E-commerce products"

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
        updated_at: date, auto

        indexes {
            index: [sku] unique
            index: [name] text
            index: [categories]
            index: [price]
        }
    }
}
```

### MongoDB Field Types

| Type | Description | Example |
|------|-------------|---------|
| `objectid` | MongoDB ObjectID | `_id: objectid, primary, auto` |
| `string` | Text field | `name: string, required` |
| `int` | 32-bit integer | `count: int, default(0)` |
| `double` | 64-bit float | `price: double, required` |
| `bool` | Boolean | `active: bool, default(true)` |
| `date` | Date/timestamp | `created_at: date, auto` |
| `array(T)` | Array of type T | `tags: array(string)` |
| `embedded { }` | Nested document | `address: embedded { ... }` |
| `object` | Flexible object | `payload: object, optional` |

### MongoDB Field Modifiers

| Modifier | Description | Example |
|----------|-------------|---------|
| `primary` | Primary key field | `_id: objectid, primary` |
| `auto` | Auto-generated | `_id: objectid, auto` |
| `required` | Cannot be null | `name: string, required` |
| `optional` | Can be null | `bio: string, optional` |
| `unique` | Must be unique | `email: string, unique` |
| `default(val)` | Default value | `status: string, default("active")` |

---

## MongoDB vs PostgreSQL

### When to Use PostgreSQL

| Use Case | Why PostgreSQL |
|----------|----------------|
| **User accounts & auth** | ACID transactions, referential integrity |
| **Financial data** | Decimal precision, transaction safety |
| **Inventory management** | Consistent counts, foreign key constraints |
| **Multi-table joins** | Efficient SQL JOINs |
| **Reporting** | Complex aggregations, window functions |
| **Regulatory compliance** | Schema enforcement, audit trails |

### When to Use MongoDB

| Use Case | Why MongoDB |
|----------|-------------|
| **Activity logs** | High write throughput, flexible schema |
| **Analytics events** | Variable event properties |
| **Content management** | Varying content structures |
| **Product catalogs** | Different attributes per product |
| **Real-time features** | Fast reads, embedded data |
| **Geospatial data** | Native 2dsphere indexes |
| **Caching/sessions** | TTL indexes, fast key-value access |

### Comparison Table

| Aspect | PostgreSQL | MongoDB |
|--------|-----------|---------|
| Schema | Rigid, predefined | Flexible, evolving |
| Relationships | Foreign keys + JOINs | Embedded docs, manual refs |
| Transactions | Full ACID always | ACID with replica set |
| Scaling | Vertical (read replicas) | Horizontal (sharding) |
| Query Language | SQL-based | Document-based |
| Best For | Structured, relational | Flexible, document-oriented |
| Learning Curve | Familiar SQL syntax | Document thinking required |

### Mixed Database Architecture

CodeAI supports using both databases together:

```cai
config {
    // Primary database (PostgreSQL)
    database_type: "postgres"
    database_host: "localhost"
    database_name: "codeai"

    // Secondary database (MongoDB)
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai_docs"
}

// Structured data in PostgreSQL
model User {
    id: uuid, primary, auto
    email: string, required, unique
    // ...
}

// Flexible data in MongoDB
database mongodb {
    collection ActivityLog {
        _id: objectid, primary, auto
        user_id: string, required
        event_type: string, required
        payload: object, optional
        timestamp: date, auto
    }
}
```

---

## Collection Definition Syntax

### Basic Collection Structure

```cai
database mongodb {
    collection CollectionName {
        description: "Description of the collection"

        // Fields
        _id: objectid, primary, auto
        field_name: type, modifiers

        // Indexes
        indexes {
            index: [field] type
        }
    }
}
```

### Embedded Documents

Embed related data directly in the parent document:

```cai
collection Order {
    _id: objectid, primary, auto
    order_number: string, required, unique

    // Embedded shipping address
    shipping_address: embedded {
        street: string, required
        city: string, required
        state: string, required
        postal_code: string, required
        country: string, required
    }

    // Array of embedded line items
    items: array(object), required
}
```

### Nested Embedded Documents

Embed documents within embedded documents:

```cai
collection Product {
    _id: objectid, primary, auto
    name: string, required

    attributes: embedded {
        color: string, optional
        dimensions: embedded {
            length: double, optional
            width: double, optional
            height: double, optional
        }
    }
}
```

### Arrays

Arrays can contain primitives or objects:

```cai
collection Post {
    _id: objectid, primary, auto
    title: string, required

    // Array of strings
    tags: array(string), optional

    // Array of numbers
    ratings: array(int), optional

    // Array of objects (flexible structure)
    comments: array(object), optional
}
```

---

## Index Types

### Standard Index

Single-field index for equality queries and sorting:

```cai
indexes {
    index: [email]
    index: [created_at]
}
```

### Unique Index

Enforce uniqueness constraint:

```cai
indexes {
    index: [email] unique
    index: [sku] unique
}
```

### Compound Index

Multi-field index for queries on multiple fields:

```cai
indexes {
    index: [user_id, created_at]
    index: [status, priority]
}
```

### Text Index

Full-text search on string fields:

```cai
indexes {
    index: [title, body] text
    index: [name, description] text
}
```

Use in queries:
```javascript
// Runtime usage (generated code)
db.collection.find({ $text: { $search: "search terms" } })
```

### Geospatial Index

2dsphere index for location queries:

```cai
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
```

Use for:
- Finding locations within a radius
- Sorting by distance
- Geo-contained queries

### Index Summary Table

| Index Type | Syntax | Use Case |
|------------|--------|----------|
| Standard | `index: [field]` | Equality, range queries |
| Unique | `index: [field] unique` | Enforce uniqueness |
| Compound | `index: [field1, field2]` | Multi-field queries |
| Text | `index: [field] text` | Full-text search |
| Geospatial | `index: [field] geospatial` | Location queries |

---

## Transaction Support

### PostgreSQL Transactions

PostgreSQL provides full ACID transactions by default:

```cai
workflow TransferFunds {
    trigger: TransferRequested

    steps {
        debit {
            update Account
            where id = trigger.from_account
            set balance = balance - trigger.amount
        }

        credit {
            update Account
            where id = trigger.to_account
            set balance = balance + trigger.amount
        }
    }

    // Automatic rollback on failure
    on_fail: rollback
}
```

### MongoDB Transactions

MongoDB transactions require a replica set:

**Requirements:**
- MongoDB 4.0+ (single document transactions)
- MongoDB 4.2+ (multi-document transactions)
- Replica set deployment (even single-node)

**Limitations:**
- Slightly higher latency than PostgreSQL
- 16MB document size limit applies
- Transaction timeout (default 60s)

**Local Development with Transactions:**

```bash
# Start MongoDB as a replica set for transaction support
mongod --replSet rs0

# Initialize the replica set
mongosh --eval "rs.initiate()"
```

**Configuration for transactions:**

```cai
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017/?replicaSet=rs0"
    mongodb_database: "codeai"
}
```

---

## Migration Strategy

### PostgreSQL Migrations

PostgreSQL uses explicit migrations:

```bash
# Generate migration
codeai migrate generate add_users_table

# Run migrations
codeai migrate up

# Rollback
codeai migrate down
```

### MongoDB Migrations

MongoDB collections are created automatically. Schema changes use a different approach:

**Auto-Creation Strategy (Default):**
- Collections created on first insert
- Indexes created at startup
- No explicit migration files needed

**Explicit Migration Strategy:**

```cai
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai"
    mongodb_auto_create: false  // Disable auto-creation
}
```

With explicit migrations:

```bash
# Generate MongoDB migration
codeai migrate generate --mongodb add_products_indexes

# Run MongoDB migrations
codeai migrate up --mongodb
```

### Schema Evolution Best Practices

| Scenario | PostgreSQL | MongoDB |
|----------|-----------|---------|
| Add field | ALTER TABLE migration | Just add to documents |
| Remove field | ALTER TABLE migration | Stop writing, lazy cleanup |
| Rename field | ALTER TABLE migration | $rename or application-level |
| Change type | Migration with data transform | Application-level handling |
| Add index | CREATE INDEX migration | ensureIndex at startup |

---

## Performance Tuning

### MongoDB Connection Pooling

```cai
config {
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai"
    mongodb_pool_size: 100        // Max connections
    mongodb_min_pool_size: 10     // Min idle connections
    mongodb_max_idle_time: "5m"   // Close idle connections after
}
```

### Index Optimization

**Query Analysis:**
```javascript
// Explain query execution
db.collection.find({ email: "test@example.com" }).explain("executionStats")

// Check index usage
db.collection.aggregate([{ $indexStats: {} }])
```

**Common Index Patterns:**

| Query Pattern | Index Strategy |
|---------------|----------------|
| `{ status: "active" }` | Single field index |
| `{ user_id: X, created_at: -1 }` | Compound index |
| `{ $text: { $search: "..." } }` | Text index |
| `{ location: { $near: [...] } }` | 2dsphere index |
| High-write collection | Minimal indexes |

### Read/Write Concerns

```cai
config {
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "codeai"

    // Read preference
    mongodb_read_preference: "primaryPreferred"  // or "secondary", "nearest"

    // Write concern
    mongodb_write_concern: "majority"  // or "1", "0"

    // Read concern
    mongodb_read_concern: "local"  // or "majority", "linearizable"
}
```

| Setting | Value | Use Case |
|---------|-------|----------|
| Read Preference | `primary` | Strong consistency |
| Read Preference | `secondary` | Read scaling, slight lag OK |
| Write Concern | `majority` | Durability guarantee |
| Write Concern | `1` | Fast writes, primary only |
| Read Concern | `majority` | Read your writes |

---

## Related Documentation

- [DSL Language Spec](./dsl_language_spec.md) - Complete syntax reference
- [DSL Cheatsheet](./dsl_cheatsheet.md) - Quick syntax reference
- [Troubleshooting](./troubleshooting.md#mongodb-issues) - Common issues and solutions
- [Examples: MongoDB Collections](../examples/06-mongodb-collections/) - MongoDB-only example
- [Examples: Mixed Databases](../examples/07-mixed-databases/) - PostgreSQL + MongoDB example
