package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDatabaseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected DatabaseType
	}{
		{"postgres lowercase", "postgres", DatabaseTypePostgres},
		{"postgres uppercase", "POSTGRES", DatabaseTypePostgres},
		{"postgresql", "postgresql", DatabaseTypePostgres},
		{"pg shorthand", "pg", DatabaseTypePostgres},
		{"mongodb lowercase", "mongodb", DatabaseTypeMongoDB},
		{"mongodb uppercase", "MONGODB", DatabaseTypeMongoDB},
		{"mongo shorthand", "mongo", DatabaseTypeMongoDB},
		{"empty defaults to postgres", "", DatabaseTypePostgres},
		{"unknown defaults to postgres", "unknown", DatabaseTypePostgres},
		{"whitespace trimmed", " postgres ", DatabaseTypePostgres},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ParseDatabaseType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseTypeIsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, DatabaseTypePostgres.IsValid())
	assert.True(t, DatabaseTypeMongoDB.IsValid())
	assert.False(t, DatabaseType("invalid").IsValid())
}

func TestDefaultDatabaseConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultDatabaseConfig()
	assert.Equal(t, DatabaseTypePostgres, cfg.Type)
	assert.NotNil(t, cfg.Postgres)
	assert.NotNil(t, cfg.MongoDB)
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, 5432, cfg.Postgres.Port)
	assert.Equal(t, "mongodb://localhost:27017", cfg.MongoDB.URI)
}

func TestDatabaseConfigFromEnv(t *testing.T) {
	// Save original env values to restore later
	origDBType := os.Getenv("DATABASE_TYPE")
	origDBHost := os.Getenv("DB_HOST")
	origMongoURI := os.Getenv("MONGODB_URI")

	// Clean up after test
	defer func() {
		os.Setenv("DATABASE_TYPE", origDBType)
		os.Setenv("DB_HOST", origDBHost)
		os.Setenv("MONGODB_URI", origMongoURI)
	}()

	t.Run("defaults when no env vars set", func(t *testing.T) {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("MONGODB_URI")

		cfg := DatabaseConfigFromEnv()
		assert.Equal(t, DatabaseTypePostgres, cfg.Type)
		assert.Equal(t, "localhost", cfg.Postgres.Host)
	})

	t.Run("respects DATABASE_TYPE", func(t *testing.T) {
		os.Setenv("DATABASE_TYPE", "mongodb")
		cfg := DatabaseConfigFromEnv()
		assert.Equal(t, DatabaseTypeMongoDB, cfg.Type)
	})

	t.Run("respects DB_HOST", func(t *testing.T) {
		os.Setenv("DB_HOST", "custom-host")
		cfg := DatabaseConfigFromEnv()
		assert.Equal(t, "custom-host", cfg.Postgres.Host)
	})

	t.Run("respects MONGODB_URI", func(t *testing.T) {
		os.Setenv("MONGODB_URI", "mongodb://custom:27017")
		cfg := DatabaseConfigFromEnv()
		assert.Equal(t, "mongodb://custom:27017", cfg.MongoDB.URI)
	})
}
