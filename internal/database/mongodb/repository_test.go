package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/pagination"
	"github.com/bargom/codeai/internal/query"
)

func TestRepository_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_crud", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "users", nil)
		ctx := context.Background()

		t.Run("InsertOne and FindByID", func(t *testing.T) {
			doc := Document{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			}

			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)
			assert.NotEmpty(t, id)

			// Find the document
			found, err := repo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, "John Doe", found["name"])
			assert.Equal(t, "john@example.com", found["email"])
			assert.NotNil(t, found["createdAt"])
			assert.NotNil(t, found["updatedAt"])
		})

		t.Run("InsertOne with custom ID", func(t *testing.T) {
			customID := "custom-id-123"
			doc := Document{
				"_id":   customID,
				"name":  "Jane Doe",
				"email": "jane@example.com",
			}

			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)
			assert.Equal(t, customID, id)

			found, err := repo.FindByID(ctx, customID)
			require.NoError(t, err)
			assert.Equal(t, "Jane Doe", found["name"])
		})

		t.Run("FindOne with filter", func(t *testing.T) {
			filter := Filter{"name": "John Doe"}
			found, err := repo.FindOne(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, "john@example.com", found["email"])
		})

		t.Run("FindOne not found", func(t *testing.T) {
			filter := Filter{"name": "Nonexistent"}
			_, err := repo.FindOne(ctx, filter)
			require.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateOne", func(t *testing.T) {
			filter := Filter{"name": "John Doe"}
			update := Update{
				"$set": bson.M{
					"age": 31,
				},
			}

			count, err := repo.UpdateOne(ctx, filter, update)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)

			// Verify update
			found, err := repo.FindOne(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, int32(31), found["age"])
		})

		t.Run("DeleteOne", func(t *testing.T) {
			// Insert a document to delete
			doc := Document{
				"name":  "ToDelete",
				"email": "delete@example.com",
			}
			_, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)

			// Delete it
			filter := Filter{"name": "ToDelete"}
			count, err := repo.DeleteOne(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)

			// Verify deletion
			_, err = repo.FindOne(ctx, filter)
			require.ErrorIs(t, err, ErrNotFound)
		})
	})
}

func TestRepository_BulkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_bulk", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "products", nil)
		ctx := context.Background()

		t.Run("InsertMany", func(t *testing.T) {
			docs := []Document{
				{"name": "Product 1", "price": 10.99, "category": "electronics"},
				{"name": "Product 2", "price": 20.99, "category": "electronics"},
				{"name": "Product 3", "price": 5.99, "category": "books"},
			}

			ids, err := repo.InsertMany(ctx, docs)
			require.NoError(t, err)
			assert.Len(t, ids, 3)
		})

		t.Run("Find with options", func(t *testing.T) {
			opts := &FindOptions{
				Sort:  bson.D{{Key: "price", Value: 1}},
				Limit: 2,
			}

			docs, err := repo.Find(ctx, Filter{}, opts)
			require.NoError(t, err)
			assert.Len(t, docs, 2)
			// Should be sorted by price ascending
			assert.Equal(t, 5.99, docs[0]["price"])
		})

		t.Run("Find with filter", func(t *testing.T) {
			filter := Filter{"category": "electronics"}
			docs, err := repo.Find(ctx, filter, nil)
			require.NoError(t, err)
			assert.Len(t, docs, 2)
		})

		t.Run("UpdateMany", func(t *testing.T) {
			filter := Filter{"category": "electronics"}
			update := Update{
				"$set": bson.M{"onSale": true},
			}

			count, err := repo.UpdateMany(ctx, filter, update)
			require.NoError(t, err)
			assert.Equal(t, int64(2), count)

			// Verify updates
			docs, err := repo.Find(ctx, filter, nil)
			require.NoError(t, err)
			for _, doc := range docs {
				assert.Equal(t, true, doc["onSale"])
			}
		})

		t.Run("DeleteMany", func(t *testing.T) {
			filter := Filter{"category": "books"}
			count, err := repo.DeleteMany(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)

			// Verify deletion
			docs, err := repo.Find(ctx, filter, nil)
			require.NoError(t, err)
			assert.Len(t, docs, 0)
		})

		t.Run("InsertMany empty slice", func(t *testing.T) {
			ids, err := repo.InsertMany(ctx, []Document{})
			require.NoError(t, err)
			assert.Len(t, ids, 0)
		})
	})
}

