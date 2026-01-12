package integration

import (
	"os"
	"strconv"
	"time"
)

// Config holds the complete configuration for an integration client.
type Config struct {
	// ServiceName is the name of the external service.
	ServiceName string

	// BaseURL is the base URL of the external service.
	BaseURL string

	// Timeout configures timeout behavior.
	Timeout TimeoutConfig

	// Retry configures retry behavior.
	Retry RetryConfig

	// CircuitBreaker configures circuit breaker behavior.
	CircuitBreaker CircuitBreakerConfig

	// Headers are default headers to include in all requests.
	Headers map[string]string

	// Auth configures authentication.
	Auth AuthConfig

	// EnableMetrics enables metrics collection.
	EnableMetrics bool

	// EnableLogging enables request/response logging.
	EnableLogging bool

	// RedactFields specifies fields to redact in logs.
	RedactFields []string

	// UserAgent is the User-Agent header value.
	UserAgent string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:        DefaultTimeoutConfig(),
		Retry:          DefaultRetryConfig(),
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		Headers:        make(map[string]string),
		EnableMetrics:  true,
		EnableLogging:  true,
		RedactFields:   []string{"password", "token", "secret", "api_key", "authorization"},
		UserAgent:      "CodeAI-Client/1.0",
	}
}

// ConfigFromEnv creates a configuration from environment variables.
// It uses the service name as a prefix (e.g., GITHUB_BASE_URL, GITHUB_TIMEOUT).
func ConfigFromEnv(serviceName string) Config {
	config := DefaultConfig()
	config.ServiceName = serviceName

	prefix := envPrefix(serviceName)

	if url := os.Getenv(prefix + "BASE_URL"); url != "" {
		config.BaseURL = url
	}

	if timeout := envDuration(prefix + "TIMEOUT"); timeout > 0 {
		config.Timeout.Default = timeout
	}

	if timeout := envDuration(prefix + "CONNECT_TIMEOUT"); timeout > 0 {
		config.Timeout.Connect = timeout
	}

	if maxAttempts := envInt(prefix + "MAX_RETRIES"); maxAttempts > 0 {
		config.Retry.MaxAttempts = maxAttempts
	}

	if baseDelay := envDuration(prefix + "RETRY_DELAY"); baseDelay > 0 {
		config.Retry.BaseDelay = baseDelay
	}

	if threshold := envInt(prefix + "CIRCUIT_THRESHOLD"); threshold > 0 {
		config.CircuitBreaker.FailureThreshold = threshold
	}

	if cbTimeout := envDuration(prefix + "CIRCUIT_TIMEOUT"); cbTimeout > 0 {
		config.CircuitBreaker.Timeout = cbTimeout
	}

	if userAgent := os.Getenv(prefix + "USER_AGENT"); userAgent != "" {
		config.UserAgent = userAgent
	}

	return config
}

// AuthConfig configures authentication.
type AuthConfig struct {
	// Type is the authentication type.
	Type AuthType

	// Token is the bearer token (for AuthBearer).
	Token string

	// APIKey is the API key (for AuthAPIKey).
	APIKey string

	// APIKeyHeader is the header name for API key (for AuthAPIKey).
	// Default: "X-API-Key"
	APIKeyHeader string

	// Username is the username (for AuthBasic).
	Username string

	// Password is the password (for AuthBasic).
	Password string

	// OAuth2Config is the OAuth2 configuration (for AuthOAuth2).
	OAuth2Config *OAuth2Config
}

// AuthType represents the type of authentication.
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBearer AuthType = "bearer"
	AuthAPIKey AuthType = "api_key"
	AuthBasic  AuthType = "basic"
	AuthOAuth2 AuthType = "oauth2"
)

// OAuth2Config configures OAuth2 authentication.
type OAuth2Config struct {
	// ClientID is the OAuth2 client ID.
	ClientID string

	// ClientSecret is the OAuth2 client secret.
	ClientSecret string

	// TokenURL is the URL to obtain access tokens.
	TokenURL string

	// Scopes are the OAuth2 scopes.
	Scopes []string

	// Token is the current access token.
	Token string

	// RefreshToken is the refresh token.
	RefreshToken string

	// TokenExpiry is when the token expires.
	TokenExpiry time.Time
}

// ConfigBuilder provides a fluent interface for building configurations.
type ConfigBuilder struct {
	config Config
}

// NewConfigBuilder creates a new configuration builder.
func NewConfigBuilder(serviceName string) *ConfigBuilder {
	config := DefaultConfig()
	config.ServiceName = serviceName
	return &ConfigBuilder{config: config}
}

// BaseURL sets the base URL.
func (b *ConfigBuilder) BaseURL(url string) *ConfigBuilder {
	b.config.BaseURL = url
	return b
}

// Timeout sets the default timeout.
func (b *ConfigBuilder) Timeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout.Default = timeout
	return b
}

// ConnectTimeout sets the connection timeout.
func (b *ConfigBuilder) ConnectTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout.Connect = timeout
	return b
}

// ReadTimeout sets the read timeout.
func (b *ConfigBuilder) ReadTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout.Read = timeout
	return b
}

// WriteTimeout sets the write timeout.
func (b *ConfigBuilder) WriteTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout.Write = timeout
	return b
}

// MaxRetries sets the maximum number of retry attempts.
func (b *ConfigBuilder) MaxRetries(attempts int) *ConfigBuilder {
	b.config.Retry.MaxAttempts = attempts
	return b
}

