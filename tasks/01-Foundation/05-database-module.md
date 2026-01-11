# Task 005: PostgreSQL Database Module

## Overview
Implement the PostgreSQL database module providing connection pooling, query execution, entity CRUD operations, and automatic schema migrations.

## Phase
Phase 1: Foundation

## Priority
Critical - Database is required for entity persistence.

## Dependencies
- Task 001: Project Structure Setup
- Task 003: AST Node Types and Transformation

## Description
Create the database module that handles all PostgreSQL interactions. The module must provide safe, parameterized queries, automatic connection pooling, and schema migration from entity definitions.

## Detailed Requirements

### 1. Module Interface (internal/modules/database/module.go)

```go
package database

import (
    "context"
    "time"

    "github.com/codeai/codeai/internal/parser"
)

// Module is the interface for database operations
type Module interface {
    // Lifecycle
    Name() string
    Initialize(config *Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus

    // Query execution
    Query(ctx context.Context, query string, args ...any) ([]Record, error)
    QueryOne(ctx context.Context, query string, args ...any) (Record, error)
    Execute(ctx context.Context, query string, args ...any) (Result, error)

    // Entity operations
    FindByID(ctx context.Context, entity string, id string) (Record, error)
    FindAll(ctx context.Context, entity string, opts QueryOptions) ([]Record, error)
    FindOne(ctx context.Context, entity string, opts QueryOptions) (Record, error)
    Count(ctx context.Context, entity string, opts QueryOptions) (int64, error)
    Create(ctx context.Context, entity string, data Record) (Record, error)
    Update(ctx context.Context, entity string, id string, data Record) (Record, error)
    Delete(ctx context.Context, entity string, id string) error
    SoftDelete(ctx context.Context, entity string, id string) error

    // Transactions
    Transaction(ctx context.Context, fn func(tx Transaction) error) error

    // Migrations
    RegisterEntity(entity *parser.Entity) error
    Migrate(ctx context.Context) error
    MigrationStatus(ctx context.Context) ([]Migration, error)
}

// Record represents a database row as a map
type Record map[string]any

// Result represents the result of an execute operation
type Result struct {
    RowsAffected int64
    LastInsertID string
}

// QueryOptions specifies options for find operations
type QueryOptions struct {
    Where     map[string]any
    WhereExpr string
    WhereArgs []any
    OrderBy   []OrderClause
    Limit     int
    Offset    int
    Include   []string // Related entities to load
}

// OrderClause specifies sorting
type OrderClause struct {
    Field     string
    Direction string // "asc" or "desc"
}

// Transaction represents a database transaction
type Transaction interface {
    Query(ctx context.Context, query string, args ...any) ([]Record, error)
    QueryOne(ctx context.Context, query string, args ...any) (Record, error)
    Execute(ctx context.Context, query string, args ...any) (Result, error)
    Commit() error
    Rollback() error
}

// Migration represents a migration record
type Migration struct {
    ID        int
    Name      string
    AppliedAt time.Time
    SQL       string
}

// HealthStatus represents module health
type HealthStatus struct {
    Status  string // healthy, degraded, unhealthy
    Message string
    Details map[string]any
}

// Config for database module
type Config struct {
    Type            string // postgres
    ConnectionString string
    PoolSize        int
    MinPoolSize     int
    MaxConnLifetime time.Duration
    MaxConnIdleTime time.Duration
}
```

### 2. PostgreSQL Implementation (internal/modules/database/postgres.go)

