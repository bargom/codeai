package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/bargom/codeai/internal/database/models"
	"github.com/google/uuid"
)

// ExecutionRepository handles execution persistence.
type ExecutionRepository struct {
	baseRepository
}

// NewExecutionRepository creates a new ExecutionRepository.
func NewExecutionRepository(db Querier) *ExecutionRepository {
	return &ExecutionRepository{
		baseRepository: newBaseRepository(db),
	}
}

// WithTx returns a new ExecutionRepository using the given transaction.
func (r *ExecutionRepository) WithTx(tx *sql.Tx) *ExecutionRepository {
	return NewExecutionRepository(tx)
}

// Create inserts a new execution into the database.
func (r *ExecutionRepository) Create(ctx context.Context, e *models.Execution) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.StartedAt.IsZero() {
		e.StartedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO executions (id, deployment_id, command, output, exit_code, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.DeploymentID, e.Command, e.Output, e.ExitCode, e.StartedAt, e.CompletedAt,
	)
	return err
}

// GetByID retrieves an execution by its ID.
func (r *ExecutionRepository) GetByID(ctx context.Context, id string) (*models.Execution, error) {
	query := `
		SELECT id, deployment_id, command, output, exit_code, started_at, completed_at
		FROM executions
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanExecution(row)
}

// List retrieves all executions with pagination.
func (r *ExecutionRepository) List(ctx context.Context, limit, offset int) ([]*models.Execution, error) {
	query := `
		SELECT id, deployment_id, command, output, exit_code, started_at, completed_at
		FROM executions
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*models.Execution
	for rows.Next() {
		e, err := r.scanExecutionRows(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, e)
	}
	return executions, rows.Err()
}

// ListByDeployment retrieves executions for a specific deployment.
func (r *ExecutionRepository) ListByDeployment(ctx context.Context, deploymentID string, limit, offset int) ([]*models.Execution, error) {
	query := `
		SELECT id, deployment_id, command, output, exit_code, started_at, completed_at
		FROM executions
		WHERE deployment_id = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, deploymentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*models.Execution
	for rows.Next() {
		e, err := r.scanExecutionRows(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, e)
	}
	return executions, rows.Err()
}

// Update updates an existing execution.
func (r *ExecutionRepository) Update(ctx context.Context, e *models.Execution) error {
	query := `
		UPDATE executions
		SET output = ?, exit_code = ?, completed_at = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		e.Output, e.ExitCode, e.CompletedAt, e.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an execution from the database.
func (r *ExecutionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM executions WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// scanExecution scans a single row into an Execution.
func (r *ExecutionRepository) scanExecution(row *sql.Row) (*models.Execution, error) {
	e := &models.Execution{}
	err := row.Scan(
		&e.ID, &e.DeploymentID, &e.Command, &e.Output, &e.ExitCode, &e.StartedAt, &e.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// scanExecutionRows scans a row from a Rows result into an Execution.
func (r *ExecutionRepository) scanExecutionRows(rows *sql.Rows) (*models.Execution, error) {
	e := &models.Execution{}
	err := rows.Scan(
		&e.ID, &e.DeploymentID, &e.Command, &e.Output, &e.ExitCode, &e.StartedAt, &e.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}
