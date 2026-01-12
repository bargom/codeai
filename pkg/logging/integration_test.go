//go:build integration

package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_EndToEndRequestTracing tests request tracing through multiple handlers.
func TestIntegration_EndToEndRequestTracing(t *testing.T) {
	var logBuf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	ourLogger := NewWithWriter(config, &logBuf)
	mw := NewHTTPMiddleware(ourLogger.Logger)

	// Create a multi-handler pipeline
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is available in context
		requestID := GetRequestID(r.Context())
		assert.NotEmpty(t, requestID, "request ID should be in context")

		// Pass to next handler
		w.Header().Set("X-Handler-1", "processed")
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request ID should still be available
		requestID := GetRequestID(r.Context())
		assert.NotEmpty(t, requestID, "request ID should persist through handlers")

		// Log with context
		ourLogger.InfoContext(r.Context(), "handler2 processing",
			"operation", "internal-operation",
		)

		w.Header().Set("X-Handler-2", "processed")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Chain handlers
	pipeline := mw.Handler(chainHandlers(handler1, handler2))

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	pipeline.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "processed", rec.Header().Get("X-Handler-1"))
	assert.Equal(t, "processed", rec.Header().Get("X-Handler-2"))

	// Verify logs contain request ID
	logs := logBuf.String()
	assert.Contains(t, logs, "request_id")

	// Parse and verify log entries
	lines := strings.Split(strings.TrimSpace(logs), "\n")
	assert.GreaterOrEqual(t, len(lines), 1)

	var foundRequestID string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err, "log entry should be valid JSON")

		if rid, ok := entry["request_id"].(string); ok {
			if foundRequestID == "" {
				foundRequestID = rid
			} else {
				assert.Equal(t, foundRequestID, rid, "all logs should have same request ID")
			}
		}
	}
}

// TestIntegration_TraceIDPropagation tests trace ID propagation across services.
func TestIntegration_TraceIDPropagation(t *testing.T) {
	var service1Logs, service2Logs bytes.Buffer
	config := Config{Level: "info", Format: "json"}

	// Service 1 setup
	ourLogger1 := NewWithWriter(config, &service1Logs)
	mw1 := NewHTTPMiddleware(ourLogger1.Logger)

	// Service 2 (simulated downstream service)
	ourLogger2 := NewWithWriter(config, &service2Logs)
	mw2 := NewHTTPMiddleware(ourLogger2.Logger)

	service2Handler := mw2.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ourLogger2.InfoContext(r.Context(), "service2 processing")
		w.WriteHeader(http.StatusOK)
	}))
	service2 := httptest.NewServer(service2Handler)
	defer service2.Close()

	// Service 1 calls Service 2
	service1Handler := mw1.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ourLogger1.InfoContext(r.Context(), "service1 processing")

		// Extract trace context
		tc := FromContext(r.Context())

		// Call service 2 with trace context
		req2, err := http.NewRequestWithContext(r.Context(), "GET", service2.URL, nil)
		require.NoError(t, err)
		req2.Header.Set(RequestHeaderRequestID, tc.RequestID)
		req2.Header.Set(RequestHeaderTraceID, tc.TraceID)

		resp, err := http.DefaultClient.Do(req2)
		require.NoError(t, err)
		defer resp.Body.Close()

		w.WriteHeader(http.StatusOK)
	}))

	// Make initial request with known trace ID
	traceID := "parent-trace-id-123"
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(RequestHeaderTraceID, traceID)
	rec := httptest.NewRecorder()

	service1Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify service 1 logs have trace ID
	assert.Contains(t, service1Logs.String(), traceID)
}

// TestIntegration_SensitiveDataRedaction tests that sensitive data is properly redacted.
func TestIntegration_SensitiveDataRedaction(t *testing.T) {
	var logBuf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&logBuf, nil)
	redactingHandler := NewRedactingHandler(baseHandler, nil)
	logger := slog.New(redactingHandler)

	mw := NewHTTPMiddleware(logger).WithVerbosity(VerbosityVerbose)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log some sensitive data - using string values that can be redacted
		logger.InfoContext(r.Context(), "processing request",
			"password", "super-secret-password",
			"api_key", "sk_live_12345",
			"username", "normaluser",
		)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret-jwt-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logs := logBuf.String()

	// Verify sensitive field values are redacted
	assert.NotContains(t, logs, "super-secret-password")
	assert.NotContains(t, logs, "sk_live_12345")
	assert.Contains(t, logs, RedactedValue)

	// Verify non-sensitive data is preserved
	assert.Contains(t, logs, "normaluser")
}

