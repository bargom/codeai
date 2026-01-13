// Package mongodb provides MongoDB database connectivity and repository operations.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/pagination"
	"github.com/bargom/codeai/internal/query"
)

// Common errors returned by repository operations.
var (
	ErrNotFound       = errors.New("document not found")
	ErrDuplicateKey   = errors.New("duplicate key error")
	ErrInvalidID      = errors.New("invalid document ID")
	ErrInvalidFilter  = errors.New("invalid filter")
	ErrNoDocuments    = errors.New("no documents matched")
	ErrClientClosed   = errors.New("mongodb client is closed")
)

// Document represents a generic MongoDB document.
type Document map[string]interface{}

// Filter represents a MongoDB query filter.
type Filter bson.M

// Update represents a MongoDB update document.
type Update bson.M

// FindOptions contains options for find operations.
type FindOptions struct {
	Sort       bson.D
	Limit      int64
	Skip       int64
	Projection bson.M
}

// Repository provides CRUD operations for MongoDB collections.
type Repository struct {
	client     *Client
	database   *mongo.Database
	collection *mongo.Collection
	collName   string
	logger     *slog.Logger
}

// NewRepository creates a new repository for the specified collection.
func NewRepository(client *Client, collectionName string, logger *slog.Logger) *Repository {
	if logger == nil {
		logger = slog.Default()
	}

	db := client.Database()
	return &Repository{
		client:     client,
		database:   db,
		collection: db.Collection(collectionName),
		collName:   collectionName,
		logger:     logger.With(slog.String("collection", collectionName)),
	}
}

// Collection returns the underlying mongo.Collection.
func (r *Repository) Collection() *mongo.Collection {
	return r.collection
}

// =============================================================================
// Single Document Operations
// =============================================================================

// FindOne retrieves a single document matching the filter.
func (r *Repository) FindOne(ctx context.Context, filter Filter) (Document, error) {
	if r.client.IsClosed() {
		return nil, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	var doc Document
	err := r.collection.FindOne(ctx, mongoFilter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("findOne failed: %w", err)
	}

	return doc, nil
}

// FindByID retrieves a document by its ID.
func (r *Repository) FindByID(ctx context.Context, id string) (Document, error) {
	filter := Filter{"_id": id}

	// Try as string first, then as ObjectID
	doc, err := r.FindOne(ctx, filter)
	if err != nil && errors.Is(err, ErrNotFound) {
		// Try parsing as ObjectID
		oid, parseErr := primitive.ObjectIDFromHex(id)
		if parseErr == nil {
			filter = Filter{"_id": oid}
			return r.FindOne(ctx, filter)
		}
	}

	return doc, err
}

// InsertOne inserts a single document into the collection.
func (r *Repository) InsertOne(ctx context.Context, doc Document) (string, error) {
	if r.client.IsClosed() {
		return "", ErrClientClosed
	}

	// Set timestamps if not present
	now := time.Now().UTC()
	if _, ok := doc["createdAt"]; !ok {
		doc["createdAt"] = now
	}
	if _, ok := doc["updatedAt"]; !ok {
		doc["updatedAt"] = now
	}

	// Generate _id if not present
	if _, ok := doc["_id"]; !ok {
		doc["_id"] = primitive.NewObjectID()
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: %v", ErrDuplicateKey, err)
		}
		return "", fmt.Errorf("insertOne failed: %w", err)
	}

	// Extract the inserted ID
	switch v := result.InsertedID.(type) {
	case primitive.ObjectID:
		return v.Hex(), nil
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// UpdateOne updates a single document matching the filter.
func (r *Repository) UpdateOne(ctx context.Context, filter Filter, update Update) (int64, error) {
	if r.client.IsClosed() {
		return 0, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	// Ensure updatedAt is set
	if _, ok := update["$set"]; ok {
		if setDoc, ok := update["$set"].(bson.M); ok {
			setDoc["updatedAt"] = time.Now().UTC()
		}
	} else if _, ok := update["$set"]; !ok {
		// Wrap raw update in $set
		update = Update{"$set": bson.M{"updatedAt": time.Now().UTC()}}
		for k, v := range update {
			if k != "$set" {
				update["$set"].(bson.M)[k] = v
			}
		}
	}

	result, err := r.collection.UpdateOne(ctx, mongoFilter, update)
	if err != nil {
		return 0, fmt.Errorf("updateOne failed: %w", err)
	}

	return result.ModifiedCount, nil
}

// DeleteOne deletes a single document matching the filter.
func (r *Repository) DeleteOne(ctx context.Context, filter Filter) (int64, error) {
	if r.client.IsClosed() {
		return 0, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	result, err := r.collection.DeleteOne(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("deleteOne failed: %w", err)
	}

	return result.DeletedCount, nil
}

// =============================================================================
// Multiple Document Operations
// =============================================================================

// Find retrieves multiple documents matching the filter with options.
func (r *Repository) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]Document, error) {
	if r.client.IsClosed() {
		return nil, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)
	findOpts := options.Find()

	if opts != nil {
		if len(opts.Sort) > 0 {
			findOpts.SetSort(opts.Sort)
		}
		if opts.Limit > 0 {
			findOpts.SetLimit(opts.Limit)
		}
		if opts.Skip > 0 {
			findOpts.SetSkip(opts.Skip)
		}
		if opts.Projection != nil {
			findOpts.SetProjection(opts.Projection)
		}
	}

	cursor, err := r.collection.Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor decode failed: %w", err)
	}

	if docs == nil {
		docs = []Document{}
	}

	return docs, nil
}

// InsertMany inserts multiple documents into the collection.
func (r *Repository) InsertMany(ctx context.Context, docs []Document) ([]string, error) {
	if r.client.IsClosed() {
		return nil, ErrClientClosed
	}

	if len(docs) == 0 {
		return []string{}, nil
	}

	// Prepare documents with timestamps and IDs
	now := time.Now().UTC()
	insertDocs := make([]interface{}, len(docs))
	for i, doc := range docs {
		if _, ok := doc["createdAt"]; !ok {
			doc["createdAt"] = now
		}
		if _, ok := doc["updatedAt"]; !ok {
			doc["updatedAt"] = now
		}
		if _, ok := doc["_id"]; !ok {
			doc["_id"] = primitive.NewObjectID()
		}
		insertDocs[i] = doc
	}

	result, err := r.collection.InsertMany(ctx, insertDocs)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: %v", ErrDuplicateKey, err)
		}
		return nil, fmt.Errorf("insertMany failed: %w", err)
	}

	ids := make([]string, len(result.InsertedIDs))
	for i, id := range result.InsertedIDs {
		switch v := id.(type) {
		case primitive.ObjectID:
			ids[i] = v.Hex()
		case string:
			ids[i] = v
		default:
			ids[i] = fmt.Sprintf("%v", v)
		}
	}

	return ids, nil
}

