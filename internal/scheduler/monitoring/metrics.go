// Package monitoring provides metrics and monitoring for the scheduler.
package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks scheduler metrics.
type Metrics struct {
	// Job counters
	jobsEnqueued   int64
	jobsScheduled  int64
	jobsStarted    int64
	jobsCompleted  int64
	jobsFailed     int64
	jobsCancelled  int64
	jobsRetrying   int64

	// Per-task type metrics
	mu              sync.RWMutex
	taskTypeMetrics map[string]*TaskTypeMetrics

	// Duration tracking
	totalDuration    int64 // in nanoseconds
	durationCount    int64

	// Start time for uptime calculation
	startTime time.Time
}

// TaskTypeMetrics tracks metrics for a specific task type.
type TaskTypeMetrics struct {
	Enqueued      int64
	Completed     int64
	Failed        int64
	TotalDuration int64 // in nanoseconds
	Count         int64
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		taskTypeMetrics: make(map[string]*TaskTypeMetrics),
		startTime:       time.Now(),
	}
}

// RecordJobEnqueued records a job enqueue event.
func (m *Metrics) RecordJobEnqueued(taskType string) {
	atomic.AddInt64(&m.jobsEnqueued, 1)
	m.getOrCreateTaskMetrics(taskType).incrementEnqueued()
}

// RecordJobScheduled records a job schedule event.
func (m *Metrics) RecordJobScheduled(taskType string) {
	atomic.AddInt64(&m.jobsScheduled, 1)
	m.getOrCreateTaskMetrics(taskType).incrementEnqueued()
}

// RecordJobStarted records a job start event.
func (m *Metrics) RecordJobStarted(taskType string) {
	atomic.AddInt64(&m.jobsStarted, 1)
}

// RecordJobCompleted records a job completion event.
func (m *Metrics) RecordJobCompleted(taskType string, duration time.Duration) {
	atomic.AddInt64(&m.jobsCompleted, 1)
	atomic.AddInt64(&m.totalDuration, int64(duration))
	atomic.AddInt64(&m.durationCount, 1)

	tm := m.getOrCreateTaskMetrics(taskType)
	tm.incrementCompleted()
	tm.addDuration(duration)
}

// RecordJobFailed records a job failure event.
func (m *Metrics) RecordJobFailed(taskType string) {
	atomic.AddInt64(&m.jobsFailed, 1)
	m.getOrCreateTaskMetrics(taskType).incrementFailed()
}

// RecordJobCancelled records a job cancellation event.
func (m *Metrics) RecordJobCancelled(taskType string) {
	atomic.AddInt64(&m.jobsCancelled, 1)
}

// RecordJobRetrying records a job retry event.
func (m *Metrics) RecordJobRetrying(taskType string) {
	atomic.AddInt64(&m.jobsRetrying, 1)
}

// GetStats returns the current metrics as a Stats struct.
func (m *Metrics) GetStats() Stats {
	enqueued := atomic.LoadInt64(&m.jobsEnqueued)
	scheduled := atomic.LoadInt64(&m.jobsScheduled)
	started := atomic.LoadInt64(&m.jobsStarted)
	completed := atomic.LoadInt64(&m.jobsCompleted)
	failed := atomic.LoadInt64(&m.jobsFailed)
	cancelled := atomic.LoadInt64(&m.jobsCancelled)
	retrying := atomic.LoadInt64(&m.jobsRetrying)
	totalDuration := atomic.LoadInt64(&m.totalDuration)
	durationCount := atomic.LoadInt64(&m.durationCount)

	var avgDuration time.Duration
	if durationCount > 0 {
		avgDuration = time.Duration(totalDuration / durationCount)
	}

	// Calculate success rate
	var successRate float64
	totalProcessed := completed + failed
	if totalProcessed > 0 {
		successRate = float64(completed) / float64(totalProcessed) * 100
	}

	return Stats{
		JobsEnqueued:    enqueued,
		JobsScheduled:   scheduled,
		JobsStarted:     started,
		JobsCompleted:   completed,
		JobsFailed:      failed,
		JobsCancelled:   cancelled,
		JobsRetrying:    retrying,
		TotalProcessed:  totalProcessed,
		SuccessRate:     successRate,
		AvgDuration:     avgDuration,
		Uptime:          time.Since(m.startTime),
		TaskTypeStats:   m.getTaskTypeStats(),
	}
}

