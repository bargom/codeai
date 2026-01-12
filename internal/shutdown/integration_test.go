//go:build integration

package shutdown_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/shutdown"
	"github.com/bargom/codeai/internal/shutdown/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_HTTPServerGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a handler that takes 100ms to respond
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTP server with listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{
		Handler: handler,
	}

	serverAddr := ln.Addr().String()
	serverDone := make(chan struct{})

	go func() {
		server.Serve(ln)
		close(serverDone)
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Create shutdown manager
	cfg := shutdown.Config{
		OverallTimeout: 5 * time.Second,
		PerHookTimeout: 3 * time.Second,
		DrainTimeout:   2 * time.Second,
	}
	cfg.Validate()

	manager := shutdown.NewManager(cfg, nil)
	manager.RegisterHook(hooks.HTTPServerShutdown(server, 2*time.Second))

	// Start some in-flight requests
	var wg sync.WaitGroup
	requestResults := make(chan int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://" + serverAddr + "/")
			if err != nil {
				requestResults <- -1
				return
			}
			defer resp.Body.Close()
			requestResults <- resp.StatusCode
		}()
	}

	// Wait a bit for requests to start
	time.Sleep(50 * time.Millisecond)

	// Initiate shutdown
	shutdownDone := make(chan struct{})
	go func() {
		manager.Shutdown()
		close(shutdownDone)
	}()

	// Wait for all requests to complete
	wg.Wait()
	close(requestResults)

	// Wait for shutdown
	<-shutdownDone

	// Verify all in-flight requests completed successfully
	successCount := 0
	for status := range requestResults {
		if status == http.StatusOK {
			successCount++
		}
	}
	assert.Equal(t, 5, successCount, "all in-flight requests should complete during graceful shutdown")

	// Wait for server to fully stop
	<-serverDone
}

func TestIntegration_HTTPDrainerRejectsNewRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a slow handler
	requestStarted := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with drainer
	drainer := shutdown.NewHTTPDrainer(handler)

	// Create and start server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{
		Handler: drainer,
	}
	serverAddr := ln.Addr().String()

	serverDone := make(chan struct{})
	go func() {
		server.Serve(ln)
		close(serverDone)
	}()

	time.Sleep(50 * time.Millisecond)

	// Start an in-flight request
	var wg sync.WaitGroup
	wg.Add(1)
	var inFlightStatus int
	go func() {
		defer wg.Done()
		resp, err := http.Get("http://" + serverAddr + "/")
		if err != nil {
			inFlightStatus = -1
			return
		}
		defer resp.Body.Close()
		inFlightStatus = resp.StatusCode
	}()

	// Wait for request to start
	<-requestStarted

	// Start draining
	drainer.StartDrain()

	// Try a new request (should be rejected)
	resp, err := http.Get("http://" + serverAddr + "/")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "new requests should be rejected during draining")
	}

	// Wait for in-flight request
	wg.Wait()
	assert.Equal(t, http.StatusOK, inFlightStatus, "in-flight request should complete")

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	server.Shutdown(ctx)
	<-serverDone
}

