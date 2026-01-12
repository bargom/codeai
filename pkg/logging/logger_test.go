package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name: "json format",
			config: Config{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "text format",
			config: Config{
				Level:  "info",
				Format: "text",
			},
		},
		{
			name: "debug level",
			config: Config{
				Level:  "debug",
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			assert.NotNil(t, logger)
		})
	}
}

func TestLogger_With(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	childLogger := logger.With("key", "value")
	assert.NotNil(t, childLogger)

	childLogger.Info("test message")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, "test message", logEntry["msg"])
}

func TestLogger_WithModule(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	moduleLogger := logger.WithModule("auth")
	moduleLogger.Info("auth event")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "auth", logEntry["module"])
}

func TestLogger_WithOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	opLogger := logger.WithOperation("create_user")
	opLogger.Info("operation started")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "create_user", logEntry["operation"])
}

func TestLogger_WithEntity(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	entityLogger := logger.WithEntity("user", "123")
	entityLogger.Info("entity action")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "user", logEntry["entity"])
	assert.Equal(t, "123", logEntry["entity_id"])
}

func TestContextHandler_WithContextValues(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")
	ctx = WithUserID(ctx, "user-789")

	logger.InfoContext(ctx, "test with context")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-123", logEntry["request_id"])
	assert.Equal(t, "trace-456", logEntry["trace_id"])
	assert.Equal(t, "user-789", logEntry["user_id"])
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "warn", Format: "json"}
	logger := NewWithWriter(config, &buf)

	// Debug and Info should be filtered
	logger.Debug("debug message")
	logger.Info("info message")
	assert.Empty(t, buf.String())

	// Warn should be logged
	logger.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")

	buf.Reset()

	// Error should be logged
	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	logger.Info("json test", "key", "value")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "json test", logEntry["msg"])
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "value", logEntry["key"])
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "text"}
	logger := NewWithWriter(config, &buf)

	logger.Info("text test", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "text test")
	assert.Contains(t, output, "key=value")
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "TEXT")
	t.Setenv("LOG_OUTPUT", "stderr")
	t.Setenv("LOG_ADD_SOURCE", "true")

	config := ConfigFromEnv()

	assert.Equal(t, "debug", config.Level)
	assert.Equal(t, "text", config.Format)
	assert.Equal(t, "stderr", config.Output)
	assert.True(t, config.AddSource)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "info", config.Level)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "stdout", config.Output)
	assert.False(t, config.AddSource)
	assert.Equal(t, 1.0, config.SampleRate)
}

func TestLogger_WithWorkflow(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	wfLogger := logger.WithWorkflow("wf-123", "exec-456")
	wfLogger.Info("workflow event")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "wf-123", logEntry["workflow_id"])
	assert.Equal(t, "exec-456", logEntry["execution_id"])
}

func TestLogger_WithJob(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	jobLogger := logger.WithJob("job-789")
	jobLogger.Info("job event")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "job-789", logEntry["job_id"])
}

func TestLogger_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)

	groupLogger := logger.WithGroup("database")
	groupLogger.Info("db event", "query", "SELECT *")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Group should create nested structure
	dbGroup, ok := logEntry["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "SELECT *", dbGroup["query"])
}

func TestModuleLogger(t *testing.T) {
	var buf bytes.Buffer

	// Create and set a custom default logger
	config := Config{Level: "info", Format: "json"}
	logger := NewWithWriter(config, &buf)
	slog.SetDefault(logger.Logger)

	moduleLog := ModuleLogger("test-module")
	moduleLog.Info("module test")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1)

	var logEntry map[string]any
	err := json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test-module", logEntry["module"])
}
