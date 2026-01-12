// Package scheduler provides configuration and initialization for the job scheduler.
package scheduler

import (
	"time"

	"github.com/bargom/codeai/internal/scheduler/queue"
)

// Config holds the complete scheduler configuration.
type Config struct {
	Redis      RedisConfig      `yaml:"redis"`
	Queue      QueueConfig      `yaml:"queue"`
	Tasks      TasksConfig      `yaml:"tasks"`
	Retry      RetryConfig      `yaml:"retry"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// QueueConfig holds queue settings.
type QueueConfig struct {
	Concurrency     int            `yaml:"concurrency"`
	Queues          map[string]int `yaml:"queues"`
	ShutdownTimeout time.Duration  `yaml:"shutdown_timeout"`
}

// TasksConfig holds task-specific configurations.
type TasksConfig struct {
	AIAgent        TaskConfig `yaml:"ai_agent"`
	TestSuite      TaskConfig `yaml:"test_suite"`
	DataProcessing TaskConfig `yaml:"data_processing"`
	Webhook        TaskConfig `yaml:"webhook"`
	Cleanup        TaskConfig `yaml:"cleanup"`
}

// TaskConfig holds configuration for a specific task type.
type TaskConfig struct {
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
	Queue      string        `yaml:"queue"`
	CronSpec   string        `yaml:"cron_spec,omitempty"`
	Retention  time.Duration `yaml:"retention,omitempty"`
}

// RetryConfig holds retry behavior settings.
type RetryConfig struct {
	InitialDelay time.Duration `yaml:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay"`
	Multiplier   float64       `yaml:"multiplier"`
}

// MonitoringConfig holds monitoring settings.
type MonitoringConfig struct {
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	MetricsPath    string `yaml:"metrics_path"`
	HealthPath     string `yaml:"health_path"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 2,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Queue: QueueConfig{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ShutdownTimeout: 30 * time.Second,
		},
		Tasks: TasksConfig{
			AIAgent: TaskConfig{
				Timeout:    5 * time.Minute,
				MaxRetries: 3,
				Queue:      "default",
			},
			TestSuite: TaskConfig{
				Timeout:    30 * time.Minute,
				MaxRetries: 2,
				Queue:      "default",
			},
			DataProcessing: TaskConfig{
				Timeout:    15 * time.Minute,
				MaxRetries: 3,
				Queue:      "low",
			},
			Webhook: TaskConfig{
				Timeout:    30 * time.Second,
				MaxRetries: 5,
				Queue:      "critical",
			},
			Cleanup: TaskConfig{
				Timeout:    10 * time.Minute,
				MaxRetries: 1,
				Queue:      "low",
				CronSpec:   "0 2 * * *",
				Retention:  30 * 24 * time.Hour,
			},
		},
		Retry: RetryConfig{
			InitialDelay: 10 * time.Second,
			MaxDelay:     10 * time.Minute,
			Multiplier:   2.0,
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled: true,
			MetricsPath:    "/metrics",
			HealthPath:     "/health",
		},
	}
}

// ToQueueConfig converts the scheduler config to a queue.Config.
func (c *Config) ToQueueConfig() queue.Config {
	return queue.Config{
		RedisAddr:       c.Redis.Addr,
		RedisPassword:   c.Redis.Password,
		RedisDB:         c.Redis.DB,
		Concurrency:     c.Queue.Concurrency,
		Queues:          c.Queue.Queues,
		MaxRetry:        3,
		ShutdownTimeout: c.Queue.ShutdownTimeout,
	}
}