```go
package database

import (
    "context"
    "fmt"
    "strings"
    "sync"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "log/slog"

    "github.com/codeai/codeai/internal/parser"
)

type PostgresModule struct {
    pool     *pgxpool.Pool
    config   *Config
    entities map[string]*entityMeta
    logger   *slog.Logger
    mu       sync.RWMutex
}

type entityMeta struct {
    Entity     *parser.Entity
    TableName  string
    Columns    []string
    PrimaryKey string
}

func NewPostgresModule(config *Config) (*PostgresModule, error) {
    return &PostgresModule{
        config:   config,
        entities: make(map[string]*entityMeta),
        logger:   slog.Default().With("module", "postgres"),
    }, nil
}

func (m *PostgresModule) Name() string {
    return "database.postgres"
}

func (m *PostgresModule) Initialize(config *Config) error {
    m.config = config
    return nil
}

func (m *PostgresModule) Start(ctx context.Context) error {
    poolConfig, err := pgxpool.ParseConfig(m.config.ConnectionString)
    if err != nil {
        return fmt.Errorf("invalid connection string: %w", err)
    }

    // Configure pool
    poolConfig.MaxConns = int32(m.config.PoolSize)
    if m.config.PoolSize == 0 {
        poolConfig.MaxConns = 20
    }
    poolConfig.MinConns = int32(m.config.MinPoolSize)
    poolConfig.MaxConnLifetime = m.config.MaxConnLifetime
    poolConfig.MaxConnIdleTime = m.config.MaxConnIdleTime

    // Create pool
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return fmt.Errorf("failed to create pool: %w", err)
    }

    // Test connection
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return fmt.Errorf("failed to connect to database: %w", err)
    }

    m.pool = pool
    m.logger.Info("database connected", "pool_size", poolConfig.MaxConns)
    return nil
}

func (m *PostgresModule) Stop(ctx context.Context) error {
    if m.pool != nil {
        m.pool.Close()
    }
    return nil
}

func (m *PostgresModule) Health() HealthStatus {
    if m.pool == nil {
        return HealthStatus{Status: "unhealthy", Message: "pool not initialized"}
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := m.pool.Ping(ctx); err != nil {
        return HealthStatus{Status: "unhealthy", Message: err.Error()}
    }

    stats := m.pool.Stat()
    return HealthStatus{
        Status: "healthy",
        Details: map[string]any{
            "total_conns":   stats.TotalConns(),
            "idle_conns":    stats.IdleConns(),
            "acquired_conns": stats.AcquiredConns(),
        },
    }
}

// Query executes a query and returns multiple rows
func (m *PostgresModule) Query(ctx context.Context, query string, args ...any) ([]Record, error) {
    m.logger.Debug("executing query", "query", query, "args", args)

    rows, err := m.pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    return m.scanRows(rows)
}

// QueryOne executes a query and returns a single row
func (m *PostgresModule) QueryOne(ctx context.Context, query string, args ...any) (Record, error) {
    records, err := m.Query(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    if len(records) == 0 {
        return nil, ErrNotFound
    }
    return records[0], nil
}

// Execute executes a statement and returns the result
func (m *PostgresModule) Execute(ctx context.Context, query string, args ...any) (Result, error) {
    m.logger.Debug("executing statement", "query", query, "args", args)

    tag, err := m.pool.Exec(ctx, query, args...)
    if err != nil {
        return Result{}, fmt.Errorf("execute failed: %w", err)
    }

    return Result{
        RowsAffected: tag.RowsAffected(),
    }, nil
}

func (m *PostgresModule) scanRows(rows pgx.Rows) ([]Record, error) {
    var records []Record
    fields := rows.FieldDescriptions()

    for rows.Next() {
        values, err := rows.Values()
        if err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }

        record := make(Record)
        for i, field := range fields {
            record[string(field.Name)] = values[i]
        }
        records = append(records, record)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows error: %w", err)
    }

    return records, nil
}
```

### 3. Entity Operations