func TestRepository_Transactions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Transactions require replica set - skip if replica set initialization fails
	// as this is environment-dependent
	cfg := DefaultTestContainerConfig()
	cfg.ReplicaSet = true
	tc := SetupTestContainerWithConfig(t, cfg)

	clientCfg := DefaultConfig()
	clientCfg.URI = tc.URI
	clientCfg.Database = "test_transactions"
	// Increase timeouts for replica set initialization
	clientCfg.MaxRetries = 10
	clientCfg.ConnectTimeout = 30 * time.Second
	clientCfg.ServerSelectionTimeout = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := New(ctx, clientCfg, nil)
	if err != nil {
		t.Skipf("skipping transaction test - replica set not ready: %v", err)
	}
	defer client.Close(context.Background())

	repo := NewRepository(client, "accounts", nil)
	cancel() // Cancel the timeout context, use fresh context for tests
	ctx = context.Background()

	t.Run("Transaction commit", func(t *testing.T) {
		// Setup initial data
		_, err := repo.InsertOne(ctx, Document{"name": "Account A", "balance": 100})
		require.NoError(t, err)
		_, err = repo.InsertOne(ctx, Document{"name": "Account B", "balance": 50})
		require.NoError(t, err)

		// Execute transaction
		err = repo.WithTransaction(ctx, func(sessCtx mongo.SessionContext) error {
			// Transfer 30 from A to B
			_, err := repo.collection.UpdateOne(sessCtx,
				bson.M{"name": "Account A"},
				bson.M{"$inc": bson.M{"balance": -30}},
			)
			if err != nil {
				return err
			}

			_, err = repo.collection.UpdateOne(sessCtx,
				bson.M{"name": "Account B"},
				bson.M{"$inc": bson.M{"balance": 30}},
			)
			return err
		})
		require.NoError(t, err)

		// Verify final balances
		accountA, err := repo.FindOne(ctx, Filter{"name": "Account A"})
		require.NoError(t, err)
		assert.Equal(t, int32(70), accountA["balance"])

		accountB, err := repo.FindOne(ctx, Filter{"name": "Account B"})
		require.NoError(t, err)
		assert.Equal(t, int32(80), accountB["balance"])
	})

	t.Run("Transaction rollback", func(t *testing.T) {
		// Get current balance
		accountA, err := repo.FindOne(ctx, Filter{"name": "Account A"})
		require.NoError(t, err)
		initialBalance := accountA["balance"]

		// Execute transaction that fails
		err = repo.WithTransaction(ctx, func(sessCtx mongo.SessionContext) error {
			_, err := repo.collection.UpdateOne(sessCtx,
				bson.M{"name": "Account A"},
				bson.M{"$inc": bson.M{"balance": -50}},
			)
			if err != nil {
				return err
			}

			// Simulate error
			return context.Canceled
		})
		require.Error(t, err)

		// Verify balance unchanged (rolled back)
		accountA, err = repo.FindOne(ctx, Filter{"name": "Account A"})
		require.NoError(t, err)
		assert.Equal(t, initialBalance, accountA["balance"])
	})
}

