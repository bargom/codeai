package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bargom/codeai/internal/database/models"
	dbtest "github.com/bargom/codeai/internal/database/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRepository_Create(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("creates config successfully", func(t *testing.T) {
		config := models.NewConfig("test-config", "deployment test {}")

		err := repo.Create(ctx, config)
		require.NoError(t, err)
		assert.NotEmpty(t, config.ID)
	})

	t.Run("creates config with AST JSON", func(t *testing.T) {
		config := models.NewConfig("config-with-ast", "deployment test {}")
		config.ASTJSON = json.RawMessage(`{"type": "deployment", "name": "test"}`)

		err := repo.Create(ctx, config)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, config.ID)
		require.NoError(t, err)
		assert.JSONEq(t, `{"type": "deployment", "name": "test"}`, string(found.ASTJSON))
	})

	t.Run("creates config with validation errors", func(t *testing.T) {
		config := models.NewConfig("config-with-errors", "invalid content")
		config.ValidationErrors = json.RawMessage(`[{"line": 1, "message": "invalid syntax"}]`)

		err := repo.Create(ctx, config)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, config.ID)
		require.NoError(t, err)
		assert.JSONEq(t, `[{"line": 1, "message": "invalid syntax"}]`, string(found.ValidationErrors))
	})
}

func TestConfigRepository_GetByID(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("returns config by ID", func(t *testing.T) {
		config := models.NewConfig("get-test", "content here")
		err := repo.Create(ctx, config)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, config.ID)
		require.NoError(t, err)
		assert.Equal(t, config.ID, found.ID)
		assert.Equal(t, config.Name, found.Name)
		assert.Equal(t, config.Content, found.Content)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestConfigRepository_GetByName(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("returns config by name", func(t *testing.T) {
		config := models.NewConfig("unique-name", "content")
		err := repo.Create(ctx, config)
		require.NoError(t, err)

		found, err := repo.GetByName(ctx, "unique-name")
		require.NoError(t, err)
		assert.Equal(t, config.ID, found.ID)
	})

	t.Run("returns ErrNotFound for non-existent name", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "does-not-exist")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestConfigRepository_List(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	// Create test configs
	for i := 0; i < 5; i++ {
		c := models.NewConfig("list-test-"+string(rune('a'+i)), "content")
		err := repo.Create(ctx, c)
		require.NoError(t, err)
	}

	t.Run("returns all configs", func(t *testing.T) {
		configs, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, configs, 5)
	})

	t.Run("respects limit", func(t *testing.T) {
		configs, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, configs, 2)
	})

	t.Run("respects offset", func(t *testing.T) {
		configs, err := repo.List(ctx, 10, 3)
		require.NoError(t, err)
		assert.Len(t, configs, 2)
	})
}

func TestConfigRepository_Update(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("updates config successfully", func(t *testing.T) {
		config := models.NewConfig("update-test", "original content")
		err := repo.Create(ctx, config)
		require.NoError(t, err)

		config.Content = "updated content"
		config.ASTJSON = json.RawMessage(`{"updated": true}`)
		err = repo.Update(ctx, config)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, config.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated content", found.Content)
		assert.JSONEq(t, `{"updated": true}`, string(found.ASTJSON))
	})

	t.Run("returns ErrNotFound for non-existent config", func(t *testing.T) {
		config := &models.Config{
			ID:      "non-existent",
			Name:    "test",
			Content: "content",
		}
		err := repo.Update(ctx, config)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestConfigRepository_Delete(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("deletes config successfully", func(t *testing.T) {
		config := models.NewConfig("delete-test", "content")
		err := repo.Create(ctx, config)
		require.NoError(t, err)

		err = repo.Delete(ctx, config.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, config.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotFound for non-existent ID", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestConfigRepository_Transaction(t *testing.T) {
	db := dbtest.SetupTestDB(t)
	defer dbtest.TeardownTestDB(t, db)

	repo := NewConfigRepository(db)
	ctx := context.Background()

	t.Run("rolls back transaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		txRepo := repo.WithTx(tx)
		config := models.NewConfig("tx-test", "content")
		err = txRepo.Create(ctx, config)
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, config.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