```go
// FindByID finds a single entity by primary key
func (m *PostgresModule) FindByID(ctx context.Context, entity string, id string) (Record, error) {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return nil, err
    }

    query := fmt.Sprintf(
        "SELECT * FROM %s WHERE %s = $1",
        meta.TableName,
        meta.PrimaryKey,
    )

    // Handle soft delete
    if meta.Entity.SoftDelete != nil {
        query += fmt.Sprintf(" AND %s IS NULL", meta.Entity.SoftDelete.Name)
    }

    return m.QueryOne(ctx, query, id)
}

// FindAll finds all entities matching the options
func (m *PostgresModule) FindAll(ctx context.Context, entity string, opts QueryOptions) ([]Record, error) {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return nil, err
    }

    query, args := m.buildSelectQuery(meta, opts)
    return m.Query(ctx, query, args...)
}

// Count returns the count of entities matching the options
func (m *PostgresModule) Count(ctx context.Context, entity string, opts QueryOptions) (int64, error) {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return 0, err
    }

    query, args := m.buildCountQuery(meta, opts)
    record, err := m.QueryOne(ctx, query, args...)
    if err != nil {
        return 0, err
    }

    return record["count"].(int64), nil
}

// Create inserts a new entity
func (m *PostgresModule) Create(ctx context.Context, entity string, data Record) (Record, error) {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return nil, err
    }

    // Generate UUID for auto fields
    for _, field := range meta.Entity.Fields {
        if field.Auto && field.Type.TypeName() == "uuid" {
            if _, exists := data[field.Name]; !exists {
                data[field.Name] = uuid.New().String()
            }
        }
        if field.Auto && field.Type.TypeName() == "timestamp" {
            if _, exists := data[field.Name]; !exists {
                data[field.Name] = time.Now()
            }
        }
    }

    columns, placeholders, values := m.buildInsertParts(meta, data)

    query := fmt.Sprintf(
        "INSERT INTO %s (%s) VALUES (%s) RETURNING *",
        meta.TableName,
        strings.Join(columns, ", "),
        strings.Join(placeholders, ", "),
    )

    return m.QueryOne(ctx, query, values...)
}

// Update modifies an existing entity
func (m *PostgresModule) Update(ctx context.Context, entity string, id string, data Record) (Record, error) {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return nil, err
    }

    // Handle auto_update timestamps
    for _, field := range meta.Entity.Fields {
        if field.AutoUpdate && field.Type.TypeName() == "timestamp" {
            data[field.Name] = time.Now()
        }
    }

    setParts, values := m.buildUpdateParts(meta, data)
    values = append(values, id)

    query := fmt.Sprintf(
        "UPDATE %s SET %s WHERE %s = $%d RETURNING *",
        meta.TableName,
        strings.Join(setParts, ", "),
        meta.PrimaryKey,
        len(values),
    )

    return m.QueryOne(ctx, query, values...)
}

// Delete removes an entity permanently
func (m *PostgresModule) Delete(ctx context.Context, entity string, id string) error {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return err
    }

    query := fmt.Sprintf(
        "DELETE FROM %s WHERE %s = $1",
        meta.TableName,
        meta.PrimaryKey,
    )

    result, err := m.Execute(ctx, query, id)
    if err != nil {
        return err
    }

    if result.RowsAffected == 0 {
        return ErrNotFound
    }

    return nil
}

// SoftDelete marks an entity as deleted
func (m *PostgresModule) SoftDelete(ctx context.Context, entity string, id string) error {
    meta, err := m.getEntityMeta(entity)
    if err != nil {
        return err
    }

    if meta.Entity.SoftDelete == nil {
        return fmt.Errorf("entity %s does not support soft delete", entity)
    }

    query := fmt.Sprintf(
        "UPDATE %s SET %s = $1 WHERE %s = $2",
        meta.TableName,
        meta.Entity.SoftDelete.Name,
        meta.PrimaryKey,
    )

    result, err := m.Execute(ctx, query, time.Now(), id)
    if err != nil {
        return err
    }

    if result.RowsAffected == 0 {
        return ErrNotFound
    }

    return nil
}
```

### 4. Transaction Support

```go
type pgxTransaction struct {
    tx     pgx.Tx
    module *PostgresModule
}

func (m *PostgresModule) Transaction(ctx context.Context, fn func(tx Transaction) error) error {
    tx, err := m.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    pgxTx := &pgxTransaction{tx: tx, module: m}

    if err := fn(pgxTx); err != nil {
        if rbErr := tx.Rollback(ctx); rbErr != nil {
            m.logger.Error("rollback failed", "error", rbErr)
        }
        return err
    }

    return tx.Commit(ctx)
}

func (t *pgxTransaction) Query(ctx context.Context, query string, args ...any) ([]Record, error) {
    rows, err := t.tx.Query(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return t.module.scanRows(rows)
}

func (t *pgxTransaction) QueryOne(ctx context.Context, query string, args ...any) (Record, error) {
    records, err := t.Query(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    if len(records) == 0 {
        return nil, ErrNotFound
    }
    return records[0], nil
}

func (t *pgxTransaction) Execute(ctx context.Context, query string, args ...any) (Result, error) {
    tag, err := t.tx.Exec(ctx, query, args...)
    if err != nil {
        return Result{}, err
    }
    return Result{RowsAffected: tag.RowsAffected()}, nil
}

func (t *pgxTransaction) Commit() error {
    return t.tx.Commit(context.Background())
}

func (t *pgxTransaction) Rollback() error {
    return t.tx.Rollback(context.Background())
}
```

### 5. Auto-Migration (internal/modules/database/migrations.go)

