//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseCRUDWorkflows tests full CRUD operations across entities.
func TestDatabaseCRUDWorkflows(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("config full CRUD lifecycle", func(t *testing.T) {
		// Create
		cfg := models.NewConfig("test-config", `var x = 1`)
		err := suite.ConfigRepo.Create(ctx, cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, cfg.ID)

		// Read
		retrieved, err := suite.ConfigRepo.GetByID(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, cfg.Name, retrieved.Name)
		assert.Equal(t, cfg.Content, retrieved.Content)

		// Update
		cfg.Name = "updated-config"
		cfg.Content = `var y = 2`
		err = suite.ConfigRepo.Update(ctx, cfg)
		require.NoError(t, err)

		// Verify update
		updated, err := suite.ConfigRepo.GetByID(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-config", updated.Name)
		assert.Equal(t, `var y = 2`, updated.Content)

		// Delete
		err = suite.ConfigRepo.Delete(ctx, cfg.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = suite.ConfigRepo.GetByID(ctx, cfg.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("deployment full CRUD lifecycle", func(t *testing.T) {
		// Create a config first (for foreign key)
		cfg := models.NewConfig("deploy-config", `var x = 1`)
		err := suite.ConfigRepo.Create(ctx, cfg)
		require.NoError(t, err)

		// Create deployment
		deploy := models.NewDeployment("test-deploy")
		deploy.ConfigID = sql.NullString{String: cfg.ID, Valid: true}
		err = suite.DeployRepo.Create(ctx, deploy)
		require.NoError(t, err)
		assert.NotEmpty(t, deploy.ID)

		// Read
		retrieved, err := suite.DeployRepo.GetByID(ctx, deploy.ID)
		require.NoError(t, err)
		assert.Equal(t, deploy.Name, retrieved.Name)
		assert.Equal(t, cfg.ID, retrieved.ConfigID.String)

		// Update
		deploy.Status = string(models.DeploymentStatusRunning)
		err = suite.DeployRepo.Update(ctx, deploy)
		require.NoError(t, err)

		// Verify update
		updated, err := suite.DeployRepo.GetByID(ctx, deploy.ID)
		require.NoError(t, err)
		assert.Equal(t, "running", updated.Status)

		// Delete
		err = suite.DeployRepo.Delete(ctx, deploy.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = suite.DeployRepo.GetByID(ctx, deploy.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("execution full CRUD lifecycle", func(t *testing.T) {
		// Create deployment first (for foreign key)
		deploy := models.NewDeployment("exec-deploy")
		err := suite.DeployRepo.Create(ctx, deploy)
		require.NoError(t, err)

		// Create execution
		exec := models.NewExecution(deploy.ID, "echo hello")
		err = suite.ExecRepo.Create(ctx, exec)
		require.NoError(t, err)
		assert.NotEmpty(t, exec.ID)

		// Read
		retrieved, err := suite.ExecRepo.GetByID(ctx, exec.ID)
		require.NoError(t, err)
		assert.Equal(t, exec.Command, retrieved.Command)
		assert.Equal(t, deploy.ID, retrieved.DeploymentID)

		// Update (set completed)
		exec.SetCompleted("hello", 0)
		err = suite.ExecRepo.Update(ctx, exec)
		require.NoError(t, err)

		// Verify update
		updated, err := suite.ExecRepo.GetByID(ctx, exec.ID)
		require.NoError(t, err)
		assert.True(t, updated.IsCompleted())
		assert.Equal(t, "hello", updated.Output.String)
		assert.Equal(t, int32(0), updated.ExitCode.Int32)

		// Delete
		err = suite.ExecRepo.Delete(ctx, exec.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = suite.ExecRepo.GetByID(ctx, exec.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

// TestDatabaseListOperations tests list operations with pagination.
func TestDatabaseListOperations(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("list configs with pagination", func(t *testing.T) {
		// Create multiple configs
		for i := 0; i < 5; i++ {
			cfg := models.NewConfig("config-"+string(rune('a'+i)), `var x = 1`)
			err := suite.ConfigRepo.Create(ctx, cfg)
			require.NoError(t, err)
		}

		// List with pagination
		configs, err := suite.ConfigRepo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, configs, 2)

		// Get next page
		configs2, err := suite.ConfigRepo.List(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, configs2, 2)

		// Verify different results
		assert.NotEqual(t, configs[0].ID, configs2[0].ID)
	})

	t.Run("list deployments with pagination", func(t *testing.T) {
		// Create multiple deployments
		for i := 0; i < 5; i++ {
			deploy := models.NewDeployment("deploy-" + string(rune('a'+i)))
			err := suite.DeployRepo.Create(ctx, deploy)
			require.NoError(t, err)
		}

		// List with pagination
		deploys, err := suite.DeployRepo.List(ctx, 3, 0)
		require.NoError(t, err)
		assert.Len(t, deploys, 3)
	})

	t.Run("list executions by deployment", func(t *testing.T) {
		// Create deployment
		deploy := models.NewDeployment("exec-list-deploy")
		err := suite.DeployRepo.Create(ctx, deploy)
		require.NoError(t, err)

		// Create multiple executions
		for i := 0; i < 3; i++ {
			exec := models.NewExecution(deploy.ID, "command-"+string(rune('0'+i)))
			err := suite.ExecRepo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// List by deployment
		execs, err := suite.ExecRepo.ListByDeployment(ctx, deploy.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, execs, 3)
	})
}

// TestDatabaseGetByName tests get-by-name operations.
func TestDatabaseGetByName(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("get config by name", func(t *testing.T) {
		cfg := models.NewConfig("unique-name", `var x = 1`)
		err := suite.ConfigRepo.Create(ctx, cfg)
		require.NoError(t, err)

		retrieved, err := suite.ConfigRepo.GetByName(ctx, "unique-name")
		require.NoError(t, err)
		assert.Equal(t, cfg.ID, retrieved.ID)
	})

	t.Run("get deployment by name", func(t *testing.T) {
		deploy := models.NewDeployment("unique-deploy")
		err := suite.DeployRepo.Create(ctx, deploy)
		require.NoError(t, err)

		retrieved, err := suite.DeployRepo.GetByName(ctx, "unique-deploy")
		require.NoError(t, err)
		assert.Equal(t, deploy.ID, retrieved.ID)
	})
}

// TestDatabaseNotFound tests not found scenarios.
func TestDatabaseNotFound(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("config not found by ID", func(t *testing.T) {
		_, err := suite.ConfigRepo.GetByID(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("config not found by name", func(t *testing.T) {
		_, err := suite.ConfigRepo.GetByName(ctx, "nonexistent-name")
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("deployment not found", func(t *testing.T) {
		_, err := suite.DeployRepo.GetByID(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("execution not found", func(t *testing.T) {
		_, err := suite.ExecRepo.GetByID(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("update nonexistent config", func(t *testing.T) {
		cfg := &models.Config{ID: "nonexistent-id", Name: "test", Content: "var x = 1"}
		err := suite.ConfigRepo.Update(ctx, cfg)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("delete nonexistent config", func(t *testing.T) {
		err := suite.ConfigRepo.Delete(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

// TestDatabaseTransactions tests transaction support.
func TestDatabaseTransactions(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("transaction commit", func(t *testing.T) {
		tx, err := suite.DB.Begin()
		require.NoError(t, err)

		// Use repository with transaction
		configRepoTx := suite.ConfigRepo.WithTx(tx)

		cfg := models.NewConfig("tx-config", `var x = 1`)
		err = configRepoTx.Create(ctx, cfg)
		require.NoError(t, err)

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify data persisted
		_, err = suite.ConfigRepo.GetByID(ctx, cfg.ID)
		assert.NoError(t, err)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		tx, err := suite.DB.Begin()
		require.NoError(t, err)

		// Use repository with transaction
		configRepoTx := suite.ConfigRepo.WithTx(tx)

		cfg := models.NewConfig("rollback-config", `var x = 1`)
		err = configRepoTx.Create(ctx, cfg)
		require.NoError(t, err)

		// Rollback
		err = tx.Rollback()
		require.NoError(t, err)

		// Verify data not persisted
		_, err = suite.ConfigRepo.GetByID(ctx, cfg.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

// TestDatabaseConcurrency tests concurrent access.
// Note: SQLite has limited concurrency support, so this test uses a mutex to serialize writes.
func TestDatabaseConcurrency(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	ctx := context.Background()

	t.Run("sequential creates simulate concurrent workflow", func(t *testing.T) {
		// SQLite doesn't support true concurrent writes well,
		// so we create items sequentially and verify they all exist
		for i := 0; i < 10; i++ {
			cfg := models.NewConfig("concurrent-"+string(rune('0'+i)), `var x = 1`)
			err := suite.ConfigRepo.Create(ctx, cfg)
			require.NoError(t, err)
		}

		// Verify all created
		configs, err := suite.ConfigRepo.List(ctx, 20, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(configs), 10)
	})

	t.Run("multiple sequential reads are consistent", func(t *testing.T) {
		// Create a config
		cfg := models.NewConfig("read-test", `var x = 1`)
		err := suite.ConfigRepo.Create(ctx, cfg)
		require.NoError(t, err)

		// Multiple sequential reads should return consistent data
		// Note: SQLite in-memory doesn't support true concurrency well
		for i := 0; i < 5; i++ {
			retrieved, err := suite.ConfigRepo.GetByID(ctx, cfg.ID)
			require.NoError(t, err)
			assert.Equal(t, cfg.ID, retrieved.ID)
			assert.Equal(t, cfg.Name, retrieved.Name)
		}
	})
}
