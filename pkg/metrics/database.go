package metrics

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// DBMetrics provides methods to record database-related metrics.
type DBMetrics struct {
	registry *Registry
}

// DB returns the database metrics interface for the registry.
func (r *Registry) DB() *DBMetrics {
	return &DBMetrics{registry: r}
}

// Operation represents a database operation type.
type Operation string

const (
	OperationSelect Operation = "SELECT"
	OperationInsert Operation = "INSERT"
	OperationUpdate Operation = "UPDATE"
	OperationDelete Operation = "DELETE"
	OperationOther  Operation = "OTHER"
)

// QueryStatus represents the result status of a database query.
type QueryStatus string

const (
	QueryStatusSuccess QueryStatus = "success"
	QueryStatusError   QueryStatus = "error"
)

// RecordQuery records metrics for a database query.
func (d *DBMetrics) RecordQuery(operation Operation, table string, duration time.Duration, err error) {
	status := QueryStatusSuccess
	if err != nil {
		status = QueryStatusError
	}

	d.registry.dbQueriesTotal.WithLabelValues(
		string(operation),
		table,
		string(status),
	).Inc()

	d.registry.dbQueryDuration.WithLabelValues(
		string(operation),
		table,
	).Observe(duration.Seconds())
}

// RecordQueryError records a query error with error type classification.
func (d *DBMetrics) RecordQueryError(operation Operation, table string, errorType string) {
	d.registry.dbQueryErrors.WithLabelValues(
		string(operation),
		table,
		errorType,
	).Inc()
}

// UpdateConnectionStats updates database connection pool metrics.
func (d *DBMetrics) UpdateConnectionStats(active, idle, max int) {
	d.registry.dbConnectionsActive.Set(float64(active))
	d.registry.dbConnectionsIdle.Set(float64(idle))
	d.registry.dbConnectionsMax.Set(float64(max))
}

// UpdateFromDBStats updates metrics from sql.DBStats.
func (d *DBMetrics) UpdateFromDBStats(stats sql.DBStats) {
	d.registry.dbConnectionsActive.Set(float64(stats.InUse))
	d.registry.dbConnectionsIdle.Set(float64(stats.Idle))
	d.registry.dbConnectionsMax.Set(float64(stats.MaxOpenConnections))
}

// DetectOperation parses a SQL query to determine its operation type.
func DetectOperation(query string) Operation {
	query = strings.TrimSpace(strings.ToUpper(query))

	switch {
	case strings.HasPrefix(query, "SELECT"):
		return OperationSelect
	case strings.HasPrefix(query, "INSERT"):
		return OperationInsert
	case strings.HasPrefix(query, "UPDATE"):
		return OperationUpdate
	case strings.HasPrefix(query, "DELETE"):
		return OperationDelete
	default:
		return OperationOther
	}
}

// QueryTimer provides a convenient way to time database queries.
type QueryTimer struct {
	dbMetrics *DBMetrics
	operation Operation
	table     string
	start     time.Time
}

// NewQueryTimer creates a new query timer.
func (d *DBMetrics) NewQueryTimer(operation Operation, table string) *QueryTimer {
	return &QueryTimer{
		dbMetrics: d,
		operation: operation,
		table:     table,
		start:     time.Now(),
	}
}

// Done records the query duration and any error.
func (qt *QueryTimer) Done(err error) {
	duration := time.Since(qt.start)
	qt.dbMetrics.RecordQuery(qt.operation, qt.table, duration, err)

	if err != nil {
		errorType := classifyDBError(err)
		qt.dbMetrics.RecordQueryError(qt.operation, qt.table, errorType)
	}
}

// classifyDBError attempts to classify a database error for metrics.
func classifyDBError(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "connection"):
		return "connection"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "constraint"):
		return "constraint_violation"
	case strings.Contains(errStr, "duplicate"):
		return "duplicate_key"
	case strings.Contains(errStr, "deadlock"):
		return "deadlock"
	case strings.Contains(errStr, "not found") || strings.Contains(errStr, "no rows"):
		return "not_found"
	default:
		return "unknown"
	}
}

// Context key for query timing.
type dbContextKey struct{}

// DBQueryContext stores query timing information in context.
type DBQueryContext struct {
	StartTime time.Time
	Operation Operation
	Table     string
}

// StartQueryContext adds query timing information to the context.
func StartQueryContext(ctx context.Context, operation Operation, table string) context.Context {
	return context.WithValue(ctx, dbContextKey{}, &DBQueryContext{
		StartTime: time.Now(),
		Operation: operation,
		Table:     table,
	})
}

// EndQueryContext records query metrics from context.
func (d *DBMetrics) EndQueryContext(ctx context.Context, err error) {
	qc, ok := ctx.Value(dbContextKey{}).(*DBQueryContext)
	if !ok {
		return
	}

	duration := time.Since(qc.StartTime)
	d.RecordQuery(qc.Operation, qc.Table, duration, err)

	if err != nil {
		errorType := classifyDBError(err)
		d.RecordQueryError(qc.Operation, qc.Table, errorType)
	}
}

// StartConnectionStatsCollector starts a goroutine that periodically updates
// connection stats from a sql.DB instance.
func (d *DBMetrics) StartConnectionStatsCollector(db *sql.DB, interval time.Duration) func() {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				d.UpdateFromDBStats(db.Stats())
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
	}
}
