package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	assert.NotNil(t, m)

	stats := m.GetStats()
	assert.Equal(t, int64(0), stats.JobsEnqueued)
	assert.Equal(t, int64(0), stats.JobsCompleted)
	assert.Equal(t, int64(0), stats.JobsFailed)
}

func TestMetrics_RecordJobEnqueued(t *testing.T) {
	m := NewMetrics()

	m.RecordJobEnqueued("task:a")
	m.RecordJobEnqueued("task:a")
	m.RecordJobEnqueued("task:b")

	stats := m.GetStats()
	assert.Equal(t, int64(3), stats.JobsEnqueued)

	// Check per-task type stats
	assert.Equal(t, int64(2), stats.TaskTypeStats["task:a"].Enqueued)
	assert.Equal(t, int64(1), stats.TaskTypeStats["task:b"].Enqueued)
}

func TestMetrics_RecordJobCompleted(t *testing.T) {
	m := NewMetrics()

	m.RecordJobCompleted("task:a", 100*time.Millisecond)
	m.RecordJobCompleted("task:a", 200*time.Millisecond)
	m.RecordJobCompleted("task:b", 300*time.Millisecond)

	stats := m.GetStats()
	assert.Equal(t, int64(3), stats.JobsCompleted)
	assert.Equal(t, 200*time.Millisecond, stats.AvgDuration)

	// Check per-task type stats
	assert.Equal(t, int64(2), stats.TaskTypeStats["task:a"].Completed)
	assert.Equal(t, int64(1), stats.TaskTypeStats["task:b"].Completed)
}

func TestMetrics_RecordJobFailed(t *testing.T) {
	m := NewMetrics()

	m.RecordJobFailed("task:a")
	m.RecordJobFailed("task:a")
	m.RecordJobFailed("task:b")

	stats := m.GetStats()
	assert.Equal(t, int64(3), stats.JobsFailed)

	// Check per-task type stats
	assert.Equal(t, int64(2), stats.TaskTypeStats["task:a"].Failed)
	assert.Equal(t, int64(1), stats.TaskTypeStats["task:b"].Failed)
}

func TestMetrics_SuccessRate(t *testing.T) {
	m := NewMetrics()

	// 8 completed, 2 failed = 80% success rate
	for i := 0; i < 8; i++ {
		m.RecordJobCompleted("task:a", 100*time.Millisecond)
	}
	for i := 0; i < 2; i++ {
		m.RecordJobFailed("task:a")
	}

	stats := m.GetStats()
	assert.Equal(t, int64(10), stats.TotalProcessed)
	assert.InDelta(t, 80.0, stats.SuccessRate, 0.01)
}

func TestMetrics_PerTaskTypeSuccessRate(t *testing.T) {
	m := NewMetrics()

	// task:a - 3 completed, 1 failed = 75%
	for i := 0; i < 3; i++ {
		m.RecordJobCompleted("task:a", 100*time.Millisecond)
	}
	m.RecordJobFailed("task:a")

	// task:b - 1 completed, 3 failed = 25%
	m.RecordJobCompleted("task:b", 100*time.Millisecond)
	for i := 0; i < 3; i++ {
		m.RecordJobFailed("task:b")
	}

	stats := m.GetStats()
	assert.InDelta(t, 75.0, stats.TaskTypeStats["task:a"].SuccessRate, 0.01)
	assert.InDelta(t, 25.0, stats.TaskTypeStats["task:b"].SuccessRate, 0.01)
}

func TestMetrics_RecordJobCancelled(t *testing.T) {
	m := NewMetrics()

	m.RecordJobCancelled("task:a")
	m.RecordJobCancelled("task:b")

	stats := m.GetStats()
	assert.Equal(t, int64(2), stats.JobsCancelled)
}

func TestMetrics_RecordJobRetrying(t *testing.T) {
	m := NewMetrics()

	m.RecordJobRetrying("task:a")
	m.RecordJobRetrying("task:a")

	stats := m.GetStats()
	assert.Equal(t, int64(2), stats.JobsRetrying)
}

func TestMetrics_Reset(t *testing.T) {
	m := NewMetrics()

	m.RecordJobEnqueued("task:a")
	m.RecordJobCompleted("task:a", 100*time.Millisecond)
	m.RecordJobFailed("task:a")

	// Verify data exists
	stats := m.GetStats()
	assert.Equal(t, int64(1), stats.JobsEnqueued)
	assert.Equal(t, int64(1), stats.JobsCompleted)

	// Reset
	m.Reset()

	// Verify data is cleared
	stats = m.GetStats()
	assert.Equal(t, int64(0), stats.JobsEnqueued)
	assert.Equal(t, int64(0), stats.JobsCompleted)
	assert.Equal(t, int64(0), stats.JobsFailed)
	assert.Empty(t, stats.TaskTypeStats)
}

func TestMetrics_Uptime(t *testing.T) {
	m := NewMetrics()

	// Wait a tiny bit
	time.Sleep(10 * time.Millisecond)

	stats := m.GetStats()
	assert.True(t, stats.Uptime >= 10*time.Millisecond)
}
