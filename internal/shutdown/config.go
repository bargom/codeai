package shutdown

import "time"

// Config holds configuration for the shutdown manager.
type Config struct {
	// OverallTimeout is the maximum time allowed for all shutdown hooks to complete.
	// Default: 30 seconds.
	OverallTimeout time.Duration

	// PerHookTimeout is the maximum time allowed for a single hook to complete.
	// Default: 10 seconds.
	PerHookTimeout time.Duration

	// DrainTimeout is the time to wait for in-flight requests to complete.
	// Default: 10 seconds.
	DrainTimeout time.Duration

	// SlowHookThreshold is the duration after which a hook is considered slow
	// and a warning is logged.
	// Default: 5 seconds.
	SlowHookThreshold time.Duration
}

// DefaultConfig returns the default shutdown configuration.
func DefaultConfig() Config {
	return Config{
		OverallTimeout:    30 * time.Second,
		PerHookTimeout:    10 * time.Second,
		DrainTimeout:      10 * time.Second,
		SlowHookThreshold: 5 * time.Second,
	}
}

// Validate validates the configuration and sets defaults for zero values.
func (c *Config) Validate() {
	if c.OverallTimeout <= 0 {
		c.OverallTimeout = 30 * time.Second
	}
	if c.PerHookTimeout <= 0 {
		c.PerHookTimeout = 10 * time.Second
	}
	if c.DrainTimeout <= 0 {
		c.DrainTimeout = 10 * time.Second
	}
	if c.SlowHookThreshold <= 0 {
		c.SlowHookThreshold = 5 * time.Second
	}
}
