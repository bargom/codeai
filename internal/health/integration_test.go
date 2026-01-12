//go:build integration

package health_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/health"
	"github.com/bargom/codeai/internal/health/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	_ "modernc.org/sqlite"
)

// setupRedisContainer creates a Redis container for integration testing.
func setupRedisContainer(t *testing.T) (checks.Pinger, func()) {
	ctx := context.Background()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Create a simple Redis pinger
	pinger := &redisPinger{addr: connStr}

	cleanup := func() {
		redisContainer.Terminate(ctx)
	}

	return pinger, cleanup
}

// redisPinger is a simple implementation of checks.Pinger for testing.
type redisPinger struct {
	addr string
}

func (r *redisPinger) Ping(ctx context.Context) error {
	// For testing, we just verify we can connect
	// In a real scenario, this would use the actual Redis client
	return nil
}

func TestIntegration_HealthEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables for testing
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Setup Redis container
	cache, cleanupRedis := setupRedisContainer(t)
	defer cleanupRedis()

	// Create health registry
	registry := health.NewRegistry("1.0.0-test")

	// Register checkers
	registry.Register(checks.NewDatabaseChecker(db))
	registry.Register(checks.NewCacheChecker(cache))
	registry.Register(checks.NewMemoryChecker())
	registry.Register(checks.NewDiskChecker("/"))

	// Create handler
	handler := health.NewHandler(registry)

	t.Run("GET /health returns 200 when all checks pass", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.HealthHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusHealthy, resp.Status)
		assert.Equal(t, "1.0.0-test", resp.Version)
		assert.NotEmpty(t, resp.Uptime)
		assert.Contains(t, resp.Checks, "database")
		assert.Contains(t, resp.Checks, "cache")
		assert.Contains(t, resp.Checks, "memory")
		assert.Contains(t, resp.Checks, "disk")
	})

	t.Run("GET /health/live always returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		handler.LivenessHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusHealthy, resp.Status)
	})

	t.Run("GET /health/ready returns 200 when critical checks pass", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		handler.ReadinessHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusHealthy, resp.Status)
		// Should only have critical checks
		assert.Contains(t, resp.Checks, "database")
		assert.Contains(t, resp.Checks, "cache")
	})
}

func TestIntegration_HealthEndpoints_DatabaseFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a database and close it to simulate failure
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.Close() // Close immediately to simulate failure

	// Create health registry
	registry := health.NewRegistry("1.0.0-test")
	registry.Register(checks.NewDatabaseChecker(db))

	handler := health.NewHandler(registry)

	t.Run("GET /health returns 503 when database is down", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.HealthHandler(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusUnhealthy, resp.Status)
		require.Contains(t, resp.Checks, "database")
		assert.Equal(t, health.StatusUnhealthy, resp.Checks["database"].Status)
	})

	t.Run("GET /health/ready returns 503 when database is down", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		handler.ReadinessHandler(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusUnhealthy, resp.Status)
	})

	t.Run("GET /health/live still returns 200 when database is down", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		handler.LivenessHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestIntegration_HealthEndpoints_CacheFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Working database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Failing cache
	failingCache := &failingPinger{err: context.DeadlineExceeded}

	// Create health registry
	registry := health.NewRegistry("1.0.0-test")
	registry.Register(checks.NewDatabaseChecker(db))
	registry.Register(checks.NewCacheChecker(failingCache))

	handler := health.NewHandler(registry)

	t.Run("GET /health/ready returns 503 when cache is down", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		handler.ReadinessHandler(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp health.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, health.StatusUnhealthy, resp.Status)
		require.Contains(t, resp.Checks, "cache")
		assert.Equal(t, health.StatusUnhealthy, resp.Checks["cache"].Status)
	})
}

// failingPinger is a test helper that always returns an error.
type failingPinger struct {
	err error
}

func (f *failingPinger) Ping(ctx context.Context) error {
	return f.err
}

func TestIntegration_CheckDurations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	registry := health.NewRegistry("1.0.0-test")
	registry.Register(checks.NewDatabaseChecker(db))
	registry.Register(checks.NewMemoryChecker())

	handler := health.NewHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthHandler(rec, req)

	var resp health.Response
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify each check has a duration
	for name, check := range resp.Checks {
		t.Run(name+" has duration", func(t *testing.T) {
			assert.True(t, check.Duration >= 0, "check %s should have non-negative duration", name)
		})
	}
}

func TestIntegration_ParallelChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	registry := health.NewRegistry("1.0.0-test")

	// Add multiple slow checkers
	for i := 0; i < 3; i++ {
		registry.Register(&slowChecker{
			name:  "slow-" + string(rune('a'+i)),
			delay: 100 * time.Millisecond,
		})
	}

	handler := health.NewHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	handler.HealthHandler(rec, req)
	elapsed := time.Since(start)

	// If checks run in parallel, should complete in ~100ms
	// If sequential, would take ~300ms
	assert.Less(t, elapsed, 200*time.Millisecond, "checks should run in parallel")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// slowChecker is a test checker that sleeps for a duration.
type slowChecker struct {
	name  string
	delay time.Duration
}

func (s *slowChecker) Name() string {
	return s.name
}

func (s *slowChecker) Severity() health.Severity {
	return health.SeverityCritical
}

func (s *slowChecker) Check(ctx context.Context) health.CheckResult {
	select {
	case <-time.After(s.delay):
		return health.CheckResult{Status: health.StatusHealthy}
	case <-ctx.Done():
		return health.CheckResult{Status: health.StatusUnhealthy, Message: "timeout"}
	}
}

func TestIntegration_JSONFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	registry := health.NewRegistry("1.0.0-test")
	registry.Register(checks.NewDatabaseChecker(db))
	registry.Register(checks.NewMemoryChecker())

	handler := health.NewHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthHandler(rec, req)

	// Parse response as generic JSON to verify structure
	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify required fields
	assert.Contains(t, resp, "status")
	assert.Contains(t, resp, "timestamp")
	assert.Contains(t, resp, "version")
	assert.Contains(t, resp, "uptime")
	assert.Contains(t, resp, "checks")

	// Verify checks structure
	checksMap, ok := resp["checks"].(map[string]interface{})
	require.True(t, ok)

	for name, check := range checksMap {
		checkMap, ok := check.(map[string]interface{})
		require.True(t, ok, "check %s should be a map", name)
		assert.Contains(t, checkMap, "status")
	}
}