func TestRepository_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_pagination", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "items", nil)
		ctx := context.Background()

		// Insert test data
		docs := make([]Document, 25)
		for i := 0; i < 25; i++ {
			docs[i] = Document{
				"name":  "Item " + string(rune('A'+i)),
				"index": i,
			}
		}
		_, err := repo.InsertMany(ctx, docs)
		require.NoError(t, err)

		t.Run("Offset pagination - first page", func(t *testing.T) {
			page := pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  1,
				Limit: 10,
			}

			result, err := repo.FindWithPagination(ctx, Filter{}, page, "index", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result.Data, 10)
			assert.True(t, result.Pagination.HasNext)
			assert.False(t, result.Pagination.HasPrevious)
			assert.Equal(t, int64(25), *result.Pagination.Total)
		})

		t.Run("Offset pagination - middle page", func(t *testing.T) {
			page := pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  2,
				Limit: 10,
			}

			result, err := repo.FindWithPagination(ctx, Filter{}, page, "index", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result.Data, 10)
			assert.True(t, result.Pagination.HasNext)
			assert.True(t, result.Pagination.HasPrevious)
		})

		t.Run("Offset pagination - last page", func(t *testing.T) {
			page := pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  3,
				Limit: 10,
			}

			result, err := repo.FindWithPagination(ctx, Filter{}, page, "index", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result.Data, 5)
			assert.False(t, result.Pagination.HasNext)
			assert.True(t, result.Pagination.HasPrevious)
		})

		t.Run("Cursor pagination - forward", func(t *testing.T) {
			// First page
			page := pagination.PageRequest{
				Type:  pagination.PaginationCursor,
				Limit: 10,
			}

			result, err := repo.FindWithPagination(ctx, Filter{}, page, "_id", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result.Data, 10)
			assert.True(t, result.Pagination.HasNext)
			assert.NotEmpty(t, result.Pagination.NextCursor)

			// Second page using cursor
			page.After = result.Pagination.NextCursor
			result2, err := repo.FindWithPagination(ctx, Filter{}, page, "_id", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result2.Data, 10)
			assert.True(t, result2.Pagination.HasNext)
			assert.True(t, result2.Pagination.HasPrevious)
		})

		t.Run("Cursor pagination - backward", func(t *testing.T) {
			// Get to page 2 first
			page := pagination.PageRequest{
				Type:  pagination.PaginationCursor,
				Limit: 10,
			}
			result, err := repo.FindWithPagination(ctx, Filter{}, page, "_id", query.OrderAsc)
			require.NoError(t, err)

			page.After = result.Pagination.NextCursor
			result2, err := repo.FindWithPagination(ctx, Filter{}, page, "_id", query.OrderAsc)
			require.NoError(t, err)

			// Go backward
			page.After = ""
			page.Before = result2.Pagination.PrevCursor
			result3, err := repo.FindWithPagination(ctx, Filter{}, page, "_id", query.OrderAsc)
			require.NoError(t, err)
			assert.Len(t, result3.Data, 10)
		})
	})
}

func TestRepository_QueryExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_query", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "orders", nil)
		ctx := context.Background()

		// Insert test data
		docs := []Document{
			{"customer": "Alice", "amount": 100.0, "status": "pending", "tags": []string{"urgent", "new"}},
			{"customer": "Bob", "amount": 250.0, "status": "completed", "tags": []string{"regular"}},
			{"customer": "Charlie", "amount": 75.0, "status": "pending", "tags": []string{"new"}},
			{"customer": "Alice", "amount": 300.0, "status": "completed", "tags": []string{"vip"}},
		}
		_, err := repo.InsertMany(ctx, docs)
		require.NoError(t, err)

		t.Run("Query with equals", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "status", Operator: query.OpEquals, Value: "pending"},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 2)
		})

		t.Run("Query with greater than", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "amount", Operator: query.OpGreaterThan, Value: 100.0},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 2)
		})

		t.Run("Query with AND conditions", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Operator: query.LogicalAnd,
					Conditions: []query.Condition{
						{Field: "customer", Operator: query.OpEquals, Value: "Alice"},
						{Field: "status", Operator: query.OpEquals, Value: "completed"},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 1)
			assert.Equal(t, 300.0, results[0]["amount"])
		})

		t.Run("Query with OR conditions", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Operator: query.LogicalOr,
					Conditions: []query.Condition{
						{Field: "customer", Operator: query.OpEquals, Value: "Alice"},
						{Field: "customer", Operator: query.OpEquals, Value: "Bob"},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 3)
		})

		t.Run("Query with IN operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "status", Operator: query.OpIn, Value: []string{"pending", "cancelled"}},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 2)
		})

		t.Run("Query with order and limit", func(t *testing.T) {
			limit := 2
			q := &query.Query{
				Type: query.QuerySelect,
				OrderBy: []query.OrderClause{
					{Field: "amount", Direction: query.OrderDesc},
				},
				Limit: &limit,
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 2)
			assert.Equal(t, 300.0, results[0]["amount"])
			assert.Equal(t, 250.0, results[1]["amount"])
		})

		t.Run("Query with field projection", func(t *testing.T) {
			q := &query.Query{
				Type:   query.QuerySelect,
				Fields: []string{"customer", "amount"},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Greater(t, len(results), 0)
			// MongoDB always includes _id unless explicitly excluded
			assert.NotNil(t, results[0]["_id"])
			assert.NotNil(t, results[0]["customer"])
			assert.NotNil(t, results[0]["amount"])
			assert.Nil(t, results[0]["status"])
		})
	})
}