func TestIntegration_ShutdownWithMultipleComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Track execution order
	executionOrder := make([]string, 0, 5)
	orderMu := sync.Mutex{}

	// Create shutdown manager
	cfg := shutdown.DefaultConfig()
	cfg.OverallTimeout = 10 * time.Second
	manager := shutdown.NewManager(cfg, nil)

	// Register hooks with different priorities
	manager.Register("http-server", shutdown.PriorityHTTPServer, func(ctx context.Context) error {
		orderMu.Lock()
		executionOrder = append(executionOrder, "http-server")
		orderMu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	manager.Register("background-worker", shutdown.PriorityBackgroundWorkers, func(ctx context.Context) error {
		orderMu.Lock()
		executionOrder = append(executionOrder, "background-worker")
		orderMu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	manager.Register("database", shutdown.PriorityDatabase, func(ctx context.Context) error {
		orderMu.Lock()
		executionOrder = append(executionOrder, "database")
		orderMu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	manager.Register("cache", shutdown.PriorityCache, func(ctx context.Context) error {
		orderMu.Lock()
		executionOrder = append(executionOrder, "cache")
		orderMu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	manager.Register("metrics", shutdown.PriorityMetrics, func(ctx context.Context) error {
		orderMu.Lock()
		executionOrder = append(executionOrder, "metrics")
		orderMu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	// Execute shutdown
	manager.Shutdown()

	// Verify execution order (higher priority first)
	assert.Equal(t, []string{
		"http-server",       // Priority 90
		"background-worker", // Priority 80
		"database",          // Priority 70
		"cache",             // Priority 60
		"metrics",           // Priority 50
	}, executionOrder)

	// Verify no errors
	assert.Empty(t, manager.Errors())
}

func TestIntegration_ShutdownTimeoutEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := shutdown.Config{
		OverallTimeout: 200 * time.Millisecond,
		PerHookTimeout: 100 * time.Millisecond,
		DrainTimeout:   50 * time.Millisecond,
	}
	cfg.Validate()

	manager := shutdown.NewManager(cfg, nil)

	// Register a hook that takes too long
	manager.Register("slow-hook", 50, func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	start := time.Now()
	manager.Shutdown()
	elapsed := time.Since(start)

	// Should timeout around 100ms (per-hook timeout)
	assert.Less(t, elapsed, 150*time.Millisecond, "should respect per-hook timeout")

	// Should have timeout error
	errs := manager.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "timed out")
}

func TestIntegration_HealthReadyDuringShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create shutdown manager
	manager := shutdown.NewManager(shutdown.DefaultConfig(), nil)

	// Create health handler with shutdown middleware
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	handler := shutdown.ShutdownReadyMiddleware(manager)(healthHandler)

	// Start server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{Handler: handler}
	serverAddr := ln.Addr().String()

	go server.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Before shutdown - should return 200
	resp, err := http.Get("http://" + serverAddr + "/health/ready")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_ConcurrentHookExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	manager := shutdown.NewManager(shutdown.DefaultConfig(), nil)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	// Register multiple hooks at same priority
	for i := 0; i < 5; i++ {
		manager.Register("hook", 50, func(ctx context.Context) error {
			current := concurrent.Add(1)
			if current > maxConcurrent.Load() {
				maxConcurrent.Store(current)
			}
			time.Sleep(100 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		})
	}

	start := time.Now()
	manager.Shutdown()
	elapsed := time.Since(start)

	// All 5 hooks should run concurrently
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(4), "hooks with same priority should run concurrently")

	// Should complete in ~100ms (parallel), not 500ms (sequential)
	assert.Less(t, elapsed, 200*time.Millisecond, "concurrent execution should be faster than sequential")
}

func TestIntegration_DrainerActiveRequestCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	drainer := shutdown.NewHTTPDrainer(handler)

	// Start server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{Handler: drainer}
	serverAddr := ln.Addr().String()

	go server.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Start multiple concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://" + serverAddr + "/")
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}

	// Wait a bit for requests to start
	time.Sleep(50 * time.Millisecond)

	// Check active count
	assert.Equal(t, int64(3), drainer.ActiveRequests())

	// Wait for requests to complete
	wg.Wait()

	// Should be zero after completion
	assert.Equal(t, int64(0), drainer.ActiveRequests())
}

func TestIntegration_PanicRecoveryDuringShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	manager := shutdown.NewManager(shutdown.DefaultConfig(), nil)

	var normalExecuted atomic.Bool

	// Register a hook that panics
	manager.Register("panicking", 90, func(ctx context.Context) error {
		panic("test panic")
	})

	// Register a normal hook at lower priority
	manager.Register("normal", 80, func(ctx context.Context) error {
		normalExecuted.Store(true)
		return nil
	})

	// Shutdown should not panic
	assert.NotPanics(t, func() {
		manager.Shutdown()
	})

	// Normal hook should still execute
	assert.True(t, normalExecuted.Load(), "normal hook should execute even after panic")

	// Should have panic error
	errs := manager.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "panicked")
}

func TestIntegration_ShutdownOnlyOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	manager := shutdown.NewManager(shutdown.DefaultConfig(), nil)

	var executeCount atomic.Int32

	manager.Register("counter", 50, func(ctx context.Context) error {
		executeCount.Add(1)
		return nil
	})

	// Call shutdown from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Shutdown()
		}()
	}

	wg.Wait()

	// Hook should only execute once
	assert.Equal(t, int32(1), executeCount.Load())
}
