package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/bargom/codeai/internal/database/models"
	"github.com/google/uuid"
)

// DeploymentRepository handles deployment persistence.
type DeploymentRepository struct {
	baseRepository
}

// NewDeploymentRepository creates a new DeploymentRepository.
func NewDeploymentRepository(db Querier) *DeploymentRepository {
	return &DeploymentRepository{
		baseRepository: newBaseRepository(db),
	}
}

// WithTx returns a new DeploymentRepository using the given transaction.
func (r *DeploymentRepository) WithTx(tx *sql.Tx) *DeploymentRepository {
	return NewDeploymentRepository(tx)
}

// Create inserts a new deployment into the database.
func (r *DeploymentRepository) Create(ctx context.Context, d *models.Deployment) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO deployments (id, name, config_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.Name, d.ConfigID, d.Status, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

// GetByID retrieves a deployment by its ID.
func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	query := `
		SELECT id, name, config_id, status, created_at, updated_at
		FROM deployments
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanDeployment(row)
}

// GetByName retrieves a deployment by its name.
func (r *DeploymentRepository) GetByName(ctx context.Context, name string) (*models.Deployment, error) {
	query := `
		SELECT id, name, config_id, status, created_at, updated_at
		FROM deployments
		WHERE name = ?
	`
	row := r.db.QueryRowContext(ctx, query, name)
	return r.scanDeployment(row)
}

// List retrieves all deployments with pagination.
func (r *DeploymentRepository) List(ctx context.Context, limit, offset int) ([]*models.Deployment, error) {
	query := `
		SELECT id, name, config_id, status, created_at, updated_at
		FROM deployments
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []*models.Deployment
	for rows.Next() {
		d, err := r.scanDeploymentRows(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

// Update updates an existing deployment.
func (r *DeploymentRepository) Update(ctx context.Context, d *models.Deployment) error {
	d.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE deployments
		SET name = ?, config_id = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		d.Name, d.ConfigID, d.Status, d.UpdatedAt, d.ID,
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

// Delete removes a deployment from the database.
func (r *DeploymentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM deployments WHERE id = ?`
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

// scanDeployment scans a single row into a Deployment.
func (r *DeploymentRepository) scanDeployment(row *sql.Row) (*models.Deployment, error) {
	d := &models.Deployment{}
	err := row.Scan(
		&d.ID, &d.Name, &d.ConfigID, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// scanDeploymentRows scans a row from a Rows result into a Deployment.
func (r *DeploymentRepository) scanDeploymentRows(rows *sql.Rows) (*models.Deployment, error) {
	d := &models.Deployment{}
	err := rows.Scan(
		&d.ID, &d.Name, &d.ConfigID, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}