// UpdateMany updates multiple documents matching the filter.
func (r *Repository) UpdateMany(ctx context.Context, filter Filter, update Update) (int64, error) {
	if r.client.IsClosed() {
		return 0, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	// Ensure updatedAt is set
	if setDoc, ok := update["$set"].(bson.M); ok {
		setDoc["updatedAt"] = time.Now().UTC()
	} else if update["$set"] == nil {
		if update["$currentDate"] == nil {
			update["$currentDate"] = bson.M{"updatedAt": true}
		}
	}

	result, err := r.collection.UpdateMany(ctx, mongoFilter, update)
	if err != nil {
		return 0, fmt.Errorf("updateMany failed: %w", err)
	}

	return result.ModifiedCount, nil
}

// DeleteMany deletes multiple documents matching the filter.
func (r *Repository) DeleteMany(ctx context.Context, filter Filter) (int64, error) {
	if r.client.IsClosed() {
		return 0, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	result, err := r.collection.DeleteMany(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("deleteMany failed: %w", err)
	}

	return result.DeletedCount, nil
}

// =============================================================================
// Transaction Support
// =============================================================================

// WithTransaction executes a function within a MongoDB transaction.
func (r *Repository) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	return r.client.WithTransaction(ctx, fn)
}

// =============================================================================
// Count and Exists
// =============================================================================

// Count returns the count of documents matching the filter.
func (r *Repository) Count(ctx context.Context, filter Filter) (int64, error) {
	if r.client.IsClosed() {
		return 0, ErrClientClosed
	}

	mongoFilter := translateFilter(filter)

	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("count failed: %w", err)
	}

	return count, nil
}

