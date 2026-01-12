package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware_BasicLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/api/test", logEntry["path"])
	assert.Equal(t, float64(200), logEntry["status"])
	assert.Contains(t, logEntry, "duration")
}

func TestHTTPMiddleware_RequestIDGeneration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID := GetRequestID(r.Context())
		assert.NotEmpty(t, requestID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify request ID is in response header
	requestID := rec.Header().Get(RequestHeaderRequestID)
	assert.NotEmpty(t, requestID)
	assert.Len(t, requestID, 36) // UUID format
}

func TestHTTPMiddleware_PreserveExistingRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	existingID := "existing-request-id-123"

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		assert.Equal(t, existingID, requestID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(RequestHeaderRequestID, existingID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, existingID, rec.Header().Get(RequestHeaderRequestID))
}

func TestHTTPMiddleware_TraceIDPropagation(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	traceID := "parent-trace-id-456"

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTraceID := GetTraceID(r.Context())
		assert.Equal(t, traceID, ctxTraceID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(RequestHeaderTraceID, traceID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestHTTPMiddleware_StatusCodeLogging(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{"success", 200, "INFO"},
		{"redirect", 302, "INFO"},
		{"client error", 400, "WARN"},
		{"not found", 404, "WARN"},
		{"server error", 500, "ERROR"},
		{"service unavailable", 503, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			mw := NewHTTPMiddleware(logger)

			handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			var logEntry map[string]any
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedLevel, logEntry["level"])
			assert.Equal(t, float64(tt.statusCode), logEntry["status"])
		})
	}
}

func TestHTTPMiddleware_VerbosityLevels(t *testing.T) {
	tests := []struct {
		name       string
		verbosity  Verbosity
		checkField string
		exists     bool
	}{
		{"minimal has method", VerbosityMinimal, "method", true},
		{"minimal has status", VerbosityMinimal, "status", true},
		{"minimal no user_agent", VerbosityMinimal, "user_agent", false},
		{"standard has user_agent", VerbosityStandard, "user_agent", true},
		{"standard has remote_addr", VerbosityStandard, "remote_addr", true},
		{"standard no request_headers", VerbosityStandard, "request_headers", false},
		{"verbose has request_headers", VerbosityVerbose, "request_headers", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			mw := NewHTTPMiddleware(logger).WithVerbosity(tt.verbosity)

			handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test?query=value", nil)
			req.Header.Set("User-Agent", "test-agent")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			var logEntry map[string]any
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)

			_, exists := logEntry[tt.checkField]
			assert.Equal(t, tt.exists, exists, "field %s existence mismatch", tt.checkField)
		})
	}
}

func TestHTTPMiddleware_QueryStringLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/search?q=test&page=1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "q=test&page=1", logEntry["query"])
}

func TestHTTPMiddleware_ResponseBytesLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	responseBody := "Hello, World!"

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, float64(len(responseBody)), logEntry["response_bytes"])
}

func TestHTTPMiddleware_SensitiveHeaderRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger).WithVerbosity(VerbosityVerbose)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Api-Key", "secret-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	headers, ok := logEntry["request_headers"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, RedactedValue, headers["Authorization"])
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func TestInjectRequestID(t *testing.T) {
	handler := InjectRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		assert.NotEmpty(t, requestID)
		w.Write([]byte(requestID))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	responseID := rec.Body.String()
	headerID := rec.Header().Get(RequestHeaderRequestID)

	assert.Equal(t, responseID, headerID)
	assert.Len(t, responseID, 36)
}

func TestInjectRequestID_PreserveExisting(t *testing.T) {
	existingID := "existing-id-123"

	handler := InjectRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		assert.Equal(t, existingID, requestID)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(RequestHeaderRequestID, existingID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, existingID, rec.Header().Get(RequestHeaderRequestID))
}

func TestLoggerFromContext(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buf, nil))

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")
	ctx = WithUserID(ctx, "user-789")

	logger := LoggerFromContext(ctx, baseLogger)
	logger.Info("test message")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-123", logEntry["request_id"])
	assert.Equal(t, "trace-456", logEntry["trace_id"])
	assert.Equal(t, "user-789", logEntry["user_id"])
}

func TestLoggerFromContext_NilLogger(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	logger := LoggerFromContext(ctx, nil)
	assert.NotNil(t, logger)
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("part1"))
		w.Write([]byte("part2"))
		w.Write([]byte("part3"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, float64(15), logEntry["response_bytes"]) // 5 + 5 + 5
}

func TestResponseWriter_StatusBeforeWrite(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	}))

	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, float64(201), logEntry["status"])
}

func TestResponseWriter_ImplicitOK(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mw := NewHTTPMiddleware(logger)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No explicit status, just write body
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, float64(200), logEntry["status"])
}

func TestRequestLogger_FunctionHelper(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	middleware := RequestLogger(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, strings.Contains(buf.String(), "http request"))
}
