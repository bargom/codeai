package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/bargom/codeai/internal/database/models"
	"github.com/google/uuid"
)

// ConfigRepository handles config persistence.
type ConfigRepository struct {
	baseRepository
}

// NewConfigRepository creates a new ConfigRepository.
func NewConfigRepository(db Querier) *ConfigRepository {
	return &ConfigRepository{
		baseRepository: newBaseRepository(db),
	}
}

// WithTx returns a new ConfigRepository using the given transaction.
func (r *ConfigRepository) WithTx(tx *sql.Tx) *ConfigRepository {
	return NewConfigRepository(tx)
}

// Create inserts a new config into the database.
func (r *ConfigRepository) Create(ctx context.Context, c *models.Config) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}

	// Convert JSON fields to string for storage
	var astJSON, validationErrors *string
	if len(c.ASTJSON) > 0 {
		s := string(c.ASTJSON)
		astJSON = &s
	}
	if len(c.ValidationErrors) > 0 {
		s := string(c.ValidationErrors)
		validationErrors = &s
	}

	query := `
		INSERT INTO configs (id, name, content, ast_json, validation_errors, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Name, c.Content, astJSON, validationErrors, c.CreatedAt,
	)
	return err
}

// GetByID retrieves a config by its ID.
func (r *ConfigRepository) GetByID(ctx context.Context, id string) (*models.Config, error) {
	query := `
		SELECT id, name, content, ast_json, validation_errors, created_at
		FROM configs
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanConfig(row)
}

// GetByName retrieves a config by its name.
func (r *ConfigRepository) GetByName(ctx context.Context, name string) (*models.Config, error) {
	query := `
		SELECT id, name, content, ast_json, validation_errors, created_at
		FROM configs
		WHERE name = ?
	`
	row := r.db.QueryRowContext(ctx, query, name)
	return r.scanConfig(row)
}

// List retrieves all configs with pagination.
func (r *ConfigRepository) List(ctx context.Context, limit, offset int) ([]*models.Config, error) {
	query := `
		SELECT id, name, content, ast_json, validation_errors, created_at
		FROM configs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.Config
	for rows.Next() {
		c, err := r.scanConfigRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// Update updates an existing config.
func (r *ConfigRepository) Update(ctx context.Context, c *models.Config) error {
	// Convert JSON fields to string for storage
	var astJSON, validationErrors *string
	if len(c.ASTJSON) > 0 {
		s := string(c.ASTJSON)
		astJSON = &s
	}
	if len(c.ValidationErrors) > 0 {
		s := string(c.ValidationErrors)
		validationErrors = &s
	}

	query := `
		UPDATE configs
		SET name = ?, content = ?, ast_json = ?, validation_errors = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		c.Name, c.Content, astJSON, validationErrors, c.ID,
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

// Delete removes a config from the database.
func (r *ConfigRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM configs WHERE id = ?`
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

// scanConfig scans a single row into a Config.
func (r *ConfigRepository) scanConfig(row *sql.Row) (*models.Config, error) {
	c := &models.Config{}
	var astJSON, validationErrors sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &c.Content, &astJSON, &validationErrors, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if astJSON.Valid {
		c.ASTJSON = json.RawMessage(astJSON.String)
	}
	if validationErrors.Valid {
		c.ValidationErrors = json.RawMessage(validationErrors.String)
	}

	return c, nil
}

// scanConfigRows scans a row from a Rows result into a Config.
func (r *ConfigRepository) scanConfigRows(rows *sql.Rows) (*models.Config, error) {
	c := &models.Config{}
	var astJSON, validationErrors sql.NullString

	err := rows.Scan(
		&c.ID, &c.Name, &c.Content, &astJSON, &validationErrors, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if astJSON.Valid {
		c.ASTJSON = json.RawMessage(astJSON.String)
	}
	if validationErrors.Valid {
		c.ValidationErrors = json.RawMessage(validationErrors.String)
	}

	return c, nil
}