func TestRepository_NestedDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_nested", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "users_nested", nil)
		ctx := context.Background()

		t.Run("Insert and query nested documents", func(t *testing.T) {
			doc := Document{
				"name": "John",
				"address": Document{
					"street": "123 Main St",
					"city":   "New York",
					"zip":    "10001",
				},
				"contacts": []Document{
					{"type": "email", "value": "john@example.com"},
					{"type": "phone", "value": "555-1234"},
				},
			}

			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)
			assert.NotEmpty(t, id)

			// Query nested field with dot notation
			filter := Filter{"address.city": "New York"}
			found, err := repo.FindOne(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, "John", found["name"])

			// Query array element
			filter = Filter{"contacts.type": "email"}
			found, err = repo.FindOne(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, "John", found["name"])
		})

		t.Run("Update nested field", func(t *testing.T) {
			filter := Filter{"name": "John"}
			update := Update{
				"$set": bson.M{"address.zip": "10002"},
			}

			count, err := repo.UpdateOne(ctx, filter, update)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)

			// Verify update
			found, err := repo.FindOne(ctx, filter)
			require.NoError(t, err)
			// Handle different possible types for nested documents
			var zip interface{}
			switch addr := found["address"].(type) {
			case primitive.M:
				zip = addr["zip"]
			case Document:
				zip = addr["zip"]
			case map[string]interface{}:
				zip = addr["zip"]
			}
			assert.Equal(t, "10002", zip)
		})
	})
}

func TestRepository_Indexes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_indexes", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "indexed_collection", nil)
		ctx := context.Background()

		t.Run("EnsureIndexes from DSL declarations", func(t *testing.T) {
			indexes := []*ast.MongoIndexDecl{
				{Fields: []string{"email"}, Unique: true},
				{Fields: []string{"createdAt", "status"}},
				{Fields: []string{"name"}, IndexKind: "text"},
			}

			err := repo.EnsureIndexes(ctx, indexes)
			require.NoError(t, err)

			// Verify by listing indexes
			cursor, err := repo.collection.Indexes().List(ctx)
			require.NoError(t, err)

			var indexDocs []bson.M
			err = cursor.All(ctx, &indexDocs)
			require.NoError(t, err)

			// Should have 4 indexes: _id (default) + 3 created
			assert.GreaterOrEqual(t, len(indexDocs), 4)
		})

		t.Run("EnsureIndex single", func(t *testing.T) {
			err := repo.EnsureIndex(ctx, []string{"category", "subcategory"}, false)
			require.NoError(t, err)
		})

		t.Run("Unique index prevents duplicates", func(t *testing.T) {
			doc1 := Document{"email": "unique@test.com", "name": "User 1"}
			_, err := repo.InsertOne(ctx, doc1)
			require.NoError(t, err)

			doc2 := Document{"email": "unique@test.com", "name": "User 2"}
			_, err = repo.InsertOne(ctx, doc2)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrDuplicateKey)
		})

		t.Run("Empty indexes slice", func(t *testing.T) {
			err := repo.EnsureIndexes(ctx, []*ast.MongoIndexDecl{})
			require.NoError(t, err)

			err = repo.EnsureIndexes(ctx, nil)
			require.NoError(t, err)
		})
	})
}

func TestRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_count", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "counted", nil)
		ctx := context.Background()

		// Insert test data
		docs := []Document{
			{"category": "A"},
			{"category": "A"},
			{"category": "B"},
		}
		_, err := repo.InsertMany(ctx, docs)
		require.NoError(t, err)

		t.Run("Count all", func(t *testing.T) {
			count, err := repo.Count(ctx, Filter{})
			require.NoError(t, err)
			assert.Equal(t, int64(3), count)
		})

		t.Run("Count with filter", func(t *testing.T) {
			count, err := repo.Count(ctx, Filter{"category": "A"})
			require.NoError(t, err)
			assert.Equal(t, int64(2), count)
		})

		t.Run("Exists true", func(t *testing.T) {
			exists, err := repo.Exists(ctx, Filter{"category": "B"})
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("Exists false", func(t *testing.T) {
			exists, err := repo.Exists(ctx, Filter{"category": "C"})
			require.NoError(t, err)
			assert.False(t, exists)
		})
	})
}