```go
package database

import (
    "context"
    "fmt"
    "strings"

    "github.com/codeai/codeai/internal/parser"
)

// RegisterEntity registers an entity for migration
func (m *PostgresModule) RegisterEntity(entity *parser.Entity) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    tableName := toSnakeCase(entity.Name)
    meta := &entityMeta{
        Entity:    entity,
        TableName: tableName,
        PrimaryKey: "id",
    }

    for _, field := range entity.Fields {
        meta.Columns = append(meta.Columns, field.Name)
        if field.Primary {
            meta.PrimaryKey = field.Name
        }
    }

    m.entities[entity.Name] = meta
    return nil
}

// Migrate runs all pending migrations
func (m *PostgresModule) Migrate(ctx context.Context) error {
    // Create migrations table if not exists
    _, err := m.Execute(ctx, `
        CREATE TABLE IF NOT EXISTS _codeai_migrations (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            sql TEXT NOT NULL,
            applied_at TIMESTAMPTZ DEFAULT NOW()
        )
    `)
    if err != nil {
        return fmt.Errorf("create migrations table: %w", err)
    }

    // Generate and run migrations for each entity
    for name, meta := range m.entities {
        if err := m.migrateEntity(ctx, name, meta); err != nil {
            return fmt.Errorf("migrate entity %s: %w", name, err)
        }
    }

    return nil
}

func (m *PostgresModule) migrateEntity(ctx context.Context, name string, meta *entityMeta) error {
    // Check if table exists
    var exists bool
    err := m.pool.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_name = $1
        )
    `, meta.TableName).Scan(&exists)
    if err != nil {
        return err
    }

    if !exists {
        // Create new table
        return m.createTable(ctx, meta)
    }

    // Compare and alter existing table
    return m.alterTable(ctx, meta)
}

func (m *PostgresModule) createTable(ctx context.Context, meta *entityMeta) error {
    var columns []string

    for _, field := range meta.Entity.Fields {
        colDef := m.buildColumnDefinition(field)
        columns = append(columns, colDef)
    }

    // Add primary key constraint
    columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", meta.PrimaryKey))

    // Add foreign key constraints
    for _, field := range meta.Entity.References {
        fk := fmt.Sprintf(
            "FOREIGN KEY (%s) REFERENCES %s(id)",
            field.Name,
            toSnakeCase(field.Reference.EntityName),
        )
        columns = append(columns, fk)
    }

    query := fmt.Sprintf(
        "CREATE TABLE %s (\n  %s\n)",
        meta.TableName,
        strings.Join(columns, ",\n  "),
    )

    m.logger.Info("creating table", "table", meta.TableName)
    _, err := m.Execute(ctx, query)
    if err != nil {
        return err
    }

    // Create indexes
    for _, idx := range meta.Entity.Indexes {
        if err := m.createIndex(ctx, meta, idx); err != nil {
            return err
        }
    }

    // Record migration
    _, err = m.Execute(ctx, `
        INSERT INTO _codeai_migrations (name, sql) VALUES ($1, $2)
    `, fmt.Sprintf("create_table_%s", meta.TableName), query)

    return err
}

func (m *PostgresModule) buildColumnDefinition(field *parser.Field) string {
    var parts []string

    parts = append(parts, field.Name)
    parts = append(parts, field.Type.SQLType("postgres"))

    if field.Required {
        parts = append(parts, "NOT NULL")
    }

    if field.Unique {
        parts = append(parts, "UNIQUE")
    }

    if field.Default != nil {
        defaultVal := m.formatDefaultValue(field.Default, field.Type)
        parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultVal))
    }

    return strings.Join(parts, " ")
}

func (m *PostgresModule) createIndex(ctx context.Context, meta *entityMeta, idx *parser.Index) error {
    indexName := idx.Name
    if indexName == "" {
        indexName = fmt.Sprintf("idx_%s_%s", meta.TableName, strings.Join(idx.Fields, "_"))
    }

    unique := ""
    if idx.Unique {
        unique = "UNIQUE "
    }

    query := fmt.Sprintf(
        "CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)",
        unique,
        indexName,
        meta.TableName,
        strings.Join(idx.Fields, ", "),
    )

    _, err := m.Execute(ctx, query)
    return err
}

func (m *PostgresModule) alterTable(ctx context.Context, meta *entityMeta) error {
    // Get existing columns
    rows, err := m.Query(ctx, `
        SELECT column_name, data_type, is_nullable, column_default
        FROM information_schema.columns
        WHERE table_name = $1
    `, meta.TableName)
    if err != nil {
        return err
    }

    existingCols := make(map[string]bool)
    for _, row := range rows {
        existingCols[row["column_name"].(string)] = true
    }

    // Add missing columns
    for _, field := range meta.Entity.Fields {
        if !existingCols[field.Name] {
            colDef := m.buildColumnDefinition(field)
            query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", meta.TableName, colDef)
            m.logger.Info("adding column", "table", meta.TableName, "column", field.Name)
            if _, err := m.Execute(ctx, query); err != nil {
                return err
            }
        }
    }

    return nil
}

// MigrationStatus returns the status of all migrations
func (m *PostgresModule) MigrationStatus(ctx context.Context) ([]Migration, error) {
    rows, err := m.Query(ctx, `
        SELECT id, name, sql, applied_at
        FROM _codeai_migrations
        ORDER BY id
    `)
    if err != nil {
        return nil, err
    }

    var migrations []Migration
    for _, row := range rows {
        migrations = append(migrations, Migration{
            ID:        int(row["id"].(int32)),
            Name:      row["name"].(string),
            SQL:       row["sql"].(string),
            AppliedAt: row["applied_at"].(time.Time),
        })
    }

    return migrations, nil
}
```

### 6. Query Builder (internal/modules/database/query.go)

```go
package database