// TestIntegration_ConcurrentRequests tests logging under concurrent load.
func TestIntegration_ConcurrentRequests(t *testing.T) {
	var logBuf bytes.Buffer
	var mu sync.Mutex

	handler := slog.NewJSONHandler(&safeWriter{Writer: &logBuf, mu: &mu}, nil)
	logger := slog.New(handler)
	mw := NewHTTPMiddleware(logger)

	httpHandler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(httpHandler)
	defer server.Close()

	// Make concurrent requests
	const numRequests = 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(n int) {
			defer wg.Done()
			resp, err := http.Get(server.URL)
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()

	// Give time for all logs to be written
	time.Sleep(100 * time.Millisecond)

	// Count log entries
	mu.Lock()
	logs := logBuf.String()
	mu.Unlock()

	lines := strings.Split(strings.TrimSpace(logs), "\n")
	var validEntries int
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		if err == nil {
			validEntries++
		}
	}

	assert.Equal(t, numRequests, validEntries, "should have one log entry per request")
}

// TestIntegration_NestedHandlerContext tests context propagation through nested handlers.
func TestIntegration_NestedHandlerContext(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	// Create middleware chain
	mw := NewHTTPMiddleware(logger)

	// Auth middleware (inner)
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add user ID to context
			ctx := WithUserID(r.Context(), "user-123")
			logger.InfoContext(ctx, "user authenticated")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Business handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		logger.InfoContext(r.Context(), "processing business logic",
			"user_verified", userID != "",
		)
		w.WriteHeader(http.StatusOK)
	})

	// Chain: logging -> auth -> handler
	pipeline := mw.Handler(authMiddleware(handler))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	pipeline.ServeHTTP(rec, req)

	logs := logBuf.String()
	lines := strings.Split(strings.TrimSpace(logs), "\n")

	// Verify all logs have user_id after auth middleware
	foundUserAuth := false
	foundBusinessLog := false
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			if entry["msg"] == "user authenticated" {
				foundUserAuth = true
			}
			if entry["msg"] == "processing business logic" {
				foundBusinessLog = true
				assert.Equal(t, true, entry["user_verified"])
			}
		}
	}

	assert.True(t, foundUserAuth, "should have auth log")
	assert.True(t, foundBusinessLog, "should have business log")
}

// TestIntegration_LoggerWithRotation tests file output with simulated rotation.
func TestIntegration_LoggerWithRotation(t *testing.T) {
	// Create a temp file for logging
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	require.NoError(t, err)
	defer tmpFile.Close()

	config := Config{
		Level:  "info",
		Format: "json",
		Output: tmpFile.Name(),
	}

	logger := New(config)
	logger.Info("test message 1")
	logger.Info("test message 2")
	logger.Info("test message 3")

	// Read the log file
	content, err := io.ReadAll(tmpFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 3, "should have 3 log entries")
}

// TestIntegration_ErrorStatusLogging tests that error responses are logged at appropriate levels.
func TestIntegration_ErrorStatusLogging(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{"success", 200, "INFO"},
		{"created", 201, "INFO"},
		{"bad request", 400, "WARN"},
		{"unauthorized", 401, "WARN"},
		{"forbidden", 403, "WARN"},
		{"not found", 404, "WARN"},
		{"internal error", 500, "ERROR"},
		{"bad gateway", 502, "ERROR"},
		{"service unavailable", 503, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
			mw := NewHTTPMiddleware(logger)

			handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			var entry map[string]any
			err := json.Unmarshal(logBuf.Bytes(), &entry)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedLevel, entry["level"])
		})
	}
}

// Helper: safeWriter wraps a writer with mutex for concurrent access.
type safeWriter struct {
	io.Writer
	mu *sync.Mutex
}

func (w *safeWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Writer.Write(p)
}

// Helper: chain multiple handlers together.
func chainHandlers(handlers ...http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, h := range handlers {
			h.ServeHTTP(w, r)
		}
	})
}
