// Package compensation provides compensation (rollback) activities for workflow steps.
package compensation

import (
	"context"
	"encoding/json"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// RollbackInput provides context for rollback operations.
type RollbackInput struct {
	WorkflowID   string            `json:"workflowId"`
	ActivityName string            `json:"activityName"`
	ResourceID   string            `json:"resourceId,omitempty"`
	ResourceType string            `json:"resourceType,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	OriginalData json.RawMessage   `json:"originalData,omitempty"`
}

// RollbackOutput provides the result of a rollback operation.
type RollbackOutput struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Rollback bool   `json:"rollback"` // Indicates this was a rollback operation
}

// AgentRollbackInput for undoing agent execution side effects.
type AgentRollbackInput struct {
	RollbackInput
	AgentType  string          `json:"agentType"`
	AgentName  string          `json:"agentName"`
	OutputPath string          `json:"outputPath,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
}

// FileRollbackInput for undoing file operations.
type FileRollbackInput struct {
	RollbackInput
	FilePath       string          `json:"filePath"`
	Operation      string          `json:"operation"` // "create", "update", "delete"
	OriginalContent json.RawMessage `json:"originalContent,omitempty"`
	BackupPath     string          `json:"backupPath,omitempty"`
}

// APIRollbackInput for compensating external API calls.
type APIRollbackInput struct {
	RollbackInput
	Endpoint         string            `json:"endpoint"`
	Method           string            `json:"method"`
	RequestID        string            `json:"requestId,omitempty"`
	CompensateURL    string            `json:"compensateUrl,omitempty"`
	CompensateMethod string            `json:"compensateMethod,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
}

// NotificationRollbackInput for canceling scheduled notifications.
type NotificationRollbackInput struct {
	RollbackInput
	NotificationType string   `json:"notificationType"` // "email", "webhook", "sms"
	NotificationID   string   `json:"notificationId"`
	Recipients       []string `json:"recipients,omitempty"`
}

// DatabaseRollbackInput for undoing database changes.
type DatabaseRollbackInput struct {
	RollbackInput
	Collection   string          `json:"collection"`
	DocumentID   string          `json:"documentId"`
	Operation    string          `json:"operation"` // "insert", "update", "delete"
	PreviousData json.RawMessage `json:"previousData,omitempty"`
}

// CompensationActivities holds compensation activity implementations.
type CompensationActivities struct {
	agentRollbackHandler        AgentRollbackHandler
	fileRollbackHandler         FileRollbackHandler
	apiRollbackHandler          APIRollbackHandler
	notificationRollbackHandler NotificationRollbackHandler
	databaseRollbackHandler     DatabaseRollbackHandler
}

// AgentRollbackHandler handles agent execution rollbacks.
type AgentRollbackHandler interface {
	RollbackAgent(ctx context.Context, input AgentRollbackInput) error
}

// FileRollbackHandler handles file operation rollbacks.
type FileRollbackHandler interface {
	RollbackFile(ctx context.Context, input FileRollbackInput) error
}

// APIRollbackHandler handles external API call rollbacks.
type APIRollbackHandler interface {
	RollbackAPI(ctx context.Context, input APIRollbackInput) error
}

// NotificationRollbackHandler handles notification rollbacks.
type NotificationRollbackHandler interface {
	RollbackNotification(ctx context.Context, input NotificationRollbackInput) error
}

// DatabaseRollbackHandler handles database rollbacks.
type DatabaseRollbackHandler interface {
	RollbackDatabase(ctx context.Context, input DatabaseRollbackInput) error
}

// NewCompensationActivities creates a new CompensationActivities instance.
func NewCompensationActivities(
	agentHandler AgentRollbackHandler,
	fileHandler FileRollbackHandler,
	apiHandler APIRollbackHandler,
	notifHandler NotificationRollbackHandler,
	dbHandler DatabaseRollbackHandler,
) *CompensationActivities {
	return &CompensationActivities{
		agentRollbackHandler:        agentHandler,
		fileRollbackHandler:         fileHandler,
		apiRollbackHandler:          apiHandler,
		notificationRollbackHandler: notifHandler,
		databaseRollbackHandler:     dbHandler,
	}
}

// RollbackAgentExecution undoes agent execution side effects.
func (ca *CompensationActivities) RollbackAgentExecution(ctx context.Context, input AgentRollbackInput) (RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back agent %s (attempt %d)", input.AgentName, info.Attempt))

	if ca.agentRollbackHandler == nil {
		return RollbackOutput{
			Success:  true,
			Message:  "no agent rollback handler configured, skipping",
			Rollback: true,
		}, nil
	}

	err := ca.agentRollbackHandler.RollbackAgent(ctx, input)
	if err != nil {
		return RollbackOutput{
			Success:  false,
			Error:    err.Error(),
			Rollback: true,
		}, err
	}

	return RollbackOutput{
		Success:  true,
		Message:  fmt.Sprintf("agent %s rollback completed", input.AgentName),
		Rollback: true,
	}, nil
}

// RollbackFileOperation undoes file creation/modification.
func (ca *CompensationActivities) RollbackFileOperation(ctx context.Context, input FileRollbackInput) (RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back file operation %s (attempt %d)", input.FilePath, info.Attempt))

	if ca.fileRollbackHandler == nil {
		return RollbackOutput{
			Success:  true,
			Message:  "no file rollback handler configured, skipping",
			Rollback: true,
		}, nil
	}

	err := ca.fileRollbackHandler.RollbackFile(ctx, input)
	if err != nil {
		return RollbackOutput{
			Success:  false,
			Error:    err.Error(),
			Rollback: true,
		}, err
	}

	return RollbackOutput{
		Success:  true,
		Message:  fmt.Sprintf("file %s rollback completed", input.FilePath),
		Rollback: true,
	}, nil
}

// RollbackExternalAPICall compensates external API calls.
func (ca *CompensationActivities) RollbackExternalAPICall(ctx context.Context, input APIRollbackInput) (RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back API call %s (attempt %d)", input.RequestID, info.Attempt))

	if ca.apiRollbackHandler == nil {
		return RollbackOutput{
			Success:  true,
			Message:  "no API rollback handler configured, skipping",
			Rollback: true,
		}, nil
	}

	err := ca.apiRollbackHandler.RollbackAPI(ctx, input)
	if err != nil {
		return RollbackOutput{
			Success:  false,
			Error:    err.Error(),
			Rollback: true,
		}, err
	}

	return RollbackOutput{
		Success:  true,
		Message:  fmt.Sprintf("API call %s rollback completed", input.RequestID),
		Rollback: true,
	}, nil
}

// RollbackNotification cancels scheduled notifications.
func (ca *CompensationActivities) RollbackNotification(ctx context.Context, input NotificationRollbackInput) (RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back notification %s (attempt %d)", input.NotificationID, info.Attempt))

	if ca.notificationRollbackHandler == nil {
		return RollbackOutput{
			Success:  true,
			Message:  "no notification rollback handler configured, skipping",
			Rollback: true,
		}, nil
	}

	err := ca.notificationRollbackHandler.RollbackNotification(ctx, input)
	if err != nil {
		return RollbackOutput{
			Success:  false,
			Error:    err.Error(),
			Rollback: true,
		}, err
	}

	return RollbackOutput{
		Success:  true,
		Message:  fmt.Sprintf("notification %s rollback completed", input.NotificationID),
		Rollback: true,
	}, nil
}

// RollbackDatabaseOperation undoes database changes.
func (ca *CompensationActivities) RollbackDatabaseOperation(ctx context.Context, input DatabaseRollbackInput) (RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back database operation on %s (attempt %d)", input.DocumentID, info.Attempt))

	if ca.databaseRollbackHandler == nil {
		return RollbackOutput{
			Success:  true,
			Message:  "no database rollback handler configured, skipping",
			Rollback: true,
		}, nil
	}

	err := ca.databaseRollbackHandler.RollbackDatabase(ctx, input)
	if err != nil {
		return RollbackOutput{
			Success:  false,
			Error:    err.Error(),
			Rollback: true,
		}, err
	}

	return RollbackOutput{
		Success:  true,
		Message:  fmt.Sprintf("database document %s rollback completed", input.DocumentID),
		Rollback: true,
	}, nil
}
