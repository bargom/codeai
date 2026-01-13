//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/database/mongodb"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBTestSuite holds resources for MongoDB integration tests.
type MongoDBTestSuite struct {
	Container *mongodb.TestContainer
	Client    *mongodb.Client
	Server    *httptest.Server
	Logger    *slog.Logger
}

// SetupMongoDBTestSuite creates a new MongoDB test suite with testcontainers.
func SetupMongoDBTestSuite(t *testing.T) *MongoDBTestSuite {
	t.Helper()

	suite := &MongoDBTestSuite{
		Logger: slog.Default(),
	}

	// Setup MongoDB container
	container, client := mongodb.SetupTestContainerWithClient(t, "test_mongodb_only")
	suite.Container = container
	suite.Client = client

	// Setup a minimal HTTP server for testing
	router := chi.NewRouter()
	setupMongoDBRoutes(router, client, suite.Logger)
	suite.Server = httptest.NewServer(router)

	t.Cleanup(func() {
		suite.Server.Close()
	})

	return suite
}

// setupMongoDBRoutes sets up HTTP routes for MongoDB CRUD operations.
func setupMongoDBRoutes(r chi.Router, client *mongodb.Client, logger *slog.Logger) {
	repo := mongodb.NewRepository(client, "tasks", logger)

	r.Route("/api/tasks", func(r chi.Router) {
		// List all tasks
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			docs, err := repo.Find(ctx, mongodb.Filter{}, nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(docs)
		})

		// Create a task
		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			var doc mongodb.Document
			if err := json.NewDecoder(req.Body).Decode(&doc); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			id, err := repo.InsertOne(ctx, doc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": id})
		})

		// Get a task by ID
		r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			id := chi.URLParam(req, "id")
			doc, err := repo.FindByID(ctx, id)
			if err != nil {
				if err == mongodb.ErrNotFound {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(doc)
		})

		// Update a task
		r.Put("/{id}", func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			id := chi.URLParam(req, "id")
			var update map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Build update filter by ID
			oid, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}
			count, err := repo.UpdateOne(ctx, mongodb.Filter{"_id": oid}, mongodb.Update{"$set": bson.M(update)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if count == 0 {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]int64{"modified": count})
		})

		// Delete a task
		r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			id := chi.URLParam(req, "id")
			// Build delete filter by ID
			oid, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}
			count, err := repo.DeleteOne(ctx, mongodb.Filter{"_id": oid})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if count == 0 {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

// TestMongoDBOnlyCRUDLifecycle tests the full CRUD lifecycle with MongoDB.
func TestMongoDBOnlyCRUDLifecycle(t *testing.T) {
	suite := SetupMongoDBTestSuite(t)
	client := &http.Client{Timeout: 10 * time.Second}

	ctx := context.Background()

	t.Run("create task", func(t *testing.T) {
		task := map[string]interface{}{
			"title":       "Test Task",
			"description": "This is a test task",
			"status":      "pending",
			"priority":    1,
		}
		body, _ := json.Marshal(task)

		req, err := http.NewRequestWithContext(ctx, "POST", suite.Server.URL+"/api/tasks", strings.NewReader(string(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]string
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("list tasks", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", suite.Server.URL+"/api/tasks", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tasks []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&tasks)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})
}

// TestMongoDBOnlyRepositoryOperations tests direct repository operations.
func TestMongoDBOnlyRepositoryOperations(t *testing.T) {
	container, client := mongodb.SetupTestContainerWithClient(t, "test_repo_ops")
	_ = container // container cleanup is handled by SetupTestContainerWithClient

	logger := slog.Default()
	repo := mongodb.NewRepository(client, "products", logger)
	ctx := context.Background()

	t.Run("insert and find documents", func(t *testing.T) {
		// Insert a document
		doc := mongodb.Document{
			"name":     "Test Product",
			"sku":      "TEST-001",
			"price":    99.99,
			"quantity": 10,
			"active":   true,
		}

		id, err := repo.InsertOne(ctx, doc)
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		// Find the document by ID
		found, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "Test Product", found["name"])
		assert.Equal(t, "TEST-001", found["sku"])
	})

	t.Run("update document", func(t *testing.T) {
		// Insert
		doc := mongodb.Document{
			"name":  "Update Test",
			"sku":   "UPD-001",
			"price": 50.00,
		}
		id, err := repo.InsertOne(ctx, doc)
		require.NoError(t, err)

		// Update using filter
		oid, err := primitive.ObjectIDFromHex(id)
		require.NoError(t, err)
		count, err := repo.UpdateOne(ctx, mongodb.Filter{"_id": oid}, mongodb.Update{
			"$set": bson.M{
				"price": 75.00,
				"name":  "Updated Product",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify
		updated, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "Updated Product", updated["name"])
	})

	t.Run("delete document", func(t *testing.T) {
		// Insert
		doc := mongodb.Document{
			"name": "Delete Test",
			"sku":  "DEL-001",
		}
		id, err := repo.InsertOne(ctx, doc)
		require.NoError(t, err)

		// Delete using filter
		oid, err := primitive.ObjectIDFromHex(id)
		require.NoError(t, err)
		count, err := repo.DeleteOne(ctx, mongodb.Filter{"_id": oid})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify deleted
		_, err = repo.FindByID(ctx, id)
		assert.ErrorIs(t, err, mongodb.ErrNotFound)
	})

	t.Run("find with filter", func(t *testing.T) {
		// Insert multiple documents
		docs := []mongodb.Document{
			{"name": "Product A", "category": "electronics", "price": 100.0},
			{"name": "Product B", "category": "electronics", "price": 200.0},
			{"name": "Product C", "category": "clothing", "price": 50.0},
		}

		for _, doc := range docs {
			_, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)
		}

		// Find electronics
		filter := mongodb.Filter{"category": "electronics"}
		results, err := repo.Find(ctx, filter, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)

		// Verify all results are electronics
		for _, r := range results {
			assert.Equal(t, "electronics", r["category"])
		}
	})

	t.Run("count documents", func(t *testing.T) {
		// Create a fresh collection for counting
		countRepo := mongodb.NewRepository(client, "count_test", logger)

		// Insert some documents
		for i := 0; i < 5; i++ {
			_, err := countRepo.InsertOne(ctx, mongodb.Document{
				"index": i,
				"type":  "test",
			})
			require.NoError(t, err)
		}

		// Count all
		count, err := countRepo.Count(ctx, mongodb.Filter{})
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		// Count with filter
		count, err = countRepo.Count(ctx, mongodb.Filter{"index": 0})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

// TestMongoDBOnlyIndexOperations tests index creation and management.
func TestMongoDBOnlyIndexOperations(t *testing.T) {
	_, client := mongodb.SetupTestContainerWithClient(t, "test_indexes")
	logger := slog.Default()
	repo := mongodb.NewRepository(client, "indexed_collection", logger)
	ctx := context.Background()

	t.Run("create unique index", func(t *testing.T) {
		// Ensure unique index on email
		err := repo.EnsureIndex(ctx, []string{"email"}, true)
		require.NoError(t, err)

		// Insert first document
		_, err = repo.InsertOne(ctx, mongodb.Document{
			"email": "test@example.com",
			"name":  "Test User",
		})
		require.NoError(t, err)

		// Try to insert duplicate - should fail
		_, err = repo.InsertOne(ctx, mongodb.Document{
			"email": "test@example.com",
			"name":  "Duplicate User",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key")
	})

	t.Run("create compound index", func(t *testing.T) {
		compoundRepo := mongodb.NewRepository(client, "compound_indexed", logger)

		err := compoundRepo.EnsureIndex(ctx, []string{"category", "created_at"}, false)
		require.NoError(t, err)
	})
}

// TestMongoDBOnlyPagination tests pagination functionality.
func TestMongoDBOnlyPagination(t *testing.T) {
	_, client := mongodb.SetupTestContainerWithClient(t, "test_pagination")
	logger := slog.Default()
	repo := mongodb.NewRepository(client, "paginated_collection", logger)
	ctx := context.Background()

	// Insert 25 documents
	for i := 0; i < 25; i++ {
		_, err := repo.InsertOne(ctx, mongodb.Document{
			"index": i,
			"name":  "Item " + string(rune('A'+i%26)),
		})
		require.NoError(t, err)
	}

	t.Run("offset pagination", func(t *testing.T) {
		// First page
		opts := &mongodb.FindOptions{
			Limit: 10,
			Skip:  0,
		}
		page1, err := repo.Find(ctx, mongodb.Filter{}, opts)
		require.NoError(t, err)
		assert.Len(t, page1, 10)

		// Second page
		opts.Skip = 10
		page2, err := repo.Find(ctx, mongodb.Filter{}, opts)
		require.NoError(t, err)
		assert.Len(t, page2, 10)

		// Third page (partial)
		opts.Skip = 20
		page3, err := repo.Find(ctx, mongodb.Filter{}, opts)
		require.NoError(t, err)
		assert.Len(t, page3, 5)

		// Verify pages don't overlap
		page1IDs := make(map[interface{}]bool)
		for _, doc := range page1 {
			page1IDs[doc["_id"]] = true
		}
		for _, doc := range page2 {
			assert.False(t, page1IDs[doc["_id"]], "page2 should not contain page1 documents")
		}
	})
}

// TestMongoDBOnlyConnectionFailures tests error handling for connection issues.
func TestMongoDBOnlyConnectionFailures(t *testing.T) {
	t.Run("invalid URI handling", func(t *testing.T) {
		cfg := mongodb.DefaultConfig()
		cfg.URI = "mongodb://invalid:27017"
		cfg.ConnectTimeout = 1 * time.Second

		logger := slog.Default()
		_, err := mongodb.New(context.Background(), cfg, logger)
		assert.Error(t, err)
	})

	t.Run("operations on closed client", func(t *testing.T) {
		_, client := mongodb.SetupTestContainerWithClient(t, "test_closed")
		logger := slog.Default()
		repo := mongodb.NewRepository(client, "test", logger)
		ctx := context.Background()

		// Close the client
		err := client.Close(ctx)
		require.NoError(t, err)

		// Attempt operations - should fail gracefully
		_, err = repo.Find(ctx, mongodb.Filter{}, nil)
		assert.Error(t, err)
	})
}

// TestMongoDBOnlyHealthCheck tests the health check functionality.
func TestMongoDBOnlyHealthCheck(t *testing.T) {
	_, client := mongodb.SetupTestContainerWithClient(t, "test_health")
	ctx := context.Background()
	logger := slog.Default()

	t.Run("health check passes when connected", func(t *testing.T) {
		healthCheck := mongodb.NewHealthCheck(client, logger)
		result := healthCheck.Check(ctx)
		assert.Equal(t, mongodb.HealthStatusHealthy, result.Status)
	})

	t.Run("ping succeeds when connected", func(t *testing.T) {
		healthCheck := mongodb.NewHealthCheck(client, logger)
		err := healthCheck.Ping(ctx)
		require.NoError(t, err)
	})

	t.Run("is healthy returns true when connected", func(t *testing.T) {
		healthCheck := mongodb.NewHealthCheck(client, logger)
		assert.True(t, healthCheck.IsHealthy(ctx))
	})
}
