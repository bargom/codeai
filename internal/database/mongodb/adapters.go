// Package mongodb provides MongoDB database connectivity and repository operations.
package mongodb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/bargom/codeai/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// =============================================================================
// Deployment Adapter
// =============================================================================

// DeploymentAdapter adapts MongoDB Repository to a deployment repository interface.
type DeploymentAdapter struct {
	repo   *Repository
	logger *slog.Logger
}

// NewDeploymentAdapter creates a new DeploymentAdapter.
func NewDeploymentAdapter(client *Client, logger *slog.Logger) *DeploymentAdapter {
	return &DeploymentAdapter{
		repo:   NewRepository(client, "deployments", logger),
		logger: logger,
	}
}

// Create inserts a new deployment.
func (a *DeploymentAdapter) Create(ctx context.Context, d *models.Deployment) error {
	doc := Document{
		"name":      d.Name,
		"status":    d.Status,
		"createdAt": d.CreatedAt,
		"updatedAt": d.UpdatedAt,
	}

	// Handle optional ConfigID
	if d.ConfigID.Valid {
		doc["configId"] = d.ConfigID.String
	}

	// Use provided ID or generate new one
	if d.ID != "" {
		doc["_id"] = d.ID
	}

	id, err := a.repo.InsertOne(ctx, doc)
	if err != nil {
		return translateAdapterError(err)
	}

	// Set ID if it was generated
	if d.ID == "" {
		d.ID = id
	}

	return nil
}

// GetByID retrieves a deployment by its ID.
func (a *DeploymentAdapter) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	doc, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateAdapterError(err)
	}
	return documentToDeployment(doc)
}

// GetByName retrieves a deployment by its name.
func (a *DeploymentAdapter) GetByName(ctx context.Context, name string) (*models.Deployment, error) {
	doc, err := a.repo.FindOne(ctx, Filter{"name": name})
	if err != nil {
		return nil, translateAdapterError(err)
	}
	return documentToDeployment(doc)
}

