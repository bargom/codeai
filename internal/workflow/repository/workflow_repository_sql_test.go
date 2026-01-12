package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, *SQLWorkflowRepository) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	repo := NewSQLWorkflowRepository(db)
	err = repo.CreateTable(context.Background())
	require.NoError(t, err)

	return db, repo
}

func TestSQLWorkflowRepository_CreateTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo := NewSQLWorkflowRepository(db)
	err = repo.CreateTable(context.Background())
	require.NoError(t, err)

	// Verify table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM workflow_executions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSQLWorkflowRepository_SaveExecution(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-123",
		WorkflowType: "ai-pipeline",
		Status:       StatusPending,
		Input:        json.RawMessage(`{"test": "input"}`),
		StartedAt:    time.Now(),
		Metadata: map[string]string{
			"env": "test",
		},
	}

	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)
	assert.NotEmpty(t, exec.ID)

	// Verify saved
	saved, err := repo.GetExecution(context.Background(), exec.ID)
	require.NoError(t, err)
	assert.Equal(t, exec.WorkflowID, saved.WorkflowID)
	assert.Equal(t, exec.WorkflowType, saved.WorkflowType)
	assert.Equal(t, exec.Status, saved.Status)
}

func TestSQLWorkflowRepository_GetExecution(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	t.Run("existing execution", func(t *testing.T) {
		exec := &WorkflowExecution{
			WorkflowID:   "wf-get-test",
			WorkflowType: "test-suite",
			Status:       StatusRunning,
			StartedAt:    time.Now(),
		}
		err := repo.SaveExecution(context.Background(), exec)
		require.NoError(t, err)

		found, err := repo.GetExecution(context.Background(), exec.ID)
		require.NoError(t, err)
		assert.Equal(t, exec.WorkflowID, found.WorkflowID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetExecution(context.Background(), "non-existent")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestSQLWorkflowRepository_GetExecutionByWorkflowID(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-by-wfid",
		WorkflowType: "ai-pipeline",
		Status:       StatusCompleted,
		StartedAt:    time.Now(),
	}
	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)

	found, err := repo.GetExecutionByWorkflowID(context.Background(), "wf-by-wfid")
	require.NoError(t, err)
	assert.Equal(t, exec.ID, found.ID)
}

func TestSQLWorkflowRepository_ListExecutions(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	// Create test data
	for i := 0; i < 5; i++ {
		status := StatusCompleted
		if i%2 == 0 {
			status = StatusFailed
		}
		exec := &WorkflowExecution{
			WorkflowID:   "wf-list-" + string(rune('a'+i)),
			WorkflowType: "ai-pipeline",
			Status:       status,
			StartedAt:    time.Now(),
		}
		err := repo.SaveExecution(context.Background(), exec)
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		executions, err := repo.ListExecutions(context.Background(), Filter{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, executions, 5)
	})

	t.Run("filter by status", func(t *testing.T) {
		executions, err := repo.ListExecutions(context.Background(), Filter{
			Status: StatusFailed,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.Len(t, executions, 3) // indices 0, 2, 4
	})

	t.Run("with limit", func(t *testing.T) {
		executions, err := repo.ListExecutions(context.Background(), Filter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})

	t.Run("with offset", func(t *testing.T) {
		executions, err := repo.ListExecutions(context.Background(), Filter{
			Limit:  10,
			Offset: 3,
		})
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})
}

func TestSQLWorkflowRepository_UpdateStatus(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-status",
		WorkflowType: "ai-pipeline",
		Status:       StatusRunning,
		StartedAt:    time.Now(),
	}
	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)

	t.Run("update to completed", func(t *testing.T) {
		err := repo.UpdateStatus(context.Background(), exec.ID, StatusCompleted, "")
		require.NoError(t, err)

		updated, err := repo.GetExecution(context.Background(), exec.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, updated.Status)
		assert.NotNil(t, updated.CompletedAt)
	})

	t.Run("update to failed with error", func(t *testing.T) {
		err := repo.UpdateStatus(context.Background(), exec.ID, StatusFailed, "test error")
		require.NoError(t, err)

		updated, err := repo.GetExecution(context.Background(), exec.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusFailed, updated.Status)
		assert.Equal(t, "test error", updated.Error)
	})
}

func TestSQLWorkflowRepository_UpdateOutput(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-output",
		WorkflowType: "ai-pipeline",
		Status:       StatusCompleted,
		StartedAt:    time.Now(),
	}
	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)

	output := json.RawMessage(`{"result": "success"}`)
	err = repo.UpdateOutput(context.Background(), exec.ID, output)
	require.NoError(t, err)

	updated, err := repo.GetExecution(context.Background(), exec.ID)
	require.NoError(t, err)
	assert.JSONEq(t, `{"result": "success"}`, string(updated.Output))
}

func TestSQLWorkflowRepository_UpdateCompensations(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-comp",
		WorkflowType: "ai-pipeline",
		Status:       StatusFailed,
		StartedAt:    time.Now(),
	}
	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)

	compensations := []CompensationRecord{
		{Name: "cleanup-files", Status: "completed"},
		{Name: "notify-users", Status: "completed"},
	}
	err = repo.UpdateCompensations(context.Background(), exec.ID, compensations)
	require.NoError(t, err)

	updated, err := repo.GetExecution(context.Background(), exec.ID)
	require.NoError(t, err)
	assert.Len(t, updated.Compensations, 2)
	assert.Equal(t, "cleanup-files", updated.Compensations[0].Name)
}

func TestSQLWorkflowRepository_DeleteExecution(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	exec := &WorkflowExecution{
		WorkflowID:   "wf-delete",
		WorkflowType: "ai-pipeline",
		Status:       StatusCompleted,
		StartedAt:    time.Now(),
	}
	err := repo.SaveExecution(context.Background(), exec)
	require.NoError(t, err)

	t.Run("delete existing", func(t *testing.T) {
		err := repo.DeleteExecution(context.Background(), exec.ID)
		require.NoError(t, err)

		_, err = repo.GetExecution(context.Background(), exec.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := repo.DeleteExecution(context.Background(), "non-existent")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestSQLWorkflowRepository_CountByStatus(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	// Create test data
	statuses := []Status{StatusCompleted, StatusCompleted, StatusFailed, StatusRunning}
	for i, status := range statuses {
		exec := &WorkflowExecution{
			WorkflowID:   "wf-count-" + string(rune('a'+i)),
			WorkflowType: "ai-pipeline",
			Status:       status,
			StartedAt:    time.Now(),
		}
		err := repo.SaveExecution(context.Background(), exec)
		require.NoError(t, err)
	}

	count, err := repo.CountByStatus(context.Background(), StatusCompleted)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = repo.CountByStatus(context.Background(), StatusFailed)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = repo.CountByStatus(context.Background(), StatusPending)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
