package hooks

import (
	"context"
	"time"

	"github.com/bargom/codeai/internal/shutdown"
)

// BackgroundWorker defines the interface for a background worker that can be shut down.
type BackgroundWorker interface {
	// Shutdown signals the worker to stop accepting new jobs.
	Shutdown()
	// WaitForCompletion waits for all current jobs to complete or timeout.
	WaitForCompletion(timeout time.Duration)
}

// BackgroundWorkerShutdownFunc creates a shutdown hook for a background worker.
func BackgroundWorkerShutdownFunc(worker BackgroundWorker, waitTimeout time.Duration) shutdown.HookFunc {
	return func(ctx context.Context) error {
		// Stop accepting new jobs
		worker.Shutdown()

		// Wait for running jobs to complete
		done := make(chan struct{})
		go func() {
			worker.WaitForCompletion(waitTimeout)
			close(done)
		}()

		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// BackgroundWorkerShutdown creates a shutdown hook for a background worker.
func BackgroundWorkerShutdown(name string, worker BackgroundWorker, waitTimeout time.Duration) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityBackgroundWorkers,
		Fn:       BackgroundWorkerShutdownFunc(worker, waitTimeout),
	}
}

// JobScheduler defines the interface for a job scheduler that can be shut down.
type JobScheduler interface {
	// Shutdown stops the scheduler from starting new jobs.
	Shutdown()
	// WaitForJobs waits for all running jobs to complete.
	WaitForJobs(ctx context.Context) error
}

// JobSchedulerShutdownFunc creates a shutdown hook for a job scheduler.
func JobSchedulerShutdownFunc(scheduler JobScheduler) shutdown.HookFunc {
	return func(ctx context.Context) error {
		// Stop accepting new jobs
		scheduler.Shutdown()

		// Wait for running jobs
		return scheduler.WaitForJobs(ctx)
	}
}

// JobSchedulerShutdown creates a shutdown hook for a job scheduler.
func JobSchedulerShutdown(name string, scheduler JobScheduler) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityBackgroundWorkers,
		Fn:       JobSchedulerShutdownFunc(scheduler),
	}
}

// WorkerPoolShutdown creates a shutdown hook that cancels a context and waits for a done channel.
func WorkerPoolShutdown(name string, cancel context.CancelFunc, done <-chan struct{}) shutdown.Hook {
	return shutdown.Hook{
		Name:     name,
		Priority: shutdown.PriorityBackgroundWorkers,
		Fn: func(ctx context.Context) error {
			// Signal workers to stop
			cancel()

			// Wait for workers to finish
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}
