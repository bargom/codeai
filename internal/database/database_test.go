package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Empty(t, cfg.Database)
	assert.Empty(t, cfg.User)
	assert.Empty(t, cfg.Password)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Host:     "dbhost",
		Port:     5433,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "require",
	}

	assert.Equal(t, "dbhost", cfg.Host)
	assert.Equal(t, 5433, cfg.Port)
	assert.Equal(t, "testdb", cfg.Database)
	assert.Equal(t, "testuser", cfg.User)
	assert.Equal(t, "testpass", cfg.Password)
	assert.Equal(t, "require", cfg.SSLMode)
}
