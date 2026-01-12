package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for OpenAPI generation.
type Config struct {
	// Metadata
	Title          string `json:"title" yaml:"title"`
	Description    string `json:"description" yaml:"description"`
	Version        string `json:"version" yaml:"version"`
	TermsOfService string `json:"terms_of_service" yaml:"terms_of_service"`

	// Contact info
	ContactName  string `json:"contact_name" yaml:"contact_name"`
	ContactURL   string `json:"contact_url" yaml:"contact_url"`
	ContactEmail string `json:"contact_email" yaml:"contact_email"`

	// License info
	LicenseName string `json:"license_name" yaml:"license_name"`
	LicenseURL  string `json:"license_url" yaml:"license_url"`

	// Servers
	Servers []ServerConfig `json:"servers" yaml:"servers"`

	// Security
	DefaultSecurity []string `json:"default_security" yaml:"default_security"`

	// Output
	OutputFormat string `json:"output_format" yaml:"output_format"` // json or yaml
}

// ServerConfig represents a server configuration.
type ServerConfig struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Title:        "API",
		Version:      "1.0.0",
		OutputFormat: "yaml",
	}
}

// LoadConfig loads configuration from a file.
// Supports JSON and YAML formats based on file extension.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, config); err != nil {
			if err := json.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config (tried YAML and JSON): %w", err)
			}
		}
	}

	return config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Title == "" {
		return fmt.Errorf("title is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.OutputFormat != "" && c.OutputFormat != "json" && c.OutputFormat != "yaml" {
		return fmt.Errorf("output_format must be 'json' or 'yaml'")
	}
	return nil
}

// ToInfo converts the config to an OpenAPI Info object.
func (c *Config) ToInfo() Info {
	info := Info{
		Title:          c.Title,
		Description:    c.Description,
		TermsOfService: c.TermsOfService,
		Version:        c.Version,
	}

	if c.ContactName != "" || c.ContactURL != "" || c.ContactEmail != "" {
		info.Contact = &Contact{
			Name:  c.ContactName,
			URL:   c.ContactURL,
			Email: c.ContactEmail,
		}
	}

	if c.LicenseName != "" {
		info.License = &License{
			Name: c.LicenseName,
			URL:  c.LicenseURL,
		}
	}

	return info
}

// ToServers converts the config to OpenAPI Server objects.
func (c *Config) ToServers() []Server {
	if len(c.Servers) == 0 {
		return nil
	}

	servers := make([]Server, len(c.Servers))
	for i, s := range c.Servers {
		servers[i] = Server{
			URL:         s.URL,
			Description: s.Description,
		}
	}
	return servers
}