// List retrieves all deployments with pagination.
func (a *DeploymentAdapter) List(ctx context.Context, limit, offset int) ([]*models.Deployment, error) {
	docs, err := a.repo.Find(ctx, Filter{}, &FindOptions{
		Sort:  bson.D{{Key: "createdAt", Value: -1}},
		Limit: int64(limit),
		Skip:  int64(offset),
	})
	if err != nil {
		return nil, translateAdapterError(err)
	}

	deployments := make([]*models.Deployment, 0, len(docs))
	for _, doc := range docs {
		d, err := documentToDeployment(doc)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// Update updates an existing deployment.
func (a *DeploymentAdapter) Update(ctx context.Context, d *models.Deployment) error {
	d.UpdatedAt = time.Now().UTC()

	update := bson.M{
		"name":      d.Name,
		"status":    d.Status,
		"updatedAt": d.UpdatedAt,
	}

	if d.ConfigID.Valid {
		update["configId"] = d.ConfigID.String
	} else {
		update["configId"] = nil
	}

	count, err := a.repo.UpdateOne(ctx, Filter{"_id": d.ID}, Update{"$set": update})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a deployment.
func (a *DeploymentAdapter) Delete(ctx context.Context, id string) error {
	count, err := a.repo.DeleteOne(ctx, Filter{"_id": id})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// =============================================================================
// Config Adapter
// =============================================================================

// ConfigAdapter adapts MongoDB Repository to a config repository interface.
type ConfigAdapter struct {
	repo   *Repository
	logger *slog.Logger
}

// NewConfigAdapter creates a new ConfigAdapter.
func NewConfigAdapter(client *Client, logger *slog.Logger) *ConfigAdapter {
	return &ConfigAdapter{
		repo:   NewRepository(client, "configs", logger),
		logger: logger,
	}
}

// Create inserts a new config.
func (a *ConfigAdapter) Create(ctx context.Context, c *models.Config) error {
	doc := Document{
		"name":      c.Name,
		"content":   c.Content,
		"createdAt": c.CreatedAt,
	}

	// Use provided ID or generate new one
	if c.ID != "" {
		doc["_id"] = c.ID
	}

	// Handle JSON fields
	if len(c.ASTJSON) > 0 {
		var astData interface{}
		if err := json.Unmarshal(c.ASTJSON, &astData); err == nil {
			doc["astJson"] = astData
		}
	}
	if len(c.ValidationErrors) > 0 {
		var errData interface{}
		if err := json.Unmarshal(c.ValidationErrors, &errData); err == nil {
			doc["validationErrors"] = errData
		}
	}

	id, err := a.repo.InsertOne(ctx, doc)
	if err != nil {
		return translateAdapterError(err)
	}

	// Set ID if it was generated
	if c.ID == "" {
		c.ID = id
	}

	return nil
}

// GetByID retrieves a config by its ID.
func (a *ConfigAdapter) GetByID(ctx context.Context, id string) (*models.Config, error) {
	doc, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateAdapterError(err)
	}
	return documentToConfig(doc)
}

// GetByName retrieves a config by its name.
func (a *ConfigAdapter) GetByName(ctx context.Context, name string) (*models.Config, error) {
	doc, err := a.repo.FindOne(ctx, Filter{"name": name})
	if err != nil {
		return nil, translateAdapterError(err)
	}
	return documentToConfig(doc)
}

// List retrieves all configs with pagination.
func (a *ConfigAdapter) List(ctx context.Context, limit, offset int) ([]*models.Config, error) {
	docs, err := a.repo.Find(ctx, Filter{}, &FindOptions{
		Sort:  bson.D{{Key: "createdAt", Value: -1}},
		Limit: int64(limit),
		Skip:  int64(offset),
	})
	if err != nil {
		return nil, translateAdapterError(err)
	}

	configs := make([]*models.Config, 0, len(docs))
	for _, doc := range docs {
		c, err := documentToConfig(doc)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}

	return configs, nil
}

// Update updates an existing config.
func (a *ConfigAdapter) Update(ctx context.Context, c *models.Config) error {
	update := bson.M{
		"name":    c.Name,
		"content": c.Content,
	}

	// Handle JSON fields
	if len(c.ASTJSON) > 0 {
		var astData interface{}
		if err := json.Unmarshal(c.ASTJSON, &astData); err == nil {
			update["astJson"] = astData
		}
	}
	if len(c.ValidationErrors) > 0 {
		var errData interface{}
		if err := json.Unmarshal(c.ValidationErrors, &errData); err == nil {
			update["validationErrors"] = errData
		}
	}

	count, err := a.repo.UpdateOne(ctx, Filter{"_id": c.ID}, Update{"$set": update})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a config.
func (a *ConfigAdapter) Delete(ctx context.Context, id string) error {
	count, err := a.repo.DeleteOne(ctx, Filter{"_id": id})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// =============================================================================
// Execution Adapter
// =============================================================================

// ExecutionAdapter adapts MongoDB Repository to an execution repository interface.
type ExecutionAdapter struct {
	repo   *Repository
	logger *slog.Logger
}

// NewExecutionAdapter creates a new ExecutionAdapter.
func NewExecutionAdapter(client *Client, logger *slog.Logger) *ExecutionAdapter {
	return &ExecutionAdapter{
		repo:   NewRepository(client, "executions", logger),
		logger: logger,
	}
}

// Create inserts a new execution.
func (a *ExecutionAdapter) Create(ctx context.Context, e *models.Execution) error {
	doc := Document{
		"deploymentId": e.DeploymentID,
		"command":      e.Command,
		"startedAt":    e.StartedAt,
	}

	// Use provided ID or generate new one
	if e.ID != "" {
		doc["_id"] = e.ID
	}

	// Handle optional fields
	if e.Output.Valid {
		doc["output"] = e.Output.String
	}
	if e.ExitCode.Valid {
		doc["exitCode"] = e.ExitCode.Int32
	}
	if e.CompletedAt.Valid {
		doc["completedAt"] = e.CompletedAt.Time
	}

	id, err := a.repo.InsertOne(ctx, doc)
	if err != nil {
		return translateAdapterError(err)
	}

	// Set ID if it was generated
	if e.ID == "" {
		e.ID = id
	}

	return nil
}

// GetByID retrieves an execution by its ID.
func (a *ExecutionAdapter) GetByID(ctx context.Context, id string) (*models.Execution, error) {
	doc, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateAdapterError(err)
	}
	return documentToExecution(doc)
}

// List retrieves all executions with pagination.
func (a *ExecutionAdapter) List(ctx context.Context, limit, offset int) ([]*models.Execution, error) {
	docs, err := a.repo.Find(ctx, Filter{}, &FindOptions{
		Sort:  bson.D{{Key: "startedAt", Value: -1}},
		Limit: int64(limit),
		Skip:  int64(offset),
	})
	if err != nil {
		return nil, translateAdapterError(err)
	}

	executions := make([]*models.Execution, 0, len(docs))
	for _, doc := range docs {
		e, err := documentToExecution(doc)
		if err != nil {
			return nil, err
		}
		executions = append(executions, e)
	}

	return executions, nil
}

// ListByDeployment retrieves executions for a specific deployment.
func (a *ExecutionAdapter) ListByDeployment(ctx context.Context, deploymentID string, limit, offset int) ([]*models.Execution, error) {
	docs, err := a.repo.Find(ctx, Filter{"deploymentId": deploymentID}, &FindOptions{
		Sort:  bson.D{{Key: "startedAt", Value: -1}},
		Limit: int64(limit),
		Skip:  int64(offset),
	})
	if err != nil {
		return nil, translateAdapterError(err)
	}

	executions := make([]*models.Execution, 0, len(docs))
	for _, doc := range docs {
		e, err := documentToExecution(doc)
		if err != nil {
			return nil, err
		}
		executions = append(executions, e)
	}

	return executions, nil
}

// Update updates an existing execution.
func (a *ExecutionAdapter) Update(ctx context.Context, e *models.Execution) error {
	update := bson.M{}

	if e.Output.Valid {
		update["output"] = e.Output.String
	}
	if e.ExitCode.Valid {
		update["exitCode"] = e.ExitCode.Int32
	}
	if e.CompletedAt.Valid {
		update["completedAt"] = e.CompletedAt.Time
	}

	count, err := a.repo.UpdateOne(ctx, Filter{"_id": e.ID}, Update{"$set": update})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes an execution.
func (a *ExecutionAdapter) Delete(ctx context.Context, id string) error {
	count, err := a.repo.DeleteOne(ctx, Filter{"_id": id})
	if err != nil {
		return translateAdapterError(err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// =============================================================================
// Document Conversion Helpers
// =============================================================================

// documentToDeployment converts a MongoDB document to a Deployment model.
func documentToDeployment(doc Document) (*models.Deployment, error) {
	d := &models.Deployment{}

	// Extract ID
	d.ID = extractDocID(doc)

	// Extract string fields
	if name, ok := doc["name"].(string); ok {
		d.Name = name
	}
	if status, ok := doc["status"].(string); ok {
		d.Status = status
	}

	// Extract optional ConfigID
	if configID, ok := doc["configId"].(string); ok && configID != "" {
		d.ConfigID = sql.NullString{String: configID, Valid: true}
	}

	// Extract timestamps
	d.CreatedAt = extractDocTime(doc, "createdAt")
	d.UpdatedAt = extractDocTime(doc, "updatedAt")

	return d, nil
}

// documentToConfig converts a MongoDB document to a Config model.
func documentToConfig(doc Document) (*models.Config, error) {
	c := &models.Config{}

	// Extract ID
	c.ID = extractDocID(doc)

	// Extract string fields
	if name, ok := doc["name"].(string); ok {
		c.Name = name
	}
	if content, ok := doc["content"].(string); ok {
		c.Content = content
	}

	// Extract JSON fields
	if astJson := doc["astJson"]; astJson != nil {
		if data, err := json.Marshal(astJson); err == nil {
			c.ASTJSON = data
		}
	}
	if validationErrors := doc["validationErrors"]; validationErrors != nil {
		if data, err := json.Marshal(validationErrors); err == nil {
			c.ValidationErrors = data
		}
	}

	// Extract timestamps
	c.CreatedAt = extractDocTime(doc, "createdAt")

	return c, nil
}

// documentToExecution converts a MongoDB document to an Execution model.
func documentToExecution(doc Document) (*models.Execution, error) {
	e := &models.Execution{}

	// Extract ID
	e.ID = extractDocID(doc)

	// Extract string fields
	if deploymentID, ok := doc["deploymentId"].(string); ok {
		e.DeploymentID = deploymentID
	}
	if command, ok := doc["command"].(string); ok {
		e.Command = command
	}

	// Extract optional fields
	if output, ok := doc["output"].(string); ok {
		e.Output = sql.NullString{String: output, Valid: true}
	}
	if exitCode, ok := doc["exitCode"]; ok {
		switch v := exitCode.(type) {
		case int32:
			e.ExitCode = sql.NullInt32{Int32: v, Valid: true}
		case int64:
			e.ExitCode = sql.NullInt32{Int32: int32(v), Valid: true}
		case int:
			e.ExitCode = sql.NullInt32{Int32: int32(v), Valid: true}
		}
	}

	// Extract timestamps
	e.StartedAt = extractDocTime(doc, "startedAt")
	if completedAt := extractDocTimePtr(doc, "completedAt"); completedAt != nil {
		e.CompletedAt = sql.NullTime{Time: *completedAt, Valid: true}
	}

	return e, nil
}

// extractDocID extracts the ID from a MongoDB document.
func extractDocID(doc Document) string {
	if id, ok := doc["_id"]; ok {
		switch v := id.(type) {
		case primitive.ObjectID:
			return v.Hex()
		case string:
			return v
		}
	}
	return ""
}

// extractDocTime extracts a time.Time from a document field.
func extractDocTime(doc Document, field string) time.Time {
	if t, ok := doc[field]; ok {
		switch v := t.(type) {
		case time.Time:
			return v
		case primitive.DateTime:
			return v.Time()
		}
	}
	return time.Time{}
}

// extractDocTimePtr extracts a *time.Time from a document field.
func extractDocTimePtr(doc Document, field string) *time.Time {
	if t, ok := doc[field]; ok && t != nil {
		switch v := t.(type) {
		case time.Time:
			return &v
		case primitive.DateTime:
			t := v.Time()
			return &t
		}
	}
	return nil
}

// translateAdapterError converts MongoDB errors to standard errors.
func translateAdapterError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}
