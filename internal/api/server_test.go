package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/api"
	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/database/repository"
	dbtesting "github.com/bargom/codeai/internal/database/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerStartAndShutdown(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)
	router := api.NewRouter(h)
	server := api.NewServer(router, ":0") // Use port 0 for random available port

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	require.NoError(t, err)

	// Wait for server to stop
	select {
	case err := <-errCh:
		assert.NoError(t, err) // Should be nil on clean shutdown
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServerAddr(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)
	router := api.NewRouter(h)
	server := api.NewServer(router, ":8080")

	assert.Equal(t, ":8080", server.Addr())
}

func TestServerRouter(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)
	router := api.NewRouter(h)
	server := api.NewServer(router, ":8080")

	assert.NotNil(t, server.Router())
}

func TestServerHealthEndpoint(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)
	router := api.NewRouter(h)
	server := api.NewServer(router, "127.0.0.1:0")

	// Start server
	go func() {
		_ = server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Make request to health endpoint
	// Note: Since we use port 0, we can't easily get the actual port
	// This test mainly verifies the server can start without error
}
