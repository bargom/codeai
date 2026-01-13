package mongodb_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/database/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  mongodb.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  mongodb.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty URI",
			config: mongodb.Config{
				URI:      "",
				Database: "test",
			},
			wantErr: true,
			errMsg:  "URI is required",
		},
		{
			name: "empty database",
			config: mongodb.Config{
				URI:      "mongodb://localhost:27017",
				Database: "",
			},
			wantErr: true,
			errMsg:  "Database name is required",
		},
		{
			name: "min pool greater than max",
			config: mongodb.Config{
				URI:         "mongodb://localhost:27017",
				Database:    "test",
				MinPoolSize: 100,
				MaxPoolSize: 10,
			},
			wantErr: true,
			errMsg:  "MinPoolSize",
		},
		{
			name: "negative max retries",
			config: mongodb.Config{
				URI:         "mongodb://localhost:27017",
				Database:    "test",
				MinPoolSize: 5,
				MaxPoolSize: 100,
				MaxRetries:  -1,
			},
			wantErr: true,
			errMsg:  "MaxRetries cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := mongodb.DefaultConfig()

	assert.Equal(t, "mongodb://localhost:27017", cfg.URI)
	assert.Equal(t, "codeai", cfg.Database)
	assert.Equal(t, uint64(5), cfg.MinPoolSize)
	assert.Equal(t, uint64(100), cfg.MaxPoolSize)
	assert.Equal(t, 10*time.Second, cfg.ConnectTimeout)
	assert.Equal(t, 30*time.Second, cfg.SocketTimeout)
	assert.Equal(t, 5*time.Second, cfg.ServerSelectionTimeout)
	assert.True(t, cfg.RetryWrites)
	assert.True(t, cfg.RetryReads)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.RetryBackoff)
	assert.Equal(t, 5*time.Second, cfg.MaxRetryBackoff)
}

func TestClient_ConnectToTestContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := mongodb.SetupTestContainer(t)

	cfg := mongodb.DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = "test_db"

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := mongodb.New(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, client)

	defer func() {
		err := client.Close(context.Background())
		require.NoError(t, err)
	}()

	// Verify database is accessible
	db := client.Database()
	require.NotNil(t, db)

	// Verify collection access
	coll := client.Collection("test_collection")
	require.NotNil(t, coll)
}

func TestClient_CRUD_Operations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_crud", func(t *testing.T, client *mongodb.Client) {
		ctx := context.Background()
		coll := client.Collection("test_items")

		// Insert
		doc := bson.M{"name": "test", "value": 42}
		insertResult, err := coll.InsertOne(ctx, doc)
		require.NoError(t, err)
		assert.NotNil(t, insertResult.InsertedID)

		// Find
		var found bson.M
		err = coll.FindOne(ctx, bson.M{"name": "test"}).Decode(&found)
		require.NoError(t, err)
		assert.Equal(t, "test", found["name"])
		assert.Equal(t, int32(42), found["value"])

		// Update
		updateResult, err := coll.UpdateOne(ctx,
			bson.M{"name": "test"},
			bson.M{"$set": bson.M{"value": 100}},
		)
		require.NoError(t, err)
		assert.Equal(t, int64(1), updateResult.ModifiedCount)

		// Verify update
		err = coll.FindOne(ctx, bson.M{"name": "test"}).Decode(&found)
		require.NoError(t, err)
		assert.Equal(t, int32(100), found["value"])

		// Delete
		deleteResult, err := coll.DeleteOne(ctx, bson.M{"name": "test"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleteResult.DeletedCount)

		// Verify deletion
		err = coll.FindOne(ctx, bson.M{"name": "test"}).Decode(&found)
		assert.Error(t, err) // Should not find document
	})
}

func TestHealthCheck_Healthy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_health", func(t *testing.T, client *mongodb.Client) {
		logger := slog.Default()
		healthCheck := mongodb.NewHealthCheck(client, logger)

		result := healthCheck.Check(context.Background())

		assert.Equal(t, mongodb.HealthStatusHealthy, result.Status)
		assert.Equal(t, "connection is healthy", result.Message)
		assert.Greater(t, result.Latency, time.Duration(0))
		assert.NotEmpty(t, result.Details)
		assert.NotNil(t, result.Details["version"])
	})
}