// RetryDelay sets the initial retry delay.
func (b *ConfigBuilder) RetryDelay(delay time.Duration) *ConfigBuilder {
	b.config.Retry.BaseDelay = delay
	return b
}

// MaxRetryDelay sets the maximum retry delay.
func (b *ConfigBuilder) MaxRetryDelay(delay time.Duration) *ConfigBuilder {
	b.config.Retry.MaxDelay = delay
	return b
}

// RetryMultiplier sets the retry delay multiplier.
func (b *ConfigBuilder) RetryMultiplier(multiplier float64) *ConfigBuilder {
	b.config.Retry.Multiplier = multiplier
	return b
}

// RetryJitter sets the retry jitter percentage.
func (b *ConfigBuilder) RetryJitter(jitter float64) *ConfigBuilder {
	b.config.Retry.Jitter = jitter
	return b
}

// CircuitBreakerThreshold sets the circuit breaker failure threshold.
func (b *ConfigBuilder) CircuitBreakerThreshold(threshold int) *ConfigBuilder {
	b.config.CircuitBreaker.FailureThreshold = threshold
	return b
}

// CircuitBreakerTimeout sets the circuit breaker timeout.
func (b *ConfigBuilder) CircuitBreakerTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.CircuitBreaker.Timeout = timeout
	return b
}

// CircuitBreakerHalfOpenRequests sets the number of half-open requests.
func (b *ConfigBuilder) CircuitBreakerHalfOpenRequests(requests int) *ConfigBuilder {
	b.config.CircuitBreaker.HalfOpenRequests = requests
	return b
}

// Header adds a default header.
func (b *ConfigBuilder) Header(key, value string) *ConfigBuilder {
	if b.config.Headers == nil {
		b.config.Headers = make(map[string]string)
	}
	b.config.Headers[key] = value
	return b
}

// Headers sets multiple default headers.
func (b *ConfigBuilder) Headers(headers map[string]string) *ConfigBuilder {
	if b.config.Headers == nil {
		b.config.Headers = make(map[string]string)
	}
	for k, v := range headers {
		b.config.Headers[k] = v
	}
	return b
}

// BearerAuth sets bearer token authentication.
func (b *ConfigBuilder) BearerAuth(token string) *ConfigBuilder {
	b.config.Auth = AuthConfig{
		Type:  AuthBearer,
		Token: token,
	}
	return b
}

// APIKeyAuth sets API key authentication.
func (b *ConfigBuilder) APIKeyAuth(apiKey, header string) *ConfigBuilder {
	if header == "" {
		header = "X-API-Key"
	}
	b.config.Auth = AuthConfig{
		Type:         AuthAPIKey,
		APIKey:       apiKey,
		APIKeyHeader: header,
	}
	return b
}

// BasicAuth sets basic authentication.
func (b *ConfigBuilder) BasicAuth(username, password string) *ConfigBuilder {
	b.config.Auth = AuthConfig{
		Type:     AuthBasic,
		Username: username,
		Password: password,
	}
	return b
}

// OAuth2Auth sets OAuth2 authentication.
func (b *ConfigBuilder) OAuth2Auth(config *OAuth2Config) *ConfigBuilder {
	b.config.Auth = AuthConfig{
		Type:         AuthOAuth2,
		OAuth2Config: config,
	}
	return b
}

// EnableMetrics enables or disables metrics collection.
func (b *ConfigBuilder) EnableMetrics(enable bool) *ConfigBuilder {
	b.config.EnableMetrics = enable
	return b
}

// EnableLogging enables or disables request/response logging.
func (b *ConfigBuilder) EnableLogging(enable bool) *ConfigBuilder {
	b.config.EnableLogging = enable
	return b
}

// RedactFields sets the fields to redact in logs.
func (b *ConfigBuilder) RedactFields(fields ...string) *ConfigBuilder {
	b.config.RedactFields = fields
	return b
}

// UserAgent sets the User-Agent header value.
func (b *ConfigBuilder) UserAgent(userAgent string) *ConfigBuilder {
	b.config.UserAgent = userAgent
	return b
}

// Build returns the built configuration.
func (b *ConfigBuilder) Build() Config {
	return b.config
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return &ConfigError{Field: "ServiceName", Message: "is required"}
	}
	if c.BaseURL == "" {
		return &ConfigError{Field: "BaseURL", Message: "is required"}
	}
	if c.Timeout.Default <= 0 {
		return &ConfigError{Field: "Timeout.Default", Message: "must be positive"}
	}
	if c.Retry.MaxAttempts <= 0 {
		return &ConfigError{Field: "Retry.MaxAttempts", Message: "must be positive"}
	}
	if c.CircuitBreaker.FailureThreshold <= 0 {
		return &ConfigError{Field: "CircuitBreaker.FailureThreshold", Message: "must be positive"}
	}
	return nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " " + e.Message
}

// Helper functions

func envPrefix(serviceName string) string {
	// Convert service name to uppercase with underscores
	result := ""
	for i, c := range serviceName {
		if c >= 'A' && c <= 'Z' && i > 0 {
			result += "_"
		}
		if c >= 'a' && c <= 'z' {
			result += string(c - 32) // to uppercase
		} else if c >= 'A' && c <= 'Z' {
			result += string(c)
		} else if c == '-' || c == ' ' {
			result += "_"
		} else {
			result += string(c)
		}
	}
	return result + "_"
}

func envDuration(key string) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return 0
	}

	// Try parsing as duration
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return 0
}

func envInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		return 0
	}
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return 0
}
