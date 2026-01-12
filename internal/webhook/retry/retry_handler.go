// Package retry provides automatic retry handling for failed webhook deliveries.
package retry

import (
	"context"
	"sync"
	"time"

	"github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/internal/webhook/service"
)

// Logger defines the logging interface for the retry handler.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Config holds configuration for the retry handler.
type Config struct {
	CheckInterval time.Duration // How often to check for retryable deliveries
	BatchSize     int           // Max number of deliveries to retry per check
	MaxRetries    int           // Maximum retry attempts
}

// DefaultConfig returns a default retry handler configuration.
func DefaultConfig() Config {
	return Config{
		CheckInterval: 1 * time.Minute,
		BatchSize:     100,
		MaxRetries:    5,
	}
}

// RetryHandler periodically checks for and retries failed webhook deliveries.
type RetryHandler struct {
	repository     repository.WebhookRepository
	webhookService *service.WebhookService
	config         Config
	logger         Logger
	ticker         *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	running        bool
	mu             sync.RWMutex
}

// Option configures the RetryHandler.
type Option func(*RetryHandler)

// WithLogger sets the logger for the handler.
func WithLogger(logger Logger) Option {
	return func(h *RetryHandler) {
		h.logger = logger
	}
}

// WithConfig sets the configuration for the handler.
func WithConfig(cfg Config) Option {
	return func(h *RetryHandler) {
		h.config = cfg
	}
}

// NewRetryHandler creates a new retry handler.
func NewRetryHandler(
	repo repository.WebhookRepository,
	svc *service.WebhookService,
	opts ...Option,
) *RetryHandler {
	h := &RetryHandler{
		repository:     repo,
		webhookService: svc,
		config:         DefaultConfig(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Start begins the periodic retry processing.
func (h *RetryHandler) Start(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	h.ctx, h.cancel = context.WithCancel(ctx)
	h.ticker = time.NewTicker(h.config.CheckInterval)
	h.running = true

	h.wg.Add(1)
	go h.run()

	if h.logger != nil {
		h.logger.Info("retry handler started",
			"checkInterval", h.config.CheckInterval,
			"batchSize", h.config.BatchSize,
		)
	}
}

// Stop gracefully shuts down the retry handler.
func (h *RetryHandler) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	if h.cancel != nil {
		h.cancel()
	}

	if h.ticker != nil {
		h.ticker.Stop()
	}

	h.wg.Wait()

	if h.logger != nil {
		h.logger.Info("retry handler stopped")
	}
}

// run is the main loop that checks for and processes retries.
func (h *RetryHandler) run() {
	defer h.wg.Done()

	// Process immediately on startup
	h.processRetries()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.ticker.C:
			h.processRetries()
		}
	}
}

// processRetries fetches and retries failed deliveries.
func (h *RetryHandler) processRetries() {
	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	failures, err := h.repository.GetFailedDeliveries(ctx, h.config.BatchSize)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to get failed deliveries", "error", err.Error())
		}
		return
	}

	if len(failures) == 0 {
		return
	}

	if h.logger != nil {
		h.logger.Debug("processing retries", "count", len(failures))
	}

	var successCount, failCount int

	for _, delivery := range failures {
		// Check if we should still retry
		if delivery.Attempts >= h.config.MaxRetries {
			// Mark as not retryable by clearing NextRetryAt
			if err := h.clearRetry(ctx, delivery.ID); err != nil {
				if h.logger != nil {
					h.logger.Error("failed to clear retry",
						"deliveryID", delivery.ID,
						"error", err.Error(),
					)
				}
			}
			continue
		}

		// Check if webhook is still active
		webhook, err := h.repository.GetWebhook(ctx, delivery.WebhookID)
		if err != nil {
			if h.logger != nil {
				h.logger.Error("failed to get webhook",
					"webhookID", delivery.WebhookID,
					"error", err.Error(),
				)
			}
			continue
		}

		if !webhook.Active {
			// Webhook is disabled, clear the retry
			if err := h.clearRetry(ctx, delivery.ID); err != nil {
				if h.logger != nil {
					h.logger.Error("failed to clear retry for disabled webhook",
						"deliveryID", delivery.ID,
						"error", err.Error(),
					)
				}
			}
			continue
		}

		// Attempt retry
		if err := h.webhookService.RetryFailedWebhook(ctx, delivery.ID); err != nil {
			failCount++
			if h.logger != nil {
				h.logger.Warn("retry failed",
					"deliveryID", delivery.ID,
					"webhookID", delivery.WebhookID,
					"attempt", delivery.Attempts+1,
					"error", err.Error(),
				)
			}
		} else {
			successCount++
			if h.logger != nil {
				h.logger.Info("retry succeeded",
					"deliveryID", delivery.ID,
					"webhookID", delivery.WebhookID,
					"attempt", delivery.Attempts+1,
				)
			}
		}

		// Check for cancellation between retries
		select {
		case <-h.ctx.Done():
			return
		default:
		}
	}

	if h.logger != nil && (successCount > 0 || failCount > 0) {
		h.logger.Info("retry batch completed",
			"success", successCount,
			"failed", failCount,
		)
	}
}

// clearRetry marks a delivery as no longer retryable.
func (h *RetryHandler) clearRetry(ctx context.Context, deliveryID string) error {
	// Get the current delivery
	delivery, err := h.repository.GetDelivery(ctx, deliveryID)
	if err != nil {
		return err
	}

	// Clear the next retry time
	delivery.NextRetryAt = nil

	return h.repository.SaveDelivery(ctx, delivery)
}

// TriggerNow forces an immediate retry check outside the normal schedule.
func (h *RetryHandler) TriggerNow() {
	h.mu.RLock()
	running := h.running
	h.mu.RUnlock()

	if !running {
		return
	}

	go h.processRetries()
}

// IsRunning returns whether the retry handler is currently running.
func (h *RetryHandler) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}
