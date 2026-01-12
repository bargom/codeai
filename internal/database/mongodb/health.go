package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// HealthStatus represents the health state of the MongoDB connection.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the connection is healthy.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the connection is unhealthy.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded indicates the connection is partially available.
	HealthStatusDegraded HealthStatus = "degraded"
)

// HealthCheck provides health check functionality for MongoDB connections.
type HealthCheck struct {
	client  *Client
	logger  *slog.Logger
	timeout time.Duration
}

// HealthCheckResult contains the result of a health check.
type HealthCheckResult struct {
	// Status is the overall health status
	Status HealthStatus

	// Message provides additional details about the health status
	Message string

	// Latency is the time taken to perform the health check
	Latency time.Duration

	// Details contains additional health check information
	Details map[string]interface{}

	// Timestamp is when the health check was performed
	Timestamp time.Time
}

// NewHealthCheck creates a new health check for the given client.
func NewHealthCheck(client *Client, logger *slog.Logger) *HealthCheck {
	if logger == nil {
		logger = slog.Default()
	}
	return &HealthCheck{
		client:  client,
		logger:  logger.With(slog.String("component", "mongodb-health")),
		timeout: 5 * time.Second,
	}
}

// SetTimeout sets the timeout for health check operations.
func (h *HealthCheck) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
}

// Check performs a health check on the MongoDB connection.
func (h *HealthCheck) Check(ctx context.Context) HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}

	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Check if client is closed
	if h.client.IsClosed() {
		result.Status = HealthStatusUnhealthy
		result.Message = "client is closed"
		result.Latency = time.Since(start)
		h.logger.Warn("health check failed: client closed")
		return result
	}

	mongoClient := h.client.Client()
	if mongoClient == nil {
		result.Status = HealthStatusUnhealthy
		result.Message = "client is nil"
		result.Latency = time.Since(start)
		h.logger.Warn("health check failed: nil client")
		return result
	}

	// Ping the primary
	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		result.Status = HealthStatusUnhealthy
		result.Message = fmt.Sprintf("ping failed: %v", err)
		result.Latency = time.Since(start)
		h.logger.Warn("health check failed",
			slog.String("error", err.Error()),
			slog.Duration("latency", result.Latency))
		return result
	}

	// Get server status for additional details
	db := h.client.Database()
	if db != nil {
		var serverStatus bson.M
		err := db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&serverStatus)
		if err == nil {
			if connections, ok := serverStatus["connections"].(bson.M); ok {
				result.Details["currentConnections"] = connections["current"]
				result.Details["availableConnections"] = connections["available"]
			}
			if version, ok := serverStatus["version"].(string); ok {
				result.Details["version"] = version
			}
			if uptime, ok := serverStatus["uptime"].(float64); ok {
				result.Details["uptimeSeconds"] = uptime
			}
		}
	}

	result.Status = HealthStatusHealthy
	result.Message = "connection is healthy"
	result.Latency = time.Since(start)

	h.logger.Debug("health check passed",
		slog.Duration("latency", result.Latency))

	return result
}

// IsHealthy returns true if the connection is healthy.
func (h *HealthCheck) IsHealthy(ctx context.Context) bool {
	result := h.Check(ctx)
	return result.Status == HealthStatusHealthy
}

// Ping performs a simple ping to verify connectivity.
func (h *HealthCheck) Ping(ctx context.Context) error {
	if h.client.IsClosed() {
		return fmt.Errorf("mongodb: client is closed")
	}

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	mongoClient := h.client.Client()
	if mongoClient == nil {
		return fmt.Errorf("mongodb: client is nil")
	}

	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongodb: ping failed: %w", err)
	}

	return nil
}

// CheckReadiness verifies the database is ready to accept operations.
func (h *HealthCheck) CheckReadiness(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	if h.client.IsClosed() {
		return fmt.Errorf("mongodb: client is closed")
	}

	db := h.client.Database()
	if db == nil {
		return fmt.Errorf("mongodb: database is nil")
	}

	// Try to list collections as a readiness check
	_, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("mongodb: not ready: %w", err)
	}

	return nil
}
