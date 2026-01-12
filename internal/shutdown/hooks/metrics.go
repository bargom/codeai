package hooks

import (
	"context"

	"github.com/bargom/codeai/internal/shutdown"
)

// MetricsFlusher defines the interface for metrics that can be flushed.
type MetricsFlusher interface {
	Flush() error
}

// LogFlusher defines the interface for a logger that can be flushed.
type LogFlusher interface {
	Sync() error
}

// MetricsShutdownFunc creates a shutdown hook for metrics flushing.
func MetricsShutdownFunc(flusher MetricsFlusher) shutdown.HookFunc {
	return func(ctx context.Context) error {
		return flusher.Flush()
	}
}

// MetricsShutdown creates a shutdown hook for metrics flushing.
func MetricsShutdown(name string, flusher MetricsFlusher) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityMetrics,
		Fn:       MetricsShutdownFunc(flusher),
	}
}

// LoggerShutdownFunc creates a shutdown hook for logger syncing.
func LoggerShutdownFunc(flusher LogFlusher) shutdown.HookFunc {
	return func(ctx context.Context) error {
		return flusher.Sync()
	}
}

// LoggerShutdown creates a shutdown hook for logger syncing.
func LoggerShutdown(name string, flusher LogFlusher) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityMetrics,
		Fn:       LoggerShutdownFunc(flusher),
	}
}

// TelemetryShutdown creates a shutdown hook for combined metrics and logging.
func TelemetryShutdown(metricsFlush func() error, logSync func() error) shutdown.Hook {
	return shutdown.Hook{
		Name:     "telemetry",
		Priority: shutdown.PriorityMetrics,
		Fn: func(ctx context.Context) error {
			var errs []error

			if metricsFlush != nil {
				if err := metricsFlush(); err != nil {
					errs = append(errs, err)
				}
			}

			if logSync != nil {
				if err := logSync(); err != nil {
					errs = append(errs, err)
				}
			}

			if len(errs) > 0 {
				return errs[0] // Return first error
			}
			return nil
		},
	}
}