func TestHealthCheck_IsHealthy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_is_healthy", func(t *testing.T, client *mongodb.Client) {
		healthCheck := mongodb.NewHealthCheck(client, slog.Default())

		assert.True(t, healthCheck.IsHealthy(context.Background()))
	})
}

func TestHealthCheck_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_ping", func(t *testing.T, client *mongodb.Client) {
		healthCheck := mongodb.NewHealthCheck(client, slog.Default())

		err := healthCheck.Ping(context.Background())
		require.NoError(t, err)
	})
}

func TestHealthCheck_Readiness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_readiness", func(t *testing.T, client *mongodb.Client) {
		healthCheck := mongodb.NewHealthCheck(client, slog.Default())

		err := healthCheck.CheckReadiness(context.Background())
		require.NoError(t, err)
	})
}

func TestHealthCheck_UnhealthyWhenClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := mongodb.SetupTestContainer(t)

	cfg := mongodb.DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = "test_closed"

	client, err := mongodb.New(context.Background(), cfg, slog.Default())
	require.NoError(t, err)

	// Close the client
	err = client.Close(context.Background())
	require.NoError(t, err)

	// Health check should fail
	healthCheck := mongodb.NewHealthCheck(client, slog.Default())
	result := healthCheck.Check(context.Background())

	assert.Equal(t, mongodb.HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "closed")
}

func TestClient_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := mongodb.SetupTestContainer(t)

	cfg := mongodb.DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = "test_close"

	client, err := mongodb.New(context.Background(), cfg, slog.Default())
	require.NoError(t, err)

	// Close should succeed
	err = client.Close(context.Background())
	require.NoError(t, err)

	// IsClosed should return true
	assert.True(t, client.IsClosed())

	// Double close should not error
	err = client.Close(context.Background())
	require.NoError(t, err)
}

func TestClient_ConnectionFailure(t *testing.T) {
	cfg := mongodb.DefaultConfig()
	cfg.URI = "mongodb://nonexistent:27017"
	cfg.Database = "test"
	cfg.ConnectTimeout = 500 * time.Millisecond
	cfg.ServerSelectionTimeout = 500 * time.Millisecond
	cfg.MaxRetries = 1
	cfg.RetryBackoff = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mongodb.New(ctx, cfg, slog.Default())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestClient_Collection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_collection", func(t *testing.T, client *mongodb.Client) {
		coll := client.Collection("my_collection")
		require.NotNil(t, coll)
		assert.Equal(t, "my_collection", coll.Name())
	})
}

func TestClient_Client(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_client_access", func(t *testing.T, client *mongodb.Client) {
		mongoClient := client.Client()
		require.NotNil(t, mongoClient)

		// Should be able to list databases
		ctx := context.Background()
		result, err := mongoClient.ListDatabaseNames(ctx, bson.D{})
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

func TestHealthCheck_SetTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mongodb.RunWithTestContainer(t, "test_timeout", func(t *testing.T, client *mongodb.Client) {
		healthCheck := mongodb.NewHealthCheck(client, slog.Default())

		// Set a very short timeout and still expect it to work
		healthCheck.SetTimeout(10 * time.Second)

		result := healthCheck.Check(context.Background())
		assert.Equal(t, mongodb.HealthStatusHealthy, result.Status)
	})
}

func TestSetupTestContainerWithConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := mongodb.TestContainerConfig{
		Image:    "mongo:7.0",
		Database: "custom_test_db",
	}

	tc := mongodb.SetupTestContainerWithConfig(t, cfg)
	require.NotEmpty(t, tc.URI)

	client := tc.NewTestClient(t, cfg.Database)
	require.NotNil(t, client)

	// Verify we can connect and operate
	ctx := context.Background()
	coll := client.Collection("test")
	_, err := coll.InsertOne(ctx, bson.M{"test": true})
	require.NoError(t, err)
}