func TestRepository_FilterTranslation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_filters", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "filter_test", nil)
		ctx := context.Background()

		// Insert test data
		docs := []Document{
			{"name": "Apple", "price": 1.50, "category": "fruit", "inStock": true},
			{"name": "Banana", "price": 0.75, "category": "fruit", "inStock": true},
			{"name": "Carrot", "price": 0.50, "category": "vegetable", "inStock": false},
			{"name": "Date", "price": 2.00, "category": "fruit", "inStock": nil},
		}
		_, err := repo.InsertMany(ctx, docs)
		require.NoError(t, err)

		t.Run("Contains operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "name", Operator: query.OpContains, Value: "an"},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 1) // Banana
		})

		t.Run("StartsWith operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "name", Operator: query.OpStartsWith, Value: "A"},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 1) // Apple
		})

		t.Run("EndsWith operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "name", Operator: query.OpEndsWith, Value: "e"},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 2) // Apple, Date
		})

		t.Run("IsNull operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "inStock", Operator: query.OpIsNull, Value: nil},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 1) // Date
		})

		t.Run("IsNotNull operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "inStock", Operator: query.OpIsNotNull, Value: nil},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 3) // Apple, Banana, Carrot
		})

		t.Run("Between operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "price", Operator: query.OpBetween, Value: query.BetweenValue{Low: 0.5, High: 1.5}},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 3) // Banana, Carrot, Apple
		})

		t.Run("NotIn operator", func(t *testing.T) {
			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "category", Operator: query.OpNotIn, Value: []string{"vegetable"}},
					},
				},
			}
			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			assert.Len(t, results, 3) // All fruits
		})
	})
}

func TestRepository_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_errors", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "error_test", nil)
		ctx := context.Background()

		t.Run("FindByID with invalid ObjectID format", func(t *testing.T) {
			// This should try both string and ObjectID lookup
			_, err := repo.FindByID(ctx, "invalid-id")
			require.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("FindByID with valid ObjectID that doesn't exist", func(t *testing.T) {
			oid := primitive.NewObjectID().Hex()
			_, err := repo.FindByID(ctx, oid)
			require.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("Operations after client close", func(t *testing.T) {
			// Create a new client and close it
			clientCfg := DefaultConfig()
			clientCfg.URI = client.config.URI
			clientCfg.Database = "test_close"

			tempClient, err := New(ctx, clientCfg, nil)
			require.NoError(t, err)

			tempRepo := NewRepository(tempClient, "temp", nil)

			// Close the client
			err = tempClient.Close(ctx)
			require.NoError(t, err)

			// Try operations
			_, err = tempRepo.FindOne(ctx, Filter{})
			require.ErrorIs(t, err, ErrClientClosed)

			_, err = tempRepo.InsertOne(ctx, Document{})
			require.ErrorIs(t, err, ErrClientClosed)

			_, err = tempRepo.Find(ctx, Filter{}, nil)
			require.ErrorIs(t, err, ErrClientClosed)

			_, err = tempRepo.Count(ctx, Filter{})
			require.ErrorIs(t, err, ErrClientClosed)
		})
	})
}

func TestRepository_ObjectIDConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	RunWithTestContainer(t, "test_objectid", func(t *testing.T, client *Client) {
		repo := NewRepository(client, "objectid_test", nil)
		ctx := context.Background()

		t.Run("Insert generates ObjectID", func(t *testing.T) {
			doc := Document{"name": "Test"}
			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)

			// ID should be valid hex ObjectID
			_, err = primitive.ObjectIDFromHex(id)
			require.NoError(t, err)
		})

		t.Run("FindByID with string ObjectID", func(t *testing.T) {
			doc := Document{"name": "FindMe"}
			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)

			found, err := repo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, "FindMe", found["name"])
		})

		t.Run("Query with _id filter string conversion", func(t *testing.T) {
			doc := Document{"name": "QueryMe"}
			id, err := repo.InsertOne(ctx, doc)
			require.NoError(t, err)

			q := &query.Query{
				Type: query.QuerySelect,
				Where: &query.WhereClause{
					Conditions: []query.Condition{
						{Field: "_id", Operator: query.OpEquals, Value: id},
					},
				},
			}

			results, err := repo.ExecuteQuery(ctx, q)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, "QueryMe", results[0]["name"])
		})
	})
}

func TestTranslateFilter(t *testing.T) {
	t.Run("nil filter", func(t *testing.T) {
		result := translateFilter(nil)
		assert.Equal(t, bson.M{}, result)
	})

	t.Run("empty filter", func(t *testing.T) {
		result := translateFilter(Filter{})
		assert.Equal(t, bson.M{}, result)
	})

	t.Run("simple filter", func(t *testing.T) {
		result := translateFilter(Filter{"name": "test"})
		assert.Equal(t, bson.M{"name": "test"}, result)
	})

	t.Run("_id with valid ObjectID string", func(t *testing.T) {
		oid := primitive.NewObjectID()
		result := translateFilter(Filter{"_id": oid.Hex()})
		assert.Equal(t, bson.M{"_id": oid}, result)
	})

	t.Run("_id with invalid ObjectID string", func(t *testing.T) {
		result := translateFilter(Filter{"_id": "not-an-objectid"})
		assert.Equal(t, bson.M{"_id": "not-an-objectid"}, result)
	})

	t.Run("_id with ObjectID type", func(t *testing.T) {
		oid := primitive.NewObjectID()
		result := translateFilter(Filter{"_id": oid})
		assert.Equal(t, bson.M{"_id": oid}, result)
	})
}