import (
    "fmt"
    "strings"
)

func (m *PostgresModule) buildSelectQuery(meta *entityMeta, opts QueryOptions) (string, []any) {
    var args []any
    argIndex := 1

    query := fmt.Sprintf("SELECT * FROM %s", meta.TableName)

    // Build WHERE clause
    var conditions []string

    // Handle soft delete
    if meta.Entity.SoftDelete != nil {
        conditions = append(conditions, fmt.Sprintf("%s IS NULL", meta.Entity.SoftDelete.Name))
    }

    // Handle Where map
    for field, value := range opts.Where {
        conditions = append(conditions, fmt.Sprintf("%s = $%d", field, argIndex))
        args = append(args, value)
        argIndex++
    }

    // Handle WhereExpr
    if opts.WhereExpr != "" {
        conditions = append(conditions, opts.WhereExpr)
        args = append(args, opts.WhereArgs...)
        argIndex += len(opts.WhereArgs)
    }

    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }

    // Build ORDER BY clause
    if len(opts.OrderBy) > 0 {
        var orders []string
        for _, o := range opts.OrderBy {
            dir := "ASC"
            if strings.ToLower(o.Direction) == "desc" {
                dir = "DESC"
            }
            orders = append(orders, fmt.Sprintf("%s %s", o.Field, dir))
        }
        query += " ORDER BY " + strings.Join(orders, ", ")
    }

    // Build LIMIT and OFFSET
    if opts.Limit > 0 {
        query += fmt.Sprintf(" LIMIT %d", opts.Limit)
    }
    if opts.Offset > 0 {
        query += fmt.Sprintf(" OFFSET %d", opts.Offset)
    }

    return query, args
}

func (m *PostgresModule) buildCountQuery(meta *entityMeta, opts QueryOptions) (string, []any) {
    var args []any
    argIndex := 1

    query := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", meta.TableName)

    var conditions []string

    if meta.Entity.SoftDelete != nil {
        conditions = append(conditions, fmt.Sprintf("%s IS NULL", meta.Entity.SoftDelete.Name))
    }

    for field, value := range opts.Where {
        conditions = append(conditions, fmt.Sprintf("%s = $%d", field, argIndex))
        args = append(args, value)
        argIndex++
    }

    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }

    return query, args
}

func (m *PostgresModule) buildInsertParts(meta *entityMeta, data Record) ([]string, []string, []any) {
    var columns, placeholders []string
    var values []any
    idx := 1

    for _, field := range meta.Entity.Fields {
        if value, ok := data[field.Name]; ok {
            columns = append(columns, field.Name)
            placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
            values = append(values, value)
            idx++
        }
    }

    return columns, placeholders, values
}

func (m *PostgresModule) buildUpdateParts(meta *entityMeta, data Record) ([]string, []any) {
    var setParts []string
    var values []any
    idx := 1

    for _, field := range meta.Entity.Fields {
        if field.Primary {
            continue // Skip primary key
        }
        if value, ok := data[field.Name]; ok {
            setParts = append(setParts, fmt.Sprintf("%s = $%d", field.Name, idx))
            values = append(values, value)
            idx++
        }
    }

    return setParts, values
}
```

## Acceptance Criteria
- [ ] Connection pooling works with configurable sizes
- [ ] CRUD operations work for all entity types
- [ ] Transactions support commit and rollback
- [ ] Auto-migration creates tables from entity definitions
- [ ] Auto-migration adds missing columns to existing tables
- [ ] Indexes are created automatically
- [ ] Soft delete is supported
- [ ] All queries are parameterized (SQL injection prevention)
- [ ] Health check endpoint works

## Testing Strategy
- Unit tests with mock database
- Integration tests with testcontainers-go (real PostgreSQL)
- Migration tests (create, alter, rollback)
- Transaction tests (commit, rollback, nested)
- Performance tests for connection pooling

## Files to Create/Modify
- `internal/modules/database/module.go`
- `internal/modules/database/postgres.go`
- `internal/modules/database/migrations.go`
- `internal/modules/database/query.go`
- `internal/modules/database/errors.go`
- `internal/modules/database/postgres_test.go`

## Notes
- Use pgx/v5 for best performance
- Always use parameterized queries
- Support both schema-first and code-first migrations
- Log all queries at debug level
