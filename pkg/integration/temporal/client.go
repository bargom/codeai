// Package temporal provides Temporal client integration utilities.
package temporal

import (
	"fmt"

	"go.temporal.io/sdk/client"
)

// ClientConfig holds configuration for the Temporal client.
type ClientConfig struct {
	HostPort  string
	Namespace string
	Identity  string
}

// DefaultClientConfig returns a default Temporal client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		HostPort:  "localhost:7233",
		Namespace: "default",
		Identity:  "codeai-client",
	}
}

// NewClient creates a new Temporal client with the given configuration.
func NewClient(cfg ClientConfig) (client.Client, error) {
	options := client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Identity:  cfg.Identity,
	}

	c, err := client.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return c, nil
}

// MustNewClient creates a new Temporal client and panics on error.
func MustNewClient(cfg ClientConfig) client.Client {
	c, err := NewClient(cfg)
	if err != nil {
		panic(err)
	}
	return c
}

// ClientFactory creates Temporal clients.
type ClientFactory struct {
	config ClientConfig
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(cfg ClientConfig) *ClientFactory {
	return &ClientFactory{config: cfg}
}

// Create creates a new Temporal client.
func (f *ClientFactory) Create() (client.Client, error) {
	return NewClient(f.config)
}

// WithNamespace returns a new ClientFactory with the specified namespace.
func (f *ClientFactory) WithNamespace(namespace string) *ClientFactory {
	newConfig := f.config
	newConfig.Namespace = namespace
	return &ClientFactory{config: newConfig}
}

// WithHostPort returns a new ClientFactory with the specified host:port.
func (f *ClientFactory) WithHostPort(hostPort string) *ClientFactory {
	newConfig := f.config
	newConfig.HostPort = hostPort
	return &ClientFactory{config: newConfig}
}
