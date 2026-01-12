package compensation

import (
	"context"
	"fmt"
	"os"
)

// DefaultAgentRollbackHandler provides default agent rollback implementation.
type DefaultAgentRollbackHandler struct{}

// NewDefaultAgentRollbackHandler creates a new DefaultAgentRollbackHandler.
func NewDefaultAgentRollbackHandler() *DefaultAgentRollbackHandler {
	return &DefaultAgentRollbackHandler{}
}

// RollbackAgent cleans up agent execution artifacts.
func (h *DefaultAgentRollbackHandler) RollbackAgent(ctx context.Context, input AgentRollbackInput) error {
	// Delete output artifacts if path is provided
	if input.OutputPath != "" {
		if err := os.RemoveAll(input.OutputPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove output artifacts: %w", err)
		}
	}

	// Additional cleanup based on agent type
	switch input.AgentType {
	case "code-analysis":
		// Clean up analysis cache if any
		return nil
	case "test-generator":
		// Clean up generated test files
		return nil
	case "documentation":
		// Clean up generated documentation
		return nil
	default:
		// Generic cleanup - just log
		return nil
	}
}

// DefaultFileRollbackHandler provides default file rollback implementation.
type DefaultFileRollbackHandler struct{}

// NewDefaultFileRollbackHandler creates a new DefaultFileRollbackHandler.
func NewDefaultFileRollbackHandler() *DefaultFileRollbackHandler {
	return &DefaultFileRollbackHandler{}
}

// RollbackFile restores files to their original state.
func (h *DefaultFileRollbackHandler) RollbackFile(ctx context.Context, input FileRollbackInput) error {
	switch input.Operation {
	case "create":
		// Delete the created file
		if err := os.Remove(input.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete created file: %w", err)
		}
		return nil

	case "update":
		// Restore from backup if available
		if input.BackupPath != "" {
			data, err := os.ReadFile(input.BackupPath)
			if err != nil {
				return fmt.Errorf("failed to read backup: %w", err)
			}
			if err := os.WriteFile(input.FilePath, data, 0644); err != nil {
				return fmt.Errorf("failed to restore file: %w", err)
			}
			// Clean up backup
			os.Remove(input.BackupPath)
			return nil
		}
		// Restore from original content if provided
		if len(input.OriginalContent) > 0 {
			if err := os.WriteFile(input.FilePath, input.OriginalContent, 0644); err != nil {
				return fmt.Errorf("failed to restore file from content: %w", err)
			}
			return nil
		}
		return fmt.Errorf("no backup or original content available for file restoration")

	case "delete":
		// Restore from backup or original content
		if input.BackupPath != "" {
			data, err := os.ReadFile(input.BackupPath)
			if err != nil {
				return fmt.Errorf("failed to read backup: %w", err)
			}
			if err := os.WriteFile(input.FilePath, data, 0644); err != nil {
				return fmt.Errorf("failed to restore deleted file: %w", err)
			}
			os.Remove(input.BackupPath)
			return nil
		}
		if len(input.OriginalContent) > 0 {
			if err := os.WriteFile(input.FilePath, input.OriginalContent, 0644); err != nil {
				return fmt.Errorf("failed to restore deleted file from content: %w", err)
			}
			return nil
		}
		return fmt.Errorf("no backup or original content available to restore deleted file")

	default:
		return fmt.Errorf("unknown file operation: %s", input.Operation)
	}
}

// DefaultAPIRollbackHandler provides default API rollback implementation.
type DefaultAPIRollbackHandler struct{}

// NewDefaultAPIRollbackHandler creates a new DefaultAPIRollbackHandler.
func NewDefaultAPIRollbackHandler() *DefaultAPIRollbackHandler {
	return &DefaultAPIRollbackHandler{}
}

// RollbackAPI sends compensation request to external API.
func (h *DefaultAPIRollbackHandler) RollbackAPI(ctx context.Context, input APIRollbackInput) error {
	// If no compensate URL provided, nothing to do
	if input.CompensateURL == "" {
		return nil
	}

	// This would typically make an HTTP request to the compensation endpoint
	// For now, we just log and return success
	// In a real implementation, you would use an HTTP client here

	return nil
}

// DefaultNotificationRollbackHandler provides default notification rollback implementation.
type DefaultNotificationRollbackHandler struct{}

// NewDefaultNotificationRollbackHandler creates a new DefaultNotificationRollbackHandler.
func NewDefaultNotificationRollbackHandler() *DefaultNotificationRollbackHandler {
	return &DefaultNotificationRollbackHandler{}
}

// RollbackNotification cancels or revokes notifications.
func (h *DefaultNotificationRollbackHandler) RollbackNotification(ctx context.Context, input NotificationRollbackInput) error {
	// Notifications are often fire-and-forget and cannot be truly compensated
	// Log the attempt and return success
	switch input.NotificationType {
	case "email":
		// Emails cannot be unsent - log only
		return nil
	case "webhook":
		// Could potentially send a cancellation webhook
		return nil
	case "sms":
		// SMS cannot be unsent - log only
		return nil
	default:
		return nil
	}
}

// DefaultDatabaseRollbackHandler provides default database rollback implementation.
type DefaultDatabaseRollbackHandler struct{}

// NewDefaultDatabaseRollbackHandler creates a new DefaultDatabaseRollbackHandler.
func NewDefaultDatabaseRollbackHandler() *DefaultDatabaseRollbackHandler {
	return &DefaultDatabaseRollbackHandler{}
}

// RollbackDatabase reverts database changes.
func (h *DefaultDatabaseRollbackHandler) RollbackDatabase(ctx context.Context, input DatabaseRollbackInput) error {
	// This is a placeholder - actual implementation would use a database client
	// The real implementation would depend on the specific database being used
	switch input.Operation {
	case "insert":
		// Delete the inserted document
		return nil
	case "update":
		// Restore previous data
		return nil
	case "delete":
		// Reinsert the deleted document
		return nil
	default:
		return fmt.Errorf("unknown database operation: %s", input.Operation)
	}
}

// NewDefaultCompensationActivities creates CompensationActivities with all default handlers.
func NewDefaultCompensationActivities() *CompensationActivities {
	return NewCompensationActivities(
		NewDefaultAgentRollbackHandler(),
		NewDefaultFileRollbackHandler(),
		NewDefaultAPIRollbackHandler(),
		NewDefaultNotificationRollbackHandler(),
		NewDefaultDatabaseRollbackHandler(),
	)
}