// Exists checks if at least one document matches the filter.
func (r *Repository) Exists(ctx context.Context, filter Filter) (bool, error) {
	count, err := r.Count(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// =============================================================================
// Pagination Support
// =============================================================================

// FindWithPagination retrieves documents with pagination support.
func (r *Repository) FindWithPagination(
	ctx context.Context,
	filter Filter,
	page pagination.PageRequest,
	sortField string,
	sortDir query.OrderDirection,
) (*pagination.PageResponse[Document], error) {
	if r.client.IsClosed() {
		return nil, ErrClientClosed
	}

	page.Validate()
	mongoFilter := translateFilter(filter)

	// Determine sort order
	sortValue := 1
	if sortDir == query.OrderDesc {
		sortValue = -1
	}
	if sortField == "" {
		sortField = "_id"
	}

	if page.IsCursorPagination() {
		return r.findWithCursorPagination(ctx, mongoFilter, page, sortField, sortValue)
	}
	return r.findWithOffsetPagination(ctx, mongoFilter, page, sortField, sortValue)
}

// findWithOffsetPagination implements offset-based pagination.
func (r *Repository) findWithOffsetPagination(
	ctx context.Context,
	filter bson.M,
	page pagination.PageRequest,
	sortField string,
	sortValue int,
) (*pagination.PageResponse[Document], error) {
	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count failed: %w", err)
	}

	// Fetch documents with limit+1 to detect hasNext
	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortValue}}).
		SetSkip(int64(page.GetOffset())).
		SetLimit(int64(page.Limit + 1))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor decode failed: %w", err)
	}

	// Determine hasNext
	hasNext := len(docs) > page.Limit
	if hasNext {
		docs = docs[:page.Limit]
	}

	// Build response
	pageNum := page.Page
	return &pagination.PageResponse[Document]{
		Data: docs,
		Pagination: pagination.PageInfo{
			Total:       &total,
			Page:        &pageNum,
			Limit:       page.Limit,
			HasNext:     hasNext,
			HasPrevious: page.Page > 1,
		},
	}, nil
}

// findWithCursorPagination implements cursor-based pagination using _id.
func (r *Repository) findWithCursorPagination(
	ctx context.Context,
	filter bson.M,
	page pagination.PageRequest,
	sortField string,
	sortValue int,
) (*pagination.PageResponse[Document], error) {
	// Parse cursor if provided
	cursorStr := page.GetCursor()
	isBackward := page.Before != ""
	if isBackward {
		cursorStr = page.Before
	}

	if cursorStr != "" {
		cursor, err := pagination.DecodeCursor(cursorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		// Build cursor condition
		var cursorFilter bson.M
		cursorID := cursor.ID

		// Try to parse as ObjectID
		if oid, err := primitive.ObjectIDFromHex(cursorID); err == nil {
			if isBackward {
				cursorFilter = bson.M{"_id": bson.M{"$lt": oid}}
			} else {
				cursorFilter = bson.M{"_id": bson.M{"$gt": oid}}
			}
		} else {
			if isBackward {
				cursorFilter = bson.M{"_id": bson.M{"$lt": cursorID}}
			} else {
				cursorFilter = bson.M{"_id": bson.M{"$gt": cursorID}}
			}
		}

		// Merge with existing filter
		if len(filter) > 0 {
			filter = bson.M{"$and": []bson.M{filter, cursorFilter}}
		} else {
			filter = cursorFilter
		}
	}

	// Adjust sort direction for backward pagination
	effectiveSortValue := sortValue
	if isBackward {
		effectiveSortValue = -sortValue
	}

	// Fetch documents with limit+1 to detect hasNext
	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: effectiveSortValue}}).
		SetLimit(int64(page.Limit + 1))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor decode failed: %w", err)
	}

	// Determine hasNext/hasPrevious
	hasMore := len(docs) > page.Limit
	if hasMore {
		docs = docs[:page.Limit]
	}

	// Reverse documents if backward pagination
	if isBackward {
		for i, j := 0, len(docs)-1; i < j; i, j = i+1, j-1 {
			docs[i], docs[j] = docs[j], docs[i]
		}
	}

	// Build cursors
	var nextCursor, prevCursor string
	if len(docs) > 0 {
		// Next cursor from last document
		lastDoc := docs[len(docs)-1]
		if id, ok := lastDoc["_id"]; ok {
			nextCursor = pagination.EncodeCursor(pagination.Cursor{
				ID:        formatID(id),
				Direction: "forward",
			})
		}

		// Prev cursor from first document
		firstDoc := docs[0]
		if id, ok := firstDoc["_id"]; ok {
			prevCursor = pagination.EncodeCursor(pagination.Cursor{
				ID:        formatID(id),
				Direction: "backward",
			})
		}
	}

	hasNext := hasMore
	hasPrevious := cursorStr != ""
	if isBackward {
		hasNext, hasPrevious = hasPrevious, hasMore
	}

	return &pagination.PageResponse[Document]{
		Data: docs,
		Pagination: pagination.PageInfo{
			Limit:       page.Limit,
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
			NextCursor:  nextCursor,
			PrevCursor:  prevCursor,
		},
	}, nil
}

