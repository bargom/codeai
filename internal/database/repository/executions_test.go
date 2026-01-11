package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/database/models"
	dbtest "github.com/bargom/codeai/internal/database/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExecutionTest(t *testing.T, db *sql.DB) *models.Deployment {
	t.Helper()

	// Create a deployment first (executions need a deployment reference)
	deploymentRepo := NewDeploymentRepository(db)
	deployment := models.NewDeployment("exec-test-deployment")
	err := deploymentRepo.Create(context.Background(), deployment)
	require.NoError(t, err)
	return deployment
}

func TestExecutionRepository_Create(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	t.Run("creates execution successfully", func(t *testing.T) {
		execution := models.NewExecution(deployment.ID, "echo hello")

		err := repo.Create(ctx, execution)
		require.NoError(t, err)
		assert.NotEmpty(t, execution.ID)
	})

	t.Run("fails with invalid deployment reference", func(t *testing.T) {
		execution := models.NewExecution("invalid-deployment-id", "echo hello")

		err := repo.Create(ctx, execution)
		assert.Error(t, err)
	})
}

func TestExecutionRepository_GetByID(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	t.Run("returns execution by ID", func(t *testing.T) {
		execution := models.NewExecution(deployment.ID, "echo hello")
		err := repo.Create(ctx, execution)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, execution.ID)
		require.NoError(t, err)
		assert.Equal(t, execution.ID, found.ID)
		assert.Equal(t, execution.DeploymentID, found.DeploymentID)
		assert.Equal(t, execution.Command, found.Command)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestExecutionRepository_List(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	// Create test executions
	for i := 0; i < 5; i++ {
		e := models.NewExecution(deployment.ID, "command-"+string(rune('a'+i)))
		err := repo.Create(ctx, e)
		require.NoError(t, err)
	}

	t.Run("returns all executions", func(t *testing.T) {
		executions, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, executions, 5)
	})

	t.Run("respects limit", func(t *testing.T) {
		executions, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})

	t.Run("respects offset", func(t *testing.T) {
		executions, err := repo.List(ctx, 10, 3)
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})
}

func TestExecutionRepository_ListByDeployment(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment1 := setupExecutionTest(t, db)

	// Create second deployment
	deploymentRepo := NewDeploymentRepository(db)
	deployment2 := models.NewDeployment("exec-test-deployment-2")
	err := deploymentRepo.Create(context.Background(), deployment2)
	require.NoError(t, err)

	repo := NewExecutionRepository(db)
	ctx := context.Background()

	// Create executions for both deployments
	for i := 0; i < 3; i++ {
		e := models.NewExecution(deployment1.ID, "d1-cmd-"+string(rune('a'+i)))
		err := repo.Create(ctx, e)
		require.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		e := models.NewExecution(deployment2.ID, "d2-cmd-"+string(rune('a'+i)))
		err := repo.Create(ctx, e)
		require.NoError(t, err)
	}

	t.Run("returns only executions for specified deployment", func(t *testing.T) {
		executions, err := repo.ListByDeployment(ctx, deployment1.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, executions, 3)

		for _, e := range executions {
			assert.Equal(t, deployment1.ID, e.DeploymentID)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		executions, err := repo.ListByDeployment(ctx, deployment1.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, executions, 2)
	})
}

func TestExecutionRepository_Update(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	t.Run("updates execution with completion info", func(t *testing.T) {
		execution := models.NewExecution(deployment.ID, "echo hello")
		err := repo.Create(ctx, execution)
		require.NoError(t, err)

		execution.SetCompleted("hello\n", 0)
		err = repo.Update(ctx, execution)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, execution.ID)
		require.NoError(t, err)
		assert.True(t, found.Output.Valid)
		assert.Equal(t, "hello\n", found.Output.String)
		assert.True(t, found.ExitCode.Valid)
		assert.Equal(t, int32(0), found.ExitCode.Int32)
		assert.True(t, found.CompletedAt.Valid)
	})

	t.Run("updates execution with non-zero exit code", func(t *testing.T) {
		execution := models.NewExecution(deployment.ID, "exit 1")
		err := repo.Create(ctx, execution)
		require.NoError(t, err)

		execution.Output = sql.NullString{String: "error", Valid: true}
		execution.ExitCode = sql.NullInt32{Int32: 1, Valid: true}
		execution.CompletedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
		err = repo.Update(ctx, execution)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, execution.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(1), found.ExitCode.Int32)
	})

	t.Run("returns ErrNotFound for non-existent execution", func(t *testing.T) {
		execution := &models.Execution{
			ID:           "non-existent",
			DeploymentID: deployment.ID,
			Command:      "test",
		}
		err := repo.Update(ctx, execution)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestExecutionRepository_Delete(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	t.Run("deletes execution successfully", func(t *testing.T) {
		execution := models.NewExecution(deployment.ID, "echo hello")
		err := repo.Create(ctx, execution)
		require.NoError(t, err)

		err = repo.Delete(ctx, execution.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, execution.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestExecutionRepository_Transaction(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deployment := setupExecutionTest(t, db)
	repo := NewExecutionRepository(db)
	ctx := context.Background()

	t.Run("rolls back transaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		txRepo := repo.WithTx(tx)
		execution := models.NewExecution(deployment.ID, "echo test")
		err = txRepo.Create(ctx, execution)
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, execution.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
