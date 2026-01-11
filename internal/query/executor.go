package query

import (
	"context"
	"database/sql"
)

// DB is the interface for database operations.
type DB interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Executor executes compiled queries against a database.
type Executor struct {
	db       DB
	compiler *SQLCompiler
}

// NewExecutor creates a new query executor.
func NewExecutor(db DB, entities map[string]*EntityMeta) *Executor {
	return &Executor{
		db:       db,
		compiler: NewSQLCompiler(entities),
	}
}

// Execute executes a query and returns the results as maps.
func (e *Executor) Execute(ctx context.Context, q *Query) ([]map[string]interface{}, error) {
	compiled, err := e.compiler.Compile(q)
	if err != nil {
		return nil, err
	}

	rows, err := e.db.QueryContext(ctx, compiled.SQL, compiled.Params...)
	if err != nil {
		return nil, NewExecutionError(err.Error())
	}
	defer rows.Close()

	return e.scanRows(rows)
}

// ExecuteString parses and executes a query string.
func (e *Executor) ExecuteString(ctx context.Context, queryStr string) ([]map[string]interface{}, error) {
	q, err := Parse(queryStr)
	if err != nil {
		return nil, err
	}
	return e.Execute(ctx, q)
}

// ExecuteCount executes a COUNT query and returns the count.
func (e *Executor) ExecuteCount(ctx context.Context, q *Query) (int64, error) {
	// Ensure it's a count query
	if q.Type != QueryCount {
		q = &Query{
			Type:   QueryCount,
			Entity: q.Entity,
			Where:  q.Where,
		}
	}

	compiled, err := e.compiler.Compile(q)
	if err != nil {
		return 0, err
	}

	var count int64
	err = e.db.QueryRowContext(ctx, compiled.SQL, compiled.Params...).Scan(&count)
	if err != nil {
		return 0, NewExecutionError(err.Error())
	}

	return count, nil
}

// ExecuteAggregate executes an aggregate query (SUM, AVG, MIN, MAX) and returns the result.
func (e *Executor) ExecuteAggregate(ctx context.Context, q *Query) (interface{}, error) {
	compiled, err := e.compiler.Compile(q)
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = e.db.QueryRowContext(ctx, compiled.SQL, compiled.Params...).Scan(&result)
	if err != nil {
		return nil, NewExecutionError(err.Error())
	}

	return result, nil
}

// ExecuteUpdate executes an UPDATE query and returns the number of affected rows.
func (e *Executor) ExecuteUpdate(ctx context.Context, q *Query) (int64, error) {
	if q.Type != QueryUpdate {
		return 0, NewCompilerError("expected UPDATE query")
	}

	compiled, err := e.compiler.Compile(q)
	if err != nil {
		return 0, err
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL, compiled.Params...)
	if err != nil {
		return 0, NewExecutionError(err.Error())
	}

	return result.RowsAffected()
}

// ExecuteDelete executes a DELETE query and returns the number of affected rows.
func (e *Executor) ExecuteDelete(ctx context.Context, q *Query) (int64, error) {
	if q.Type != QueryDelete {
		return 0, NewCompilerError("expected DELETE query")
	}

	compiled, err := e.compiler.Compile(q)
	if err != nil {
		return 0, err
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL, compiled.Params...)
	if err != nil {
		return 0, NewExecutionError(err.Error())
	}

	return result.RowsAffected()
}

// ExecuteOne executes a query and returns a single result.
func (e *Executor) ExecuteOne(ctx context.Context, q *Query) (map[string]interface{}, error) {
	// Add LIMIT 1 if not already set
	if q.Limit == nil {
		limit := 1
		q.Limit = &limit
	}

	results, err := e.Execute(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

// scanRows scans SQL rows into a slice of maps.
func (e *Executor) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, NewExecutionError(err.Error())
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, NewExecutionError(err.Error())
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, NewExecutionError(err.Error())
	}

	return results, nil
}

// QueryBuilder provides a fluent interface for building queries.
type QueryBuilder struct {
	query *Query
}

// Select creates a new SELECT query builder.
func Select(entity string) *QueryBuilder {
	return &QueryBuilder{
		query: &Query{
			Type:   QuerySelect,
			Entity: entity,
		},
	}
}

// Count creates a new COUNT query builder.
func Count(entity string) *QueryBuilder {
	return &QueryBuilder{
		query: &Query{
			Type:   QueryCount,
			Entity: entity,
		},
	}
}

// Update creates a new UPDATE query builder.
func Update(entity string) *QueryBuilder {
	return &QueryBuilder{
		query: &Query{
			Type:   QueryUpdate,
			Entity: entity,
		},
	}
}

// Delete creates a new DELETE query builder.
func Delete(entity string) *QueryBuilder {
	return &QueryBuilder{
		query: &Query{
			Type:   QueryDelete,
			Entity: entity,
		},
	}
}

// Fields specifies the fields to select.
func (qb *QueryBuilder) Fields(fields ...string) *QueryBuilder {
	qb.query.Fields = fields
	return qb
}

// Where adds a WHERE condition.
func (qb *QueryBuilder) Where(field string, op CompareOp, value interface{}) *QueryBuilder {
	cond := Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	}

	if qb.query.Where == nil {
		qb.query.Where = &WhereClause{
			Operator: LogicalAnd,
		}
	}
	qb.query.Where.Conditions = append(qb.query.Where.Conditions, cond)

	return qb
}

// WhereClause sets the entire WHERE clause.
func (qb *QueryBuilder) WhereClause(where *WhereClause) *QueryBuilder {
	qb.query.Where = where
	return qb
}

// OrderBy adds an ORDER BY clause.
func (qb *QueryBuilder) OrderBy(field string, direction OrderDirection) *QueryBuilder {
	qb.query.OrderBy = append(qb.query.OrderBy, OrderClause{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// Limit sets the LIMIT.
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.query.Limit = &limit
	return qb
}

// Offset sets the OFFSET.
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.query.Offset = &offset
	return qb
}

// Include adds related entities to load.
func (qb *QueryBuilder) Include(relations ...string) *QueryBuilder {
	qb.query.Include = append(qb.query.Include, relations...)
	return qb
}

// Set adds an update set operation.
func (qb *QueryBuilder) Set(field string, value interface{}) *QueryBuilder {
	qb.query.Updates = append(qb.query.Updates, UpdateSet{
		Field: field,
		Value: value,
		Op:    UpdateSetValue,
	})
	return qb
}

// Increment adds an increment operation.
func (qb *QueryBuilder) Increment(field string, value interface{}) *QueryBuilder {
	qb.query.Updates = append(qb.query.Updates, UpdateSet{
		Field: field,
		Value: value,
		Op:    UpdateIncrement,
	})
	return qb
}

// Decrement adds a decrement operation.
func (qb *QueryBuilder) Decrement(field string, value interface{}) *QueryBuilder {
	qb.query.Updates = append(qb.query.Updates, UpdateSet{
		Field: field,
		Value: value,
		Op:    UpdateDecrement,
	})
	return qb
}

// Build returns the constructed query.
func (qb *QueryBuilder) Build() *Query {
	return qb.query
}

// Execute executes the query using the provided executor.
func (qb *QueryBuilder) Execute(ctx context.Context, exec *Executor) ([]map[string]interface{}, error) {
	return exec.Execute(ctx, qb.query)
}

// ExecuteOne executes the query and returns a single result.
func (qb *QueryBuilder) ExecuteOne(ctx context.Context, exec *Executor) (map[string]interface{}, error) {
	return exec.ExecuteOne(ctx, qb.query)
}