func TestTranslateCondition(t *testing.T) {
	t.Run("equals", func(t *testing.T) {
		cond := query.Condition{Field: "name", Operator: query.OpEquals, Value: "test"}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"name": "test"}, result)
	})

	t.Run("not equals", func(t *testing.T) {
		cond := query.Condition{Field: "name", Operator: query.OpNotEquals, Value: "test"}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"name": bson.M{"$ne": "test"}}, result)
	})

	t.Run("greater than", func(t *testing.T) {
		cond := query.Condition{Field: "age", Operator: query.OpGreaterThan, Value: 18}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"age": bson.M{"$gt": 18}}, result)
	})

	t.Run("in operator", func(t *testing.T) {
		values := []string{"a", "b", "c"}
		cond := query.Condition{Field: "status", Operator: query.OpIn, Value: values}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"status": bson.M{"$in": values}}, result)
	})

	t.Run("contains with regex", func(t *testing.T) {
		cond := query.Condition{Field: "name", Operator: query.OpContains, Value: "test"}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"name": bson.M{"$regex": "test", "$options": "i"}}, result)
	})

	t.Run("between", func(t *testing.T) {
		cond := query.Condition{
			Field:    "price",
			Operator: query.OpBetween,
			Value:    query.BetweenValue{Low: 10, High: 100},
		}
		result := translateCondition(cond)
		assert.Equal(t, bson.M{"price": bson.M{"$gte": 10, "$lte": 100}}, result)
	})
}

func TestEscapeRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with.dot", "with\\.dot"},
		{"with*star", "with\\*star"},
		{"test+plus", "test\\+plus"},
		{"(parens)", "\\(parens\\)"},
		{"[brackets]", "\\[brackets\\]"},
		{"{braces}", "\\{braces\\}"},
		{"all.special+chars*in?one^string$", "all\\.special\\+chars\\*in\\?one\\^string\\$"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeRegex(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLikeToRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"%test%", "^.*test.*$"},
		{"test%", "^test.*$"},
		{"%test", "^.*test$"},
		{"te_t", "^te.t$"},
		{"test", "^test$"},
		{"%te_t%", "^.*te.t.*$"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := likeToRegex(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatID(t *testing.T) {
	t.Run("ObjectID", func(t *testing.T) {
		oid := primitive.NewObjectID()
		result := formatID(oid)
		assert.Equal(t, oid.Hex(), result)
	})

	t.Run("string", func(t *testing.T) {
		result := formatID("custom-id")
		assert.Equal(t, "custom-id", result)
	})

	t.Run("int", func(t *testing.T) {
		result := formatID(123)
		assert.Equal(t, "123", result)
	})
}

func BenchmarkRepository_InsertOne(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	tc := SetupTestContainer(&testing.T{})
	defer tc.Container.Terminate(context.Background())

	cfg := DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = "bench"

	client, _ := New(context.Background(), cfg, nil)
	defer client.Close(context.Background())

	repo := NewRepository(client, "bench_insert", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := Document{
			"name":  "Benchmark User",
			"email": "bench@example.com",
			"index": i,
		}
		repo.InsertOne(ctx, doc)
	}
}

func BenchmarkRepository_Find(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	tc := SetupTestContainer(&testing.T{})
	defer tc.Container.Terminate(context.Background())

	cfg := DefaultConfig()
	cfg.URI = tc.URI
	cfg.Database = "bench"

	client, _ := New(context.Background(), cfg, nil)
	defer client.Close(context.Background())

	repo := NewRepository(client, "bench_find", nil)
	ctx := context.Background()

	// Insert test data
	docs := make([]Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = Document{
			"name":     "User",
			"category": "cat" + string(rune('A'+i%26)),
			"index":    i,
		}
	}
	repo.InsertMany(ctx, docs)

	// Create index
	repo.EnsureIndex(ctx, []string{"category"}, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter := Filter{"category": "catA"}
		opts := &FindOptions{Limit: 100}
		repo.Find(ctx, filter, opts)
	}
}