// =============================================================================
// Index Management
// =============================================================================

// EnsureIndexes creates indexes based on DSL MongoIndexDecl definitions.
func (r *Repository) EnsureIndexes(ctx context.Context, indexes []*ast.MongoIndexDecl) error {
	if r.client.IsClosed() {
		return ErrClientClosed
	}

	if len(indexes) == 0 {
		return nil
	}

	var indexModels []mongo.IndexModel

	for _, idx := range indexes {
		keys := bson.D{}

		for _, field := range idx.Fields {
			switch idx.IndexKind {
			case "text":
				keys = append(keys, bson.E{Key: field, Value: "text"})
			case "geospatial", "2dsphere":
				keys = append(keys, bson.E{Key: field, Value: "2dsphere"})
			default:
				// Regular ascending index
				keys = append(keys, bson.E{Key: field, Value: 1})
			}
		}

		indexOpts := options.Index()
		if idx.Unique {
			indexOpts.SetUnique(true)
		}

		indexModels = append(indexModels, mongo.IndexModel{
			Keys:    keys,
			Options: indexOpts,
		})
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("create indexes failed: %w", err)
	}

	r.logger.Info("indexes created", slog.Int("count", len(indexModels)))
	return nil
}

// EnsureIndex creates a single index on the specified fields.
func (r *Repository) EnsureIndex(ctx context.Context, fields []string, unique bool) error {
	if r.client.IsClosed() {
		return ErrClientClosed
	}

	keys := bson.D{}
	for _, field := range fields {
		keys = append(keys, bson.E{Key: field, Value: 1})
	}

	indexOpts := options.Index()
	if unique {
		indexOpts.SetUnique(true)
	}

	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    keys,
		Options: indexOpts,
	})
	if err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}

	return nil
}

// =============================================================================
// Query Translation
// =============================================================================

// ExecuteQuery executes a query.Query and returns matching documents.
func (r *Repository) ExecuteQuery(ctx context.Context, q *query.Query) ([]Document, error) {
	if r.client.IsClosed() {
		return nil, ErrClientClosed
	}

	filter := bson.M{}
	if q.Where != nil {
		filter = translateWhereClause(q.Where)
	}

	opts := options.Find()

	// Handle field selection
	if len(q.Fields) > 0 {
		projection := bson.M{}
		for _, f := range q.Fields {
			projection[f] = 1
		}
		opts.SetProjection(projection)
	}

	// Handle ordering
	if len(q.OrderBy) > 0 {
		sort := bson.D{}
		for _, o := range q.OrderBy {
			sortVal := 1
			if o.Direction == query.OrderDesc {
				sortVal = -1
			}
			sort = append(sort, bson.E{Key: o.Field, Value: sortVal})
		}
		opts.SetSort(sort)
	}

	// Handle limit/offset
	if q.Limit != nil {
		opts.SetLimit(int64(*q.Limit))
	}
	if q.Offset != nil {
		opts.SetSkip(int64(*q.Offset))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor decode failed: %w", err)
	}

	if docs == nil {
		docs = []Document{}
	}

	return docs, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// translateFilter converts a Filter to a MongoDB bson.M query.
func translateFilter(filter Filter) bson.M {
	if filter == nil {
		return bson.M{}
	}

	result := bson.M{}
	for k, v := range filter {
		// Handle ObjectID string conversion
		if k == "_id" {
			if idStr, ok := v.(string); ok {
				if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
					result[k] = oid
					continue
				}
			}
		}
		result[k] = v
	}
	return result
}

