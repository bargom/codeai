package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// TestContainer holds a MongoDB test container instance.
type TestContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
	client    *Client
}

// SetupTestContainer creates a MongoDB container for testing.
// It returns a TestContainer with the connection URI and a cleanup function.
func SetupTestContainer(t *testing.T) *TestContainer {
	t.Helper()

	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7.0")
	if err != nil {
		t.Fatalf("failed to start MongoDB container: %v", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	tc := &TestContainer{
		Container: container,
		URI:       uri,
	}

	t.Cleanup(func() {
		tc.Cleanup(t)
	})

	return tc
}

// SetupTestContainerWithClient creates a MongoDB container and connects a client.
func SetupTestContainerWithClient(t *testing.T, database string) (*TestContainer, *Client) {
	t.Helper()

	tc := SetupTestContainer(t)

	cfg := DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = database
	cfg.MaxRetries = 3

	logger := slog.Default()
	client, err := New(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("failed to create MongoDB client: %v", err)
	}

	tc.client = client

	t.Cleanup(func() {
		if client != nil {
			_ = client.Close(context.Background())
		}
	})

	return tc, client
}

// Cleanup terminates the test container.
func (tc *TestContainer) Cleanup(t *testing.T) {
	t.Helper()

	if tc.client != nil {
		if err := tc.client.Close(context.Background()); err != nil {
			t.Logf("warning: failed to close client: %v", err)
		}
		tc.client = nil
	}

	if tc.Container != nil {
		if err := tc.Container.Terminate(context.Background()); err != nil {
			t.Logf("warning: failed to terminate container: %v", err)
		}
	}
}

// NewTestClient creates a new client connected to the test container.
func (tc *TestContainer) NewTestClient(t *testing.T, database string) *Client {
	t.Helper()

	cfg := DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = database

	logger := slog.Default()
	client, err := New(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("failed to create MongoDB client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close(context.Background())
	})

	return client
}

// TestContainerConfig holds configuration for the test container.
type TestContainerConfig struct {
	// Image is the MongoDB image to use (default: mongo:7.0)
	Image string

	// Database is the database name to use
	Database string

	// ReplicaSet enables replica set mode (required for transactions)
	ReplicaSet bool
}

// DefaultTestContainerConfig returns default test container configuration.
func DefaultTestContainerConfig() TestContainerConfig {
	return TestContainerConfig{
		Image:    "mongo:7.0",
		Database: "test_codeai",
	}
}

// SetupTestContainerWithConfig creates a MongoDB container with custom configuration.
func SetupTestContainerWithConfig(t *testing.T, cfg TestContainerConfig) *TestContainer {
	t.Helper()

	ctx := context.Background()

	var opts []testcontainers.ContainerCustomizer

	if cfg.ReplicaSet {
		opts = append(opts, mongodb.WithReplicaSet("rs0"))
	}

	container, err := mongodb.Run(ctx, cfg.Image, opts...)
	if err != nil {
		t.Fatalf("failed to start MongoDB container: %v", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	tc := &TestContainer{
		Container: container,
		URI:       uri,
	}

	t.Cleanup(func() {
		tc.Cleanup(t)
	})

	return tc
}

// RunWithTestContainer is a helper for running tests with a MongoDB container.
func RunWithTestContainer(t *testing.T, database string, fn func(t *testing.T, client *Client)) {
	t.Helper()

	_, client := SetupTestContainerWithClient(t, database)
	fn(t, client)
}

// MustConnectionString returns the connection string or panics.
// This is useful for integration tests that need the URI.
func MustConnectionString(ctx context.Context, container *mongodb.MongoDBContainer) string {
	uri, err := container.ConnectionString(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get connection string: %v", err))
	}
	return uri
}
