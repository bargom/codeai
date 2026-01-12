// Package queue provides asynchronous webhook delivery with worker pool.
package queue

import (
	"context"
	"sync"
	"time"

	"github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/pkg/integration/webhook"
)

// Logger defines the logging interface for the queue.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Config holds configuration for the delivery queue.
type Config struct {
	QueueSize     int
	WorkerCount   int
	BatchSize     int
	DrainTimeout  time.Duration
}

// DefaultConfig returns a default queue configuration.
func DefaultConfig() Config {
	return Config{
		QueueSize:    1000,
		WorkerCount:  10,
		BatchSize:    100,
		DrainTimeout: 30 * time.Second,
	}
}

// DeliveryItem represents a webhook to be delivered.
type DeliveryItem struct {
	Webhook   *webhook.Webhook
	WebhookID string // The config ID for tracking
	EventID   string
}

// DeliveryQueue manages asynchronous webhook delivery with a worker pool.
type DeliveryQueue struct {
	queue      chan DeliveryItem
	workers    int
	client     *webhook.Client
	repository repository.WebhookRepository
	logger     Logger
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	stopped    bool
	mu         sync.RWMutex
}

// NewDeliveryQueue creates a new delivery queue with the specified configuration.
func NewDeliveryQueue(
	client *webhook.Client,
	repo repository.WebhookRepository,
	cfg Config,
	opts ...Option,
) *DeliveryQueue {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1000
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 10
	}

	q := &DeliveryQueue{
		queue:      make(chan DeliveryItem, cfg.QueueSize),
		workers:    cfg.WorkerCount,
		client:     client,
		repository: repo,
	}

	for _, opt := range opts {
		opt(q)
	}

	return q
}

// Option configures the DeliveryQueue.
type Option func(*DeliveryQueue)

// WithLogger sets the logger for the queue.
func WithLogger(logger Logger) Option {
	return func(q *DeliveryQueue) {
		q.logger = logger
	}
}

// Start begins processing webhooks from the queue.
func (q *DeliveryQueue) Start(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.ctx != nil {
		return // Already started
	}

	q.ctx, q.cancel = context.WithCancel(ctx)
	q.stopped = false

	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	if q.logger != nil {
		q.logger.Info("delivery queue started", "workers", q.workers)
	}
}

// Stop gracefully shuts down the queue, waiting for pending deliveries.
func (q *DeliveryQueue) Stop() {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		return
	}
	q.stopped = true
	q.mu.Unlock()

	if q.cancel != nil {
		q.cancel()
	}

	// Close the queue to signal workers to stop
	close(q.queue)

	// Wait for workers to finish
	q.wg.Wait()

	if q.logger != nil {
		q.logger.Info("delivery queue stopped")
	}
}

// Enqueue adds a webhook to the delivery queue.
// Returns false if the queue is full or stopped.
func (q *DeliveryQueue) Enqueue(item DeliveryItem) bool {
	q.mu.RLock()
	stopped := q.stopped
	q.mu.RUnlock()

	if stopped {
		if q.logger != nil {
			q.logger.Warn("attempted to enqueue to stopped queue",
				"webhookID", item.WebhookID,
			)
		}
		return false
	}

	select {
	case q.queue <- item:
		if q.logger != nil {
			q.logger.Debug("webhook enqueued",
				"webhookID", item.WebhookID,
				"eventID", item.EventID,
			)
		}
		return true
	default:
		if q.logger != nil {
			q.logger.Warn("queue full, dropping webhook",
				"webhookID", item.WebhookID,
				"eventID", item.EventID,
			)
		}
		return false
	}
}

// Pending returns the number of items waiting in the queue.
func (q *DeliveryQueue) Pending() int {
	return len(q.queue)
}

// worker processes webhooks from the queue.
func (q *DeliveryQueue) worker(id int) {
	defer q.wg.Done()

	if q.logger != nil {
		q.logger.Debug("worker started", "workerID", id)
	}

	for item := range q.queue {
		q.processItem(item)
	}

	if q.logger != nil {
		q.logger.Debug("worker stopped", "workerID", id)
	}
}

// processItem handles a single webhook delivery.
func (q *DeliveryQueue) processItem(item DeliveryItem) {
	ctx := q.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := q.client.Send(ctx, item.Webhook)

	// Record delivery result
	delivery := &repository.WebhookDelivery{
		ID:          item.Webhook.ID,
		WebhookID:   item.WebhookID,
		EventID:     item.EventID,
		URL:         item.Webhook.URL,
		RequestBody: item.Webhook.Payload,
		DeliveredAt: time.Now(),
	}

	if result != nil {
		delivery.StatusCode = result.StatusCode
		delivery.ResponseBody = result.ResponseBody
		delivery.Duration = result.Duration
		delivery.Attempts = result.Attempts
		delivery.Success = result.Success
		delivery.Error = result.Error
	} else if err != nil {
		delivery.Error = err.Error()
	}

	// Save delivery record
	if saveErr := q.repository.SaveDelivery(ctx, delivery); saveErr != nil {
		if q.logger != nil {
			q.logger.Error("failed to save delivery",
				"deliveryID", item.Webhook.ID,
				"error", saveErr.Error(),
			)
		}
	}

	// Handle success/failure
	if delivery.Success {
		if err := q.repository.ResetFailureCount(ctx, item.WebhookID); err != nil {
			if q.logger != nil {
				q.logger.Error("failed to reset failure count",
					"webhookID", item.WebhookID,
					"error", err.Error(),
				)
			}
		}

		if q.logger != nil {
			q.logger.Info("webhook delivered",
				"webhookID", item.WebhookID,
				"eventID", item.EventID,
				"statusCode", delivery.StatusCode,
				"duration", delivery.Duration,
			)
		}
	} else {
		if err := q.repository.IncrementFailureCount(ctx, item.WebhookID); err != nil {
			if q.logger != nil {
				q.logger.Error("failed to increment failure count",
					"webhookID", item.WebhookID,
					"error", err.Error(),
				)
			}
		}

		// Schedule retry
		retryPolicy := webhook.DefaultRetryPolicy()
		if delivery.Attempts < retryPolicy.MaxAttempts {
			nextRetryAt := time.Now().Add(retryPolicy.CalculateBackoff(delivery.Attempts))
			if retryErr := q.repository.UpdateDeliveryRetry(ctx, delivery.ID, nextRetryAt); retryErr != nil {
				if q.logger != nil {
					q.logger.Error("failed to schedule retry",
						"deliveryID", delivery.ID,
						"error", retryErr.Error(),
					)
				}
			}
		}

		if q.logger != nil {
			q.logger.Warn("webhook delivery failed",
				"webhookID", item.WebhookID,
				"eventID", item.EventID,
				"attempts", delivery.Attempts,
				"error", delivery.Error,
			)
		}
	}
}
