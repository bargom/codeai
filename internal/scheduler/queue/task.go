package queue

import (
	"encoding/json"
	"time"
)

// Task represents a task to be enqueued.
type Task struct {
	// Type is the task type identifier.
	Type string

	// Payload is the task payload data.
	Payload json.RawMessage

	// Queue is the queue name (defaults to "default").
	Queue string

	// MaxRetry is the maximum number of retries (defaults to 3).
	MaxRetry int

	// Timeout is the task execution timeout.
	Timeout time.Duration

	// Deadline is the absolute time by which the task must be processed.
	Deadline time.Time

	// Retention is how long to keep the completed task.
	Retention time.Duration

	// UniqueKey prevents duplicate tasks with the same key.
	UniqueKey string

	// UniqueTTL is how long to enforce uniqueness.
	UniqueTTL time.Duration

	// Group is used for task grouping.
	Group string
}

// NewTask creates a new task with the given type and payload.
func NewTask(taskType string, payload any) (*Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Task{
		Type:     taskType,
		Payload:  data,
		Queue:    QueueDefault,
		MaxRetry: 3,
	}, nil
}

// WithQueue sets the queue for the task.
func (t *Task) WithQueue(queue string) *Task {
	t.Queue = queue
	return t
}

// WithMaxRetry sets the max retry count for the task.
func (t *Task) WithMaxRetry(maxRetry int) *Task {
	t.MaxRetry = maxRetry
	return t
}

// WithTimeout sets the timeout for the task.
func (t *Task) WithTimeout(timeout time.Duration) *Task {
	t.Timeout = timeout
	return t
}

// WithDeadline sets the deadline for the task.
func (t *Task) WithDeadline(deadline time.Time) *Task {
	t.Deadline = deadline
	return t
}

// WithRetention sets the retention period for the task.
func (t *Task) WithRetention(retention time.Duration) *Task {
	t.Retention = retention
	return t
}

// WithUnique sets uniqueness constraints for the task.
func (t *Task) WithUnique(key string, ttl time.Duration) *Task {
	t.UniqueKey = key
	t.UniqueTTL = ttl
	return t
}

// WithGroup sets the group for the task.
func (t *Task) WithGroup(group string) *Task {
	t.Group = group
	return t
}

// RetryPolicy defines retry behavior for tasks.
type RetryPolicy struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	RetryOnError   func(err error) bool
}

// DefaultRetryPolicy returns a default retry policy with exponential backoff.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 10 * time.Second,
		MaxDelay:     10 * time.Minute,
		Multiplier:   2.0,
	}
}

// CalculateDelay calculates the delay for the nth retry attempt.
func (p RetryPolicy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialDelay
	}

	delay := p.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * p.Multiplier)
		if delay > p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}
	return delay
}
