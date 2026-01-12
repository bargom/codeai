package shutdown

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalHandler manages OS signal handling for graceful shutdown.
type SignalHandler struct {
	sigChan  chan os.Signal
	doneChan chan struct{}
	signals  []os.Signal
}

// NewSignalHandler creates a new signal handler that listens for the specified signals.
// If no signals are provided, it defaults to SIGTERM, SIGINT, and SIGQUIT.
func NewSignalHandler(signals ...os.Signal) *SignalHandler {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}
	}

	return &SignalHandler{
		sigChan:  make(chan os.Signal, 1),
		doneChan: make(chan struct{}),
		signals:  signals,
	}
}

// Listen starts listening for shutdown signals.
// It returns a channel that will receive the signal when one is caught.
// The signal channel will only deliver one signal before being closed.
func (h *SignalHandler) Listen() <-chan os.Signal {
	signal.Notify(h.sigChan, h.signals...)

	outChan := make(chan os.Signal, 1)

	go func() {
		select {
		case sig := <-h.sigChan:
			outChan <- sig
			close(outChan)
		case <-h.doneChan:
			close(outChan)
		}
	}()

	return outChan
}

// Stop stops listening for signals and cleans up.
func (h *SignalHandler) Stop() {
	signal.Stop(h.sigChan)
	close(h.doneChan)
}

// Signals returns the list of signals being monitored.
func (h *SignalHandler) Signals() []os.Signal {
	result := make([]os.Signal, len(h.signals))
	copy(result, h.signals)
	return result
}

// DefaultShutdownSignals returns the default signals used for graceful shutdown.
func DefaultShutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}
}
