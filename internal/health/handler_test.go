package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	t.Run("returns healthy status", func(t *testing.T) {
		r := NewRegistry("1.0.0")
		h := NewHandler(r)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.HealthHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, resp.Status)
		assert.Equal(t, "1.0.0", resp.Version)
	})

	t.Run("returns unhealthy status with 503", func(t *testing.T) {
		r := NewRegistry("1.0.0")
		r.Register(&mockChecker{
			name:     "failing",
			severity: SeverityCritical,
			result:   CheckResult{Status: StatusUnhealthy, Message: "db down"},
		})
		h := NewHandler(r)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.HealthHandler(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusUnhealthy, resp.Status)
	})
}

func TestLivenessHandler(t *testing.T) {
	t.Run("always returns 200", func(t *testing.T) {
		r := NewRegistry("1.0.0")
		// Even with a failing checker, liveness should return 200
		r.Register(&mockChecker{
			name:     "failing",
			severity: SeverityCritical,
			result:   CheckResult{Status: StatusUnhealthy},
		})
		h := NewHandler(r)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		h.LivenessHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, resp.Status)
	})
}

func TestReadinessHandler(t *testing.T) {
	t.Run("returns 200 when ready", func(t *testing.T) {
		r := NewRegistry("1.0.0")
		r.Register(&mockChecker{
			name:     "db",
			severity: SeverityCritical,
			result:   CheckResult{Status: StatusHealthy},
		})
		h := NewHandler(r)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		h.ReadinessHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("returns 503 when not ready", func(t *testing.T) {
		r := NewRegistry("1.0.0")
		r.Register(&mockChecker{
			name:     "db",
			severity: SeverityCritical,
			result:   CheckResult{Status: StatusUnhealthy, Message: "connection refused"},
		})
		h := NewHandler(r)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		h.ReadinessHandler(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}

func TestRegisterRoutes(t *testing.T) {
	r := NewRegistry("1.0.0")
	h := NewHandler(r)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/health", http.StatusOK},
		{"/health/live", http.StatusOK},
		{"/health/ready", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// mockChecker for handler tests
type mockHandlerChecker struct {
	name     string
	severity Severity
	result   CheckResult
}

func (m *mockHandlerChecker) Name() string {
	return m.name
}

func (m *mockHandlerChecker) Severity() Severity {
	return m.severity
}

func (m *mockHandlerChecker) Check(ctx context.Context) CheckResult {
	return m.result
}