// getOrCreateTaskMetrics gets or creates metrics for a task type.
func (m *Metrics) getOrCreateTaskMetrics(taskType string) *TaskTypeMetrics {
	m.mu.RLock()
	tm, ok := m.taskTypeMetrics[taskType]
	m.mu.RUnlock()

	if ok {
		return tm
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if tm, ok = m.taskTypeMetrics[taskType]; ok {
		return tm
	}

	tm = &TaskTypeMetrics{}
	m.taskTypeMetrics[taskType] = tm
	return tm
}

// getTaskTypeStats returns stats for all task types.
func (m *Metrics) getTaskTypeStats() map[string]TaskTypeStat {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]TaskTypeStat, len(m.taskTypeMetrics))
	for taskType, tm := range m.taskTypeMetrics {
		enqueued := atomic.LoadInt64(&tm.Enqueued)
		completed := atomic.LoadInt64(&tm.Completed)
		failed := atomic.LoadInt64(&tm.Failed)
		totalDuration := atomic.LoadInt64(&tm.TotalDuration)
		count := atomic.LoadInt64(&tm.Count)

		var avgDuration time.Duration
		if count > 0 {
			avgDuration = time.Duration(totalDuration / count)
		}

		var successRate float64
		total := completed + failed
		if total > 0 {
			successRate = float64(completed) / float64(total) * 100
		}

		stats[taskType] = TaskTypeStat{
			Enqueued:    enqueued,
			Completed:   completed,
			Failed:      failed,
			SuccessRate: successRate,
			AvgDuration: avgDuration,
		}
	}
	return stats
}

// incrementEnqueued increments the enqueued counter.
func (tm *TaskTypeMetrics) incrementEnqueued() {
	atomic.AddInt64(&tm.Enqueued, 1)
}

// incrementCompleted increments the completed counter.
func (tm *TaskTypeMetrics) incrementCompleted() {
	atomic.AddInt64(&tm.Completed, 1)
}

// incrementFailed increments the failed counter.
func (tm *TaskTypeMetrics) incrementFailed() {
	atomic.AddInt64(&tm.Failed, 1)
}

// addDuration adds a duration to the total.
func (tm *TaskTypeMetrics) addDuration(d time.Duration) {
	atomic.AddInt64(&tm.TotalDuration, int64(d))
	atomic.AddInt64(&tm.Count, 1)
}

// Stats represents scheduler statistics.
type Stats struct {
	JobsEnqueued   int64                    `json:"jobs_enqueued"`
	JobsScheduled  int64                    `json:"jobs_scheduled"`
	JobsStarted    int64                    `json:"jobs_started"`
	JobsCompleted  int64                    `json:"jobs_completed"`
	JobsFailed     int64                    `json:"jobs_failed"`
	JobsCancelled  int64                    `json:"jobs_cancelled"`
	JobsRetrying   int64                    `json:"jobs_retrying"`
	TotalProcessed int64                    `json:"total_processed"`
	SuccessRate    float64                  `json:"success_rate"`
	AvgDuration    time.Duration            `json:"avg_duration"`
	Uptime         time.Duration            `json:"uptime"`
	TaskTypeStats  map[string]TaskTypeStat  `json:"task_type_stats"`
}

// TaskTypeStat represents statistics for a specific task type.
type TaskTypeStat struct {
	Enqueued    int64         `json:"enqueued"`
	Completed   int64         `json:"completed"`
	Failed      int64         `json:"failed"`
	SuccessRate float64       `json:"success_rate"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// Reset resets all metrics to zero.
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.jobsEnqueued, 0)
	atomic.StoreInt64(&m.jobsScheduled, 0)
	atomic.StoreInt64(&m.jobsStarted, 0)
	atomic.StoreInt64(&m.jobsCompleted, 0)
	atomic.StoreInt64(&m.jobsFailed, 0)
	atomic.StoreInt64(&m.jobsCancelled, 0)
	atomic.StoreInt64(&m.jobsRetrying, 0)
	atomic.StoreInt64(&m.totalDuration, 0)
	atomic.StoreInt64(&m.durationCount, 0)

	m.mu.Lock()
	m.taskTypeMetrics = make(map[string]*TaskTypeMetrics)
	m.startTime = time.Now()
	m.mu.Unlock()
}