// translateWhereClause converts a query.WhereClause to MongoDB bson.M.
func translateWhereClause(where *query.WhereClause) bson.M {
	if where == nil || (len(where.Conditions) == 0 && len(where.Groups) == 0) {
		return bson.M{}
	}

	conditions := make([]bson.M, 0, len(where.Conditions)+len(where.Groups))

	// Translate individual conditions
	for _, cond := range where.Conditions {
		mongoCondition := translateCondition(cond)
		if len(mongoCondition) > 0 {
			conditions = append(conditions, mongoCondition)
		}
	}

	// Translate nested groups
	for _, group := range where.Groups {
		nestedCondition := translateWhereClause(group)
		if len(nestedCondition) > 0 {
			conditions = append(conditions, nestedCondition)
		}
	}

	if len(conditions) == 0 {
		return bson.M{}
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	// Combine with AND or OR
	if where.Operator == query.LogicalOr {
		return bson.M{"$or": conditions}
	}
	return bson.M{"$and": conditions}
}

// translateCondition converts a single query.Condition to MongoDB bson.M.
func translateCondition(cond query.Condition) bson.M {
	field := cond.Field
	value := cond.Value

	// Handle nested conditions
	if cond.Nested != nil {
		return translateWhereClause(cond.Nested)
	}

	// Handle ObjectID conversion for _id field
	if field == "_id" {
		if idStr, ok := value.(string); ok {
			if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
				value = oid
			}
		}
	}

	var mongoOp bson.M

	switch cond.Operator {
	case query.OpEquals:
		mongoOp = bson.M{field: value}
	case query.OpNotEquals:
		mongoOp = bson.M{field: bson.M{"$ne": value}}
	case query.OpGreaterThan:
		mongoOp = bson.M{field: bson.M{"$gt": value}}
	case query.OpGreaterThanOrEqual:
		mongoOp = bson.M{field: bson.M{"$gte": value}}
	case query.OpLessThan:
		mongoOp = bson.M{field: bson.M{"$lt": value}}
	case query.OpLessThanOrEqual:
		mongoOp = bson.M{field: bson.M{"$lte": value}}
	case query.OpContains:
		// Case-insensitive contains
		if str, ok := value.(string); ok {
			mongoOp = bson.M{field: bson.M{"$regex": str, "$options": "i"}}
		}
	case query.OpStartsWith:
		if str, ok := value.(string); ok {
			mongoOp = bson.M{field: bson.M{"$regex": "^" + escapeRegex(str), "$options": "i"}}
		}
	case query.OpEndsWith:
		if str, ok := value.(string); ok {
			mongoOp = bson.M{field: bson.M{"$regex": escapeRegex(str) + "$", "$options": "i"}}
		}
	case query.OpIn:
		mongoOp = bson.M{field: bson.M{"$in": value}}
	case query.OpNotIn:
		mongoOp = bson.M{field: bson.M{"$nin": value}}
	case query.OpIsNull:
		mongoOp = bson.M{field: nil}
	case query.OpIsNotNull:
		mongoOp = bson.M{field: bson.M{"$ne": nil}}
	case query.OpIncludes, query.OpArrayContains:
		// Array contains element
		mongoOp = bson.M{field: value}
	case query.OpLike:
		if str, ok := value.(string); ok {
			// Convert SQL LIKE pattern to regex
			pattern := likeToRegex(str)
			mongoOp = bson.M{field: bson.M{"$regex": pattern}}
		}
	case query.OpILike:
		if str, ok := value.(string); ok {
			pattern := likeToRegex(str)
			mongoOp = bson.M{field: bson.M{"$regex": pattern, "$options": "i"}}
		}
	case query.OpBetween:
		if between, ok := value.(query.BetweenValue); ok {
			mongoOp = bson.M{field: bson.M{"$gte": between.Low, "$lte": between.High}}
		}
	case query.OpFuzzy:
		// Text search (requires text index)
		if str, ok := value.(string); ok {
			mongoOp = bson.M{"$text": bson.M{"$search": str}}
		}
	default:
		// Default to equality
		mongoOp = bson.M{field: value}
	}

	// Apply NOT modifier
	if cond.Not && mongoOp != nil {
		return bson.M{"$not": mongoOp}
	}

	return mongoOp
}

// escapeRegex escapes special regex characters in a string.
func escapeRegex(s string) string {
	special := []string{"\\", ".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
	result := s
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// likeToRegex converts a SQL LIKE pattern to a MongoDB regex pattern.
func likeToRegex(pattern string) string {
	// Escape regex special characters first
	escaped := escapeRegex(pattern)
	// Convert LIKE wildcards to regex
	escaped = strings.ReplaceAll(escaped, "%", ".*")
	escaped = strings.ReplaceAll(escaped, "_", ".")
	return "^" + escaped + "$"
}

// formatID converts an ID value to a string.
func formatID(id interface{}) string {
	switch v := id.(type) {
	case primitive.ObjectID:
		return v.Hex()
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
