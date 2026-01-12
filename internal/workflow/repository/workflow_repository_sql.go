package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SQLWorkflowRepository implements WorkflowRepository using SQL.
type SQLWorkflowRepository struct {
	db *sql.DB
}

// NewSQLWorkflowRepository creates a new SQL-based workflow repository.
func NewSQLWorkflowRepository(db *sql.DB) *SQLWorkflowRepository {
	return &SQLWorkflowRepository{db: db}
}

// CreateTable creates the workflow_executions table if it doesn't exist.
func (r *SQLWorkflowRepository) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS workflow_executions (
			id TEXT PRIMARY KEY,
			workflow_id TEXT NOT NULL,
			workflow_type TEXT NOT NULL,
			run_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			input TEXT,
			output TEXT,
			error TEXT,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			compensations TEXT,
			metadata TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating workflow_executions table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow_id ON workflow_executions(workflow_id)",
		"CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow_type ON workflow_executions(workflow_type)",
		"CREATE INDEX IF NOT EXISTS idx_workflow_executions_started_at ON workflow_executions(started_at)",
	}

	for _, idx := range indexes {
		if _, err := r.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

// SaveExecution saves a new workflow execution record.
func (r *SQLWorkflowRepository) SaveExecution(ctx context.Context, exec *WorkflowExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	now := time.Now()
	exec.CreatedAt = now
	exec.UpdatedAt = now

	compensationsJSON, err := json.Marshal(exec.Compensations)
	if err != nil {
		return fmt.Errorf("marshaling compensations: %w", err)
	}

	metadataJSON, err := json.Marshal(exec.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	query := `
		INSERT INTO workflow_executions (
			id, workflow_id, workflow_type, run_id, status, input, output, error,
			started_at, completed_at, compensations, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		exec.ID,
		exec.WorkflowID,
		exec.WorkflowType,
		exec.RunID,
		string(exec.Status),
		string(exec.Input),
		string(exec.Output),
		exec.Error,
		exec.StartedAt,
		exec.CompletedAt,
		string(compensationsJSON),
		string(metadataJSON),
		exec.CreatedAt,
		exec.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("inserting workflow execution: %w", err)
	}

	return nil
}

// GetExecution retrieves a workflow execution by its ID.
func (r *SQLWorkflowRepository) GetExecution(ctx context.Context, id string) (*WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, workflow_type, run_id, status, input, output, error,
			   started_at, completed_at, compensations, metadata, created_at, updated_at
		FROM workflow_executions
		WHERE id = ?
	`

	return r.scanExecution(r.db.QueryRowContext(ctx, query, id))
}

// GetExecutionByWorkflowID retrieves a workflow execution by workflow ID.
func (r *SQLWorkflowRepository) GetExecutionByWorkflowID(ctx context.Context, workflowID string) (*WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, workflow_type, run_id, status, input, output, error,
			   started_at, completed_at, compensations, metadata, created_at, updated_at
		FROM workflow_executions
		WHERE workflow_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanExecution(r.db.QueryRowContext(ctx, query, workflowID))
}

// ListExecutions lists workflow executions with optional filtering.
func (r *SQLWorkflowRepository) ListExecutions(ctx context.Context, filter Filter) ([]WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, workflow_type, run_id, status, input, output, error,
			   started_at, completed_at, compensations, metadata, created_at, updated_at
		FROM workflow_executions
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.WorkflowType != "" {
		query += " AND workflow_type = ?"
		args = append(args, filter.WorkflowType)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}

	if filter.StartedAfter != nil {
		query += " AND started_at >= ?"
		args = append(args, *filter.StartedAfter)
	}

	if filter.StartedBefore != nil {
		query += " AND started_at <= ?"
		args = append(args, *filter.StartedBefore)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying workflow executions: %w", err)
	}
	defer rows.Close()

	var executions []WorkflowExecution
	for rows.Next() {
		exec, err := r.scanExecutionFromRows(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, *exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating workflow executions: %w", err)
	}

	return executions, nil
}

// UpdateStatus updates the status of a workflow execution.
func (r *SQLWorkflowRepository) UpdateStatus(ctx context.Context, id string, status Status, errorMsg string) error {
	query := `
		UPDATE workflow_executions
		SET status = ?, error = ?, updated_at = ?
		WHERE id = ?
	`

	var completedAt *time.Time
	if status == StatusCompleted || status == StatusFailed || status == StatusCanceled {
		now := time.Now()
		completedAt = &now

		query = `
			UPDATE workflow_executions
			SET status = ?, error = ?, completed_at = ?, updated_at = ?
			WHERE id = ?
		`
		_, err := r.db.ExecContext(ctx, query, string(status), errorMsg, completedAt, time.Now(), id)
		if err != nil {
			return fmt.Errorf("updating workflow execution status: %w", err)
		}
	} else {
		_, err := r.db.ExecContext(ctx, query, string(status), errorMsg, time.Now(), id)
		if err != nil {
			return fmt.Errorf("updating workflow execution status: %w", err)
		}
	}

	return nil
}

// UpdateOutput updates the output of a workflow execution.
func (r *SQLWorkflowRepository) UpdateOutput(ctx context.Context, id string, output json.RawMessage) error {
	query := `
		UPDATE workflow_executions
		SET output = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, string(output), time.Now(), id)
	if err != nil {
		return fmt.Errorf("updating workflow execution output: %w", err)
	}

	return nil
}

// UpdateCompensations updates the compensation records of a workflow execution.
func (r *SQLWorkflowRepository) UpdateCompensations(ctx context.Context, id string, compensations []CompensationRecord) error {
	compensationsJSON, err := json.Marshal(compensations)
	if err != nil {
		return fmt.Errorf("marshaling compensations: %w", err)
	}

	query := `
		UPDATE workflow_executions
		SET compensations = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query, string(compensationsJSON), time.Now(), id)
	if err != nil {
		return fmt.Errorf("updating workflow execution compensations: %w", err)
	}

	return nil
}

// DeleteExecution deletes a workflow execution by its ID.
func (r *SQLWorkflowRepository) DeleteExecution(ctx context.Context, id string) error {
	query := `DELETE FROM workflow_executions WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting workflow execution: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// CountByStatus counts workflow executions by status.
func (r *SQLWorkflowRepository) CountByStatus(ctx context.Context, status Status) (int64, error) {
	query := `SELECT COUNT(*) FROM workflow_executions WHERE status = ?`

	var count int64
	err := r.db.QueryRowContext(ctx, query, string(status)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting workflow executions: %w", err)
	}

	return count, nil
}

func (r *SQLWorkflowRepository) scanExecution(row *sql.Row) (*WorkflowExecution, error) {
	var exec WorkflowExecution
	var runID, input, output, errorMsg, compensationsJSON, metadataJSON sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	err := row.Scan(
		&exec.ID,
		&exec.WorkflowID,
		&exec.WorkflowType,
		&runID,
		&exec.Status,
		&input,
		&output,
		&errorMsg,
		&startedAt,
		&completedAt,
		&compensationsJSON,
		&metadataJSON,
		&exec.CreatedAt,
		&exec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning workflow execution: %w", err)
	}

	if runID.Valid {
		exec.RunID = runID.String
	}
	if input.Valid {
		exec.Input = json.RawMessage(input.String)
	}
	if output.Valid {
		exec.Output = json.RawMessage(output.String)
	}
	if errorMsg.Valid {
		exec.Error = errorMsg.String
	}
	if startedAt.Valid {
		exec.StartedAt = startedAt.Time
	}
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if compensationsJSON.Valid && compensationsJSON.String != "" {
		if err := json.Unmarshal([]byte(compensationsJSON.String), &exec.Compensations); err != nil {
			return nil, fmt.Errorf("unmarshaling compensations: %w", err)
		}
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &exec.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	return &exec, nil
}

func (r *SQLWorkflowRepository) scanExecutionFromRows(rows *sql.Rows) (*WorkflowExecution, error) {
	var exec WorkflowExecution
	var runID, input, output, errorMsg, compensationsJSON, metadataJSON sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	err := rows.Scan(
		&exec.ID,
		&exec.WorkflowID,
		&exec.WorkflowType,
		&runID,
		&exec.Status,
		&input,
		&output,
		&errorMsg,
		&startedAt,
		&completedAt,
		&compensationsJSON,
		&metadataJSON,
		&exec.CreatedAt,
		&exec.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("scanning workflow execution row: %w", err)
	}

	if runID.Valid {
		exec.RunID = runID.String
	}
	if input.Valid {
		exec.Input = json.RawMessage(input.String)
	}
	if output.Valid {
		exec.Output = json.RawMessage(output.String)
	}
	if errorMsg.Valid {
		exec.Error = errorMsg.String
	}
	if startedAt.Valid {
		exec.StartedAt = startedAt.Time
	}
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if compensationsJSON.Valid && compensationsJSON.String != "" {
		if err := json.Unmarshal([]byte(compensationsJSON.String), &exec.Compensations); err != nil {
			return nil, fmt.Errorf("unmarshaling compensations: %w", err)
		}
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &exec.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	return &exec, nil
}
