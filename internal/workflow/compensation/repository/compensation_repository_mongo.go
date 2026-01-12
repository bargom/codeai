package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	compensationsCollection = "compensation_records"
)

// MongoCompensationRepository implements CompensationRepository using MongoDB.
type MongoCompensationRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

// NewMongoCompensationRepository creates a new MongoDB-backed compensation repository.
func NewMongoCompensationRepository(db *mongo.Database) *MongoCompensationRepository {
	collection := db.Collection(compensationsCollection)
	return &MongoCompensationRepository{
		db:         db,
		collection: collection,
	}
}

// EnsureIndexes creates the necessary indexes for the compensation collection.
func (r *MongoCompensationRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "workflowId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "activityName", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetExpireAfterSeconds(86400 * 30), // 30 day TTL
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// SaveCompensationRecord creates or updates a compensation record.
func (r *MongoCompensationRepository) SaveCompensationRecord(ctx context.Context, record *CompensationRecord) error {
	record.UpdatedAt = time.Now().UTC()

	if record.ID == "" {
		record.ID = primitive.NewObjectID().Hex()
		record.CreatedAt = time.Now().UTC()
		_, err := r.collection.InsertOne(ctx, record)
		return err
	}

	filter := bson.M{"_id": record.ID}
	update := bson.M{"$set": record}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetCompensationRecord retrieves a compensation record by ID.
func (r *MongoCompensationRepository) GetCompensationRecord(ctx context.Context, recordID string) (*CompensationRecord, error) {
	var record CompensationRecord
	filter := bson.M{"_id": recordID}
	err := r.collection.FindOne(ctx, filter).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("compensation record not found: %s", recordID)
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetCompensationHistory retrieves all compensation records for a workflow.
func (r *MongoCompensationRepository) GetCompensationHistory(ctx context.Context, workflowID string) ([]CompensationRecord, error) {
	filter := bson.M{"workflowId": workflowID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []CompensationRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// ListCompensations retrieves compensation records with filtering.
func (r *MongoCompensationRepository) ListCompensations(ctx context.Context, filter ListCompensationsFilter) ([]CompensationRecord, error) {
	query := bson.M{}

	if filter.WorkflowID != "" {
		query["workflowId"] = filter.WorkflowID
	}
	if filter.ActivityName != "" {
		query["activityName"] = filter.ActivityName
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if !filter.StartTime.IsZero() || !filter.EndTime.IsZero() {
		timeQuery := bson.M{}
		if !filter.StartTime.IsZero() {
			timeQuery["$gte"] = filter.StartTime
		}
		if !filter.EndTime.IsZero() {
			timeQuery["$lte"] = filter.EndTime
		}
		query["createdAt"] = timeQuery
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []CompensationRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// MarkCompensationStarted updates record status to running.
func (r *MongoCompensationRepository) MarkCompensationStarted(ctx context.Context, recordID string) error {
	filter := bson.M{"_id": recordID}
	update := bson.M{
		"$set": bson.M{
			"status":    StatusRunning,
			"startedAt": time.Now().UTC(),
			"updatedAt": time.Now().UTC(),
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("compensation record not found: %s", recordID)
	}
	return nil
}

// MarkCompensationCompleted updates record status to completed.
func (r *MongoCompensationRepository) MarkCompensationCompleted(ctx context.Context, recordID string) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": recordID}

	// Get the record to calculate duration
	var record CompensationRecord
	if err := r.collection.FindOne(ctx, filter).Decode(&record); err != nil {
		return err
	}

	var duration time.Duration
	if !record.StartedAt.IsZero() {
		duration = now.Sub(record.StartedAt)
	}

	update := bson.M{
		"$set": bson.M{
			"status":      StatusCompleted,
			"completedAt": now,
			"duration":    duration,
			"updatedAt":   now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("compensation record not found: %s", recordID)
	}
	return nil
}

// MarkCompensationFailed updates record status to failed with error.
func (r *MongoCompensationRepository) MarkCompensationFailed(ctx context.Context, recordID string, err error) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": recordID}

	// Get the record to calculate duration
	var record CompensationRecord
	if findErr := r.collection.FindOne(ctx, filter).Decode(&record); findErr != nil {
		return findErr
	}

	var duration time.Duration
	if !record.StartedAt.IsZero() {
		duration = now.Sub(record.StartedAt)
	}

	update := bson.M{
		"$set": bson.M{
			"status":      StatusFailed,
			"error":       err.Error(),
			"completedAt": now,
			"duration":    duration,
			"updatedAt":   now,
		},
	}

	result, updateErr := r.collection.UpdateOne(ctx, filter, update)
	if updateErr != nil {
		return updateErr
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("compensation record not found: %s", recordID)
	}
	return nil
}

// GetCompensationSummary retrieves aggregated compensation statistics.
func (r *MongoCompensationRepository) GetCompensationSummary(ctx context.Context, workflowID string) (*CompensationSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"workflowId": workflowID}}},
		{{Key: "$group", Value: bson.M{
			"_id":       "$workflowId",
			"total":     bson.M{"$sum": 1},
			"completed": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", StatusCompleted}}, 1, 0}}},
			"failed":    bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", StatusFailed}}, 1, 0}}},
			"skipped":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", StatusSkipped}}, 1, 0}}},
			"pending":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", StatusPending}}, 1, 0}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID        string `bson:"_id"`
		Total     int    `bson:"total"`
		Completed int    `bson:"completed"`
		Failed    int    `bson:"failed"`
		Skipped   int    `bson:"skipped"`
		Pending   int    `bson:"pending"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &CompensationSummary{
			WorkflowID: workflowID,
		}, nil
	}

	return &CompensationSummary{
		WorkflowID:         workflowID,
		TotalCompensations: results[0].Total,
		Completed:          results[0].Completed,
		Failed:             results[0].Failed,
		Skipped:            results[0].Skipped,
		Pending:            results[0].Pending,
	}, nil
}

// DeleteCompensationHistory removes all compensation records for a workflow.
func (r *MongoCompensationRepository) DeleteCompensationHistory(ctx context.Context, workflowID string) error {
	filter := bson.M{"workflowId": workflowID}
	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}

// GetPendingCompensations retrieves all pending compensations for retry.
func (r *MongoCompensationRepository) GetPendingCompensations(ctx context.Context, limit int) ([]CompensationRecord, error) {
	filter := bson.M{
		"status": bson.M{"$in": []CompensationStatus{StatusPending, StatusFailed}},
		"retries": bson.M{"$lt": 3}, // Max 3 retries
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []CompensationRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// IncrementRetryCount increments the retry counter for a compensation record.
func (r *MongoCompensationRepository) IncrementRetryCount(ctx context.Context, recordID string) error {
	filter := bson.M{"_id": recordID}
	update := bson.M{
		"$inc": bson.M{"retries": 1},
		"$set": bson.M{"updatedAt": time.Now().UTC()},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("compensation record not found: %s", recordID)
	}
	return nil
}
