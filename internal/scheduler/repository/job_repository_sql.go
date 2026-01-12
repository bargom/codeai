package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SQLJobRepository implements JobRepository using SQL database.
type SQLJobRepository struct {
	db *sql.DB
}

// NewSQLJobRepository creates a new SQL-based job repository.
func NewSQLJobRepository(db *sql.DB) *SQLJobRepository {
	return &SQLJobRepository{db: db}
}

// CreateJob creates a new job record.
func (r *SQLJobRepository) CreateJob(ctx context.Context, job *Job) error {
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	metadataBytes, err := json.Marshal(job.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO scheduler_jobs (
			id, task_type, payload, status, queue, scheduled_at,
			retry_count, max_retries, cron_expression, cron_entry_id,
			timeout, created_at, updated_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err = r.db.ExecContext(ctx, query,
		job.ID, job.TaskType, payloadBytes, job.Status, job.Queue, job.ScheduledAt,
		job.RetryCount, job.MaxRetries, job.CronExpression, job.CronEntryID,
		job.Timeout, job.CreatedAt, job.UpdatedAt, metadataBytes,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID.
func (r *SQLJobRepository) GetJob(ctx context.Context, jobID string) (*Job, error) {
	query := `
		SELECT id, task_type, payload, status, queue, scheduled_at,
			started_at, completed_at, failed_at, retry_count, max_retries,
			error, result, cron_expression, cron_entry_id, timeout,
			created_at, updated_at, metadata
		FROM scheduler_jobs
		WHERE id = $1`

	job := &Job{}
	var payloadBytes, resultBytes, metadataBytes []byte
	var scheduledAt, startedAt, completedAt, failedAt sql.NullTime
	var cronExpr, cronEntryID, errorStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.TaskType, &payloadBytes, &job.Status, &job.Queue, &scheduledAt,
		&startedAt, &completedAt, &failedAt, &job.RetryCount, &job.MaxRetries,
		&errorStr, &resultBytes, &cronExpr, &cronEntryID, &job.Timeout,
		&job.CreatedAt, &job.UpdatedAt, &metadataBytes,
	)
	if err == sql.ErrNoRows {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	job.Payload = payloadBytes
	job.Result = resultBytes
	if scheduledAt.Valid {
		job.ScheduledAt = &scheduledAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if failedAt.Valid {
		job.FailedAt = &failedAt.Time
	}
	if cronExpr.Valid {
		job.CronExpression = cronExpr.String
	}
	if cronEntryID.Valid {
		job.CronEntryID = cronEntryID.String
	}
	if errorStr.Valid {
		job.Error = errorStr.String
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &job.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return job, nil
}

// UpdateJob updates an existing job.
func (r *SQLJobRepository) UpdateJob(ctx context.Context, job *Job) error {
	job.UpdatedAt = time.Now()

	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	metadataBytes, err := json.Marshal(job.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE scheduler_jobs SET
			task_type = $2, payload = $3, status = $4, queue = $5,
			scheduled_at = $6, started_at = $7, completed_at = $8, failed_at = $9,
			retry_count = $10, max_retries = $11, error = $12, result = $13,
			cron_expression = $14, cron_entry_id = $15, timeout = $16,
			updated_at = $17, metadata = $18
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		job.ID, job.TaskType, payloadBytes, job.Status, job.Queue,
		job.ScheduledAt, job.StartedAt, job.CompletedAt, job.FailedAt,
		job.RetryCount, job.MaxRetries, job.Error, job.Result,
		job.CronExpression, job.CronEntryID, job.Timeout,
		job.UpdatedAt, metadataBytes,
	)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// UpdateJobStatus updates only the status and related timestamps.
func (r *SQLJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, jobErr error) error {
	now := time.Now()
	var query string
	var args []any

	switch status {
	case JobStatusRunning:
		query = `UPDATE scheduler_jobs SET status = $2, started_at = $3, updated_at = $3 WHERE id = $1`
		args = []any{jobID, status, now}
	case JobStatusCompleted:
		query = `UPDATE scheduler_jobs SET status = $2, completed_at = $3, updated_at = $3 WHERE id = $1`
		args = []any{jobID, status, now}
	case JobStatusFailed:
		errStr := ""
		if jobErr != nil {
			errStr = jobErr.Error()
		}
		query = `UPDATE scheduler_jobs SET status = $2, failed_at = $3, error = $4, updated_at = $3 WHERE id = $1`
		args = []any{jobID, status, now, errStr}
	default:
		query = `UPDATE scheduler_jobs SET status = $2, updated_at = $3 WHERE id = $1`
		args = []any{jobID, status, now}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// DeleteJob removes a job.
func (r *SQLJobRepository) DeleteJob(ctx context.Context, jobID string) error {
	query := `DELETE FROM scheduler_jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// ListJobs lists jobs based on filter criteria.
func (r *SQLJobRepository) ListJobs(ctx context.Context, filter JobFilter) ([]Job, error) {
	query, args := r.buildListQuery(filter, false)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return jobs, nil
}

// CountJobs counts jobs based on filter criteria.
func (r *SQLJobRepository) CountJobs(ctx context.Context, filter JobFilter) (int64, error) {
	query, args := r.buildListQuery(filter, true)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count jobs: %w", err)
	}

	return count, nil
}

// GetJobsByStatus retrieves jobs by status with a limit.
func (r *SQLJobRepository) GetJobsByStatus(ctx context.Context, status JobStatus, limit int) ([]Job, error) {
	return r.ListJobs(ctx, JobFilter{
		Status: []JobStatus{status},
		Limit:  limit,
	})
}

// GetPendingJobs retrieves all pending jobs ready to be executed.
func (r *SQLJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]Job, error) {
	now := time.Now()
	return r.ListJobs(ctx, JobFilter{
		Status:          []JobStatus{JobStatusPending, JobStatusScheduled},
		ScheduledBefore: &now,
		Limit:           limit,
		OrderBy:         "scheduled_at",
		OrderDirection:  "ASC",
	})
}

// GetRecurringJobs retrieves all jobs with cron expressions.
func (r *SQLJobRepository) GetRecurringJobs(ctx context.Context) ([]Job, error) {
	withCron := true
	return r.ListJobs(ctx, JobFilter{
		WithCron: &withCron,
		Limit:    1000,
	})
}

// SetJobResult stores the result of a job execution.
func (r *SQLJobRepository) SetJobResult(ctx context.Context, jobID string, result json.RawMessage) error {
	query := `UPDATE scheduler_jobs SET result = $2, updated_at = $3 WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, jobID, result, time.Now())
	if err != nil {
		return fmt.Errorf("set job result: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// IncrementRetryCount increments the retry count for a job.
func (r *SQLJobRepository) IncrementRetryCount(ctx context.Context, jobID string) error {
	query := `UPDATE scheduler_jobs SET retry_count = retry_count + 1, updated_at = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, jobID, time.Now())
	if err != nil {
		return fmt.Errorf("increment retry count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// buildListQuery builds the SQL query for listing jobs.
func (r *SQLJobRepository) buildListQuery(filter JobFilter, countOnly bool) (string, []any) {
	var conditions []string
	var args []any
	argIndex := 1

	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, s := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, s)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(filter.TaskTypes) > 0 {
		placeholders := make([]string, len(filter.TaskTypes))
		for i, t := range filter.TaskTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, t)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("task_type IN (%s)", strings.Join(placeholders, ", ")))
	}

	if filter.Queue != "" {
		conditions = append(conditions, fmt.Sprintf("queue = $%d", argIndex))
		args = append(args, filter.Queue)
		argIndex++
	}

	if filter.ScheduledBefore != nil {
		conditions = append(conditions, fmt.Sprintf("(scheduled_at IS NULL OR scheduled_at <= $%d)", argIndex))
		args = append(args, *filter.ScheduledBefore)
		argIndex++
	}

	if filter.ScheduledAfter != nil {
		conditions = append(conditions, fmt.Sprintf("scheduled_at >= $%d", argIndex))
		args = append(args, *filter.ScheduledAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.WithCron != nil {
		if *filter.WithCron {
			conditions = append(conditions, "cron_expression IS NOT NULL AND cron_expression != ''")
		} else {
			conditions = append(conditions, "(cron_expression IS NULL OR cron_expression = '')")
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	if countOnly {
		return fmt.Sprintf("SELECT COUNT(*) FROM scheduler_jobs %s", whereClause), args
	}

	// Validate order by column
	orderBy := "created_at"
	if filter.OrderBy != "" {
		switch filter.OrderBy {
		case "created_at", "updated_at", "scheduled_at", "status", "task_type":
			orderBy = filter.OrderBy
		}
	}

	orderDir := "DESC"
	if filter.OrderDirection == "ASC" {
		orderDir = "ASC"
	}

	limit := 100
	if filter.Limit > 0 && filter.Limit <= 1000 {
		limit = filter.Limit
	}

	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT id, task_type, payload, status, queue, scheduled_at,
			started_at, completed_at, failed_at, retry_count, max_retries,
			error, result, cron_expression, cron_entry_id, timeout,
			created_at, updated_at, metadata
		FROM scheduler_jobs
		%s
		ORDER BY %s %s
		LIMIT %d OFFSET %d`,
		whereClause, orderBy, orderDir, limit, offset)

	return query, args
}

// scanJob scans a row into a Job struct.
func (r *SQLJobRepository) scanJob(rows *sql.Rows) (*Job, error) {
	job := &Job{}
	var payloadBytes, resultBytes, metadataBytes []byte
	var scheduledAt, startedAt, completedAt, failedAt sql.NullTime
	var cronExpr, cronEntryID, errorStr sql.NullString

	err := rows.Scan(
		&job.ID, &job.TaskType, &payloadBytes, &job.Status, &job.Queue, &scheduledAt,
		&startedAt, &completedAt, &failedAt, &job.RetryCount, &job.MaxRetries,
		&errorStr, &resultBytes, &cronExpr, &cronEntryID, &job.Timeout,
		&job.CreatedAt, &job.UpdatedAt, &metadataBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}

	job.Payload = payloadBytes
	job.Result = resultBytes
	if scheduledAt.Valid {
		job.ScheduledAt = &scheduledAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if failedAt.Valid {
		job.FailedAt = &failedAt.Time
	}
	if cronExpr.Valid {
		job.CronExpression = cronExpr.String
	}
	if cronEntryID.Valid {
		job.CronEntryID = cronEntryID.String
	}
	if errorStr.Valid {
		job.Error = errorStr.String
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &job.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return job, nil
}

// CreateTable creates the scheduler_jobs table if it doesn't exist.
func (r *SQLJobRepository) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS scheduler_jobs (
			id TEXT PRIMARY KEY,
			task_type TEXT NOT NULL,
			payload BYTEA,
			status TEXT NOT NULL DEFAULT 'pending',
			queue TEXT NOT NULL DEFAULT 'default',
			scheduled_at TIMESTAMP WITH TIME ZONE,
			started_at TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			failed_at TIMESTAMP WITH TIME ZONE,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			error TEXT,
			result BYTEA,
			cron_expression TEXT,
			cron_entry_id TEXT,
			timeout BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			metadata JSONB
		);

		CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_status ON scheduler_jobs(status);
		CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_task_type ON scheduler_jobs(task_type);
		CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_queue ON scheduler_jobs(queue);
		CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_scheduled_at ON scheduler_jobs(scheduled_at);
		CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_created_at ON scheduler_jobs(created_at);
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}
