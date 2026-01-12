package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "localhost:7233", cfg.TemporalHostPort)
	assert.Equal(t, "default", cfg.Namespace)
	assert.Equal(t, "codeai-workflows", cfg.TaskQueue)
	assert.Equal(t, 100, cfg.MaxConcurrentWorkflows)
	assert.Equal(t, 100, cfg.MaxConcurrentActivities)
	assert.Equal(t, 30*time.Minute, cfg.DefaultTimeout)
	assert.Equal(t, "codeai-worker", cfg.WorkerID)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty temporal host",
			config: Config{
				TemporalHostPort:        "",
				Namespace:               "default",
				TaskQueue:               "test",
				MaxConcurrentWorkflows:  10,
				MaxConcurrentActivities: 10,
			},
			wantErr: true,
			errMsg:  "TemporalHostPort",
		},
		{
			name: "empty namespace",
			config: Config{
				TemporalHostPort:        "localhost:7233",
				Namespace:               "",
				TaskQueue:               "test",
				MaxConcurrentWorkflows:  10,
				MaxConcurrentActivities: 10,
			},
			wantErr: true,
			errMsg:  "Namespace",
		},
		{
			name: "empty task queue",
			config: Config{
				TemporalHostPort:        "localhost:7233",
				Namespace:               "default",
				TaskQueue:               "",
				MaxConcurrentWorkflows:  10,
				MaxConcurrentActivities: 10,
			},
			wantErr: true,
			errMsg:  "TaskQueue",
		},
		{
			name: "zero concurrent workflows",
			config: Config{
				TemporalHostPort:        "localhost:7233",
				Namespace:               "default",
				TaskQueue:               "test",
				MaxConcurrentWorkflows:  0,
				MaxConcurrentActivities: 10,
			},
			wantErr: true,
			errMsg:  "MaxConcurrentWorkflows",
		},
		{
			name: "zero concurrent activities",
			config: Config{
				TemporalHostPort:        "localhost:7233",
				Namespace:               "default",
				TaskQueue:               "test",
				MaxConcurrentWorkflows:  10,
				MaxConcurrentActivities: 0,
			},
			wantErr: true,
			errMsg:  "MaxConcurrentActivities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewEngine(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		eng, err := NewEngine(cfg)

		require.NoError(t, err)
		assert.NotNil(t, eng)
		assert.False(t, eng.IsRunning())
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := Config{}
		eng, err := NewEngine(cfg)

		require.Error(t, err)
		assert.Nil(t, eng)
	})
}

func TestEngineRegisterWorkflow(t *testing.T) {
	cfg := DefaultConfig()
	eng, err := NewEngine(cfg)
	require.NoError(t, err)

	// Register a mock workflow
	mockWorkflow := func() {}
	eng.RegisterWorkflow(mockWorkflow)

	// Verify workflow was registered (internal state check)
	assert.Equal(t, 1, len(eng.workflows))
}

func TestEngineRegisterActivity(t *testing.T) {
	cfg := DefaultConfig()
	eng, err := NewEngine(cfg)
	require.NoError(t, err)

	// Register a mock activity
	mockActivity := func() {}
	eng.RegisterActivity(mockActivity)

	// Verify activity was registered (internal state check)
	assert.Equal(t, 1, len(eng.activities))
}

func TestEngineNotStartedErrors(t *testing.T) {
	cfg := DefaultConfig()
	eng, err := NewEngine(cfg)
	require.NoError(t, err)

	// Should return error when engine is not started
	err = eng.Stop()
	assert.ErrorIs(t, err, ErrEngineNotStarted)
}

func TestErrConfigInvalid(t *testing.T) {
	err := ErrConfigInvalid{Field: "TestField", Reason: "test reason"}
	assert.Contains(t, err.Error(), "TestField")
	assert.Contains(t, err.Error(), "test reason")
}

func TestErrWorkflowFailed(t *testing.T) {
	cause := assert.AnError
	err := ErrWorkflowFailed{
		WorkflowID: "wf-123",
		RunID:      "run-456",
		Cause:      cause,
	}

	assert.Contains(t, err.Error(), "wf-123")
	assert.Contains(t, err.Error(), "run-456")
	assert.ErrorIs(t, err, cause)
}

func TestErrActivityFailed(t *testing.T) {
	cause := assert.AnError
	err := ErrActivityFailed{
		ActivityType: "TestActivity",
		Cause:        cause,
	}

	assert.Contains(t, err.Error(), "TestActivity")
	assert.ErrorIs(t, err, cause)
}
