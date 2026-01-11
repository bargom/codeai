package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bargom/codeai/internal/database/models"
	dbtest "github.com/bargom/codeai/internal/database/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentRepository_Create(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("creates deployment successfully", func(t *testing.T) {
		deployment := models.NewDeployment("test-deployment")

		err := repo.Create(ctx, deployment)
		require.NoError(t, err)
		assert.NotEmpty(t, deployment.ID)
	})

	t.Run("fails on duplicate name", func(t *testing.T) {
		d1 := models.NewDeployment("duplicate-name")
		err := repo.Create(ctx, d1)
		require.NoError(t, err)

		d2 := models.NewDeployment("duplicate-name")
		err = repo.Create(ctx, d2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNIQUE constraint failed")
	})
}

func TestDeploymentRepository_GetByID(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("returns deployment by ID", func(t *testing.T) {
		deployment := models.NewDeployment("get-test")
		err := repo.Create(ctx, deployment)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, found.ID)
		assert.Equal(t, deployment.Name, found.Name)
		assert.Equal(t, deployment.Status, found.Status)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestDeploymentRepository_GetByName(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("returns deployment by name", func(t *testing.T) {
		deployment := models.NewDeployment("name-lookup-test")
		err := repo.Create(ctx, deployment)
		require.NoError(t, err)

		found, err := repo.GetByName(ctx, "name-lookup-test")
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, found.ID)
		assert.Equal(t, "name-lookup-test", found.Name)
	})

	t.Run("returns ErrNotFound for non-existent name", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "does-not-exist")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestDeploymentRepository_List(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	// Create test deployments
	for i := 0; i < 5; i++ {
		d := models.NewDeployment("list-test-" + string(rune('a'+i)))
		err := repo.Create(ctx, d)
		require.NoError(t, err)
	}

	t.Run("returns all deployments", func(t *testing.T) {
		deployments, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, deployments, 5)
	})

	t.Run("respects limit", func(t *testing.T) {
		deployments, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, deployments, 2)
	})

	t.Run("respects offset", func(t *testing.T) {
		deployments, err := repo.List(ctx, 10, 3)
		require.NoError(t, err)
		assert.Len(t, deployments, 2)
	})
}

func TestDeploymentRepository_Update(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("updates deployment successfully", func(t *testing.T) {
		deployment := models.NewDeployment("update-test")
		err := repo.Create(ctx, deployment)
		require.NoError(t, err)

		deployment.Status = string(models.DeploymentStatusRunning)
		err = repo.Update(ctx, deployment)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, string(models.DeploymentStatusRunning), found.Status)
	})

	t.Run("returns ErrNotFound for non-existent deployment", func(t *testing.T) {
		deployment := &models.Deployment{
			ID:     "non-existent",
			Name:   "test",
			Status: "pending",
		}
		err := repo.Update(ctx, deployment)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestDeploymentRepository_Delete(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("deletes deployment successfully", func(t *testing.T) {
		deployment := models.NewDeployment("delete-test")
		err := repo.Create(ctx, deployment)
		require.NoError(t, err)

		err = repo.Delete(ctx, deployment.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, deployment.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestDeploymentRepository_WithConfig(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	deploymentRepo := NewDeploymentRepository(db)
	configRepo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("creates deployment with config reference", func(t *testing.T) {
		// Create config first
		config := models.NewConfig("test-config", "content here")
		err := configRepo.Create(ctx, config)
		require.NoError(t, err)

		// Create deployment with config reference
		deployment := models.NewDeployment("with-config")
		deployment.ConfigID = sql.NullString{String: config.ID, Valid: true}
		err = deploymentRepo.Create(ctx, deployment)
		require.NoError(t, err)

		// Verify the reference
		found, err := deploymentRepo.GetByID(ctx, deployment.ID)
		require.NoError(t, err)
		assert.True(t, found.ConfigID.Valid)
		assert.Equal(t, config.ID, found.ConfigID.String)
	})

	t.Run("fails with invalid config reference", func(t *testing.T) {
		deployment := models.NewDeployment("invalid-config-ref")
		deployment.ConfigID = sql.NullString{String: "invalid-id", Valid: true}
		err := deploymentRepo.Create(ctx, deployment)
		assert.Error(t, err)
	})
}

func TestDeploymentRepository_Transaction(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	t.Run("commits transaction successfully", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		txRepo := repo.WithTx(tx)
		deployment := models.NewDeployment("tx-commit-test")
		err = txRepo.Create(ctx, deployment)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Verify the deployment exists
		found, err := repo.GetByID(ctx, deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, found.ID)
	})

	t.Run("rolls back transaction on error", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		txRepo := repo.WithTx(tx)
		deployment := models.NewDeployment("tx-rollback-test")
		err = txRepo.Create(ctx, deployment)
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		// Verify the deployment does not exist
		_, err = repo.GetByID(ctx, deployment.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
