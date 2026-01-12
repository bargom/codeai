package email

import (
	"os"

	"github.com/bargom/codeai/pkg/integration/brevo"
)

// Config holds the email notification configuration.
type Config struct {
	Brevo    brevo.Config
	Settings Settings
}

// Settings holds email service settings.
type Settings struct {
	EnableAutoNotifications bool
	BatchSize               int
	RetryAttempts           int
	RetryDelaySeconds       int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Brevo: brevo.Config{
			APIKey: os.Getenv("BREVO_API_KEY"),
			DefaultSender: brevo.EmailAddress{
				Name:  "CodeAI Platform",
				Email: "noreply@codeai.io",
			},
			TimeoutSeconds: 30,
		},
		Settings: Settings{
			EnableAutoNotifications: true,
			BatchSize:               50,
			RetryAttempts:           3,
			RetryDelaySeconds:       5,
		},
	}
}

// ConfigFromEnv creates a configuration from environment variables.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if apiKey := os.Getenv("BREVO_API_KEY"); apiKey != "" {
		cfg.Brevo.APIKey = apiKey
	}

	if senderName := os.Getenv("EMAIL_SENDER_NAME"); senderName != "" {
		cfg.Brevo.DefaultSender.Name = senderName
	}

	if senderEmail := os.Getenv("EMAIL_SENDER_ADDRESS"); senderEmail != "" {
		cfg.Brevo.DefaultSender.Email = senderEmail
	}

	return cfg
}
