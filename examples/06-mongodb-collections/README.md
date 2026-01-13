# Example 06: MongoDB Collections

This example demonstrates MongoDB collection definitions in CodeAI DSL, including embedded documents, arrays, and various index types.

## Overview

| Aspect | Description |
|--------|-------------|
| **Focus** | MongoDB document modeling |
| **Database** | MongoDB only |
| **Key Concepts** | Collections, embedded documents, arrays, indexes |
| **Difficulty** | Intermediate |

## What You'll Learn

- MongoDB collection definitions with `database mongodb { }`
- Field types specific to MongoDB (`objectid`, `embedded`, `array`)
- Embedded documents for denormalized data
- Nested embedded documents
- Array fields for lists and flexible data
- MongoDB index types (standard, unique, text, geospatial)

## Collections in This Example

| Collection | Purpose | Key Features |
|------------|---------|--------------|
| `User` | User accounts | Basic fields, unique indexes |
| `Address` | User addresses | Embedded location (geospatial), GeoJSON |
| `Product` | E-commerce products | Nested embedded docs, text search |
| `Order` | Customer orders | Embedded addresses, array of items |
| `Review` | Product reviews | Text search on title and body |

## Quick Start

### 1. Prerequisites

```bash
# Start MongoDB locally
mongod --dbpath /data/db

# Or with Docker
docker run -d -p 27017:27017 --name mongodb mongo:6
```

### 2. Review the DSL

```bash
cat mongodb-collections.cai
```

### 3. Validate the DSL

```bash
codeai validate mongodb-collections.cai
```

### 4. Generate and Run

```bash
codeai generate mongodb-collections.cai
codeai run
```

## DSL Highlights

### Basic Collection with ObjectID

```cai
database mongodb {
    collection User {
        _id: objectid, primary, auto
        email: string, required, unique
        username: string, required, unique
        password_hash: string, required

        created_at: date, auto
        updated_at: date, auto

        indexes {
            index: [email] unique
            index: [username] unique
        }
    }
}
```

### Embedded Documents

Embed related data directly in the parent document:

```cai
collection Address {
    _id: objectid, primary, auto
    user_id: objectid, required

    street: string, required
    city: string, required

    // Embedded GeoJSON location
    location: embedded {
        type: string, required, default("Point")
        coordinates: array(double), required
    }

    indexes {
        index: [user_id]
        index: [location] geospatial
    }
}
```

### Nested Embedded Documents

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

### Array Fields

```cai
collection Product {
    // Array of strings
    categories: array(string), optional
    tags: array(string), optional

    // Array of flexible objects
    items: array(object), required
}
```

### Text Search Index

```cai
indexes {
    index: [name, description] text
}
```

### Geospatial Index

```cai
indexes {
    index: [location] geospatial
}
```

## MongoDB Field Types Reference

| Type | Description | Example |
|------|-------------|---------|
| `objectid` | MongoDB ObjectID | `_id: objectid, primary, auto` |
| `string` | Text field | `name: string, required` |
| `int` | 32-bit integer | `count: int, default(0)` |
| `double` | 64-bit floating point | `price: double, required` |
| `bool` | Boolean value | `active: bool, default(true)` |
| `date` | Date/timestamp | `created_at: date, auto` |
| `array(T)` | Array of type T | `tags: array(string)` |
| `embedded { }` | Nested document | `address: embedded { ... }` |
| `object` | Flexible object | `payload: object, optional` |

## Index Types Reference

| Type | Syntax | Use Case |
|------|--------|----------|
| Standard | `index: [field]` | Equality and range queries |
| Unique | `index: [field] unique` | Enforce uniqueness |
| Compound | `index: [field1, field2]` | Multi-field queries |
| Text | `index: [field] text` | Full-text search |
| Geospatial | `index: [field] geospatial` | Location-based queries |

## Testing the API

### Create a User

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "username": "alice",
    "password_hash": "hashed_password",
    "first_name": "Alice",
    "last_name": "Smith"
  }'
```

### Create a Product

```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "PROD-001",
    "name": "Wireless Mouse",
    "description": "Ergonomic wireless mouse",
    "price": 29.99,
    "categories": ["electronics", "accessories"],
    "tags": ["wireless", "ergonomic"],
    "attributes": {
      "color": "black",
      "dimensions": {
        "length": 12.0,
        "width": 6.5,
        "height": 4.0
      }
    }
  }'
```

### Text Search

```bash
curl "http://localhost:8080/api/v1/products?search=wireless+mouse"
```

### Geospatial Query

```bash
curl "http://localhost:8080/api/v1/addresses?near=-73.97,40.77&maxDistance=5000"
```

## When to Use This Pattern

**Use MongoDB collections when:**

- Schema varies between documents
- Embedded documents reduce JOINs
- High write throughput is needed
- Geospatial queries are required
- Full-text search without external tools

**Consider PostgreSQL instead when:**

- Strong referential integrity needed
- Complex multi-table JOINs required
- ACID transactions are critical
- Structured reporting is primary use case

## Related Resources

- [Database Guide](../../docs/03-database.md) - Full database documentation
- [DSL Cheatsheet](../../docs/dsl_cheatsheet.md) - Quick syntax reference
- [Mixed Databases Example](../07-mixed-databases/) - Using PostgreSQL + MongoDB together
- [Troubleshooting](../../docs/troubleshooting.md#mongodb-issues) - Common MongoDB issues
