// Package logging provides structured logging with request tracing and sensitive data redaction.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config holds the logging configuration.
type Config struct {
	// Level sets the minimum log level: debug, info, warn, error
	Level string

	// Format specifies the output format: json or text
	Format string

	// Output specifies the output destination: stdout, stderr, or a file path
	Output string

	// AddSource adds source file and line number to log entries
	AddSource bool

	// SampleRate for high-volume logs (0.0-1.0, 1.0 = log all)
	SampleRate float64

	// SlowQueryThreshold marks queries slower than this as slow
	SlowQueryThreshold time.Duration

	// RedactPatterns are additional patterns to redact
	RedactPatterns []string

	// AllowlistFields are field names that should not be redacted even if they match patterns
	AllowlistFields []string
}

// DefaultConfig returns sensible defaults for logging configuration.
func DefaultConfig() Config {
	return Config{
		Level:              "info",
		Format:             "json",
		Output:             "stdout",
		AddSource:          false,
		SampleRate:         1.0,
		SlowQueryThreshold: 100 * time.Millisecond,
	}
}

// ConfigFromEnv creates a configuration from environment variables.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Level = strings.ToLower(level)
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Format = strings.ToLower(format)
	}

	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		cfg.Output = output
	}

	if os.Getenv("LOG_ADD_SOURCE") == "true" {
		cfg.AddSource = true
	}

	return cfg
}

// ParseLevel converts a string level to slog.Level.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetOutput returns the io.Writer for the configured output.
func (c Config) GetOutput() io.Writer {
	switch c.Output {
	case "", "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		f, err := os.OpenFile(c.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return os.Stdout
		}
		return f
	}
}
