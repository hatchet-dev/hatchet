package hatchet

import (
	"context"
	"fmt"
	"os"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: bridges to the internal v0 client option type
)

const EmbeddedDatabaseURLEnv = "HATCHET_CLIENT_EMBEDDED_DATABASE_URL"

func resolveEmbeddedConfig(probe *v0Client.ClientOpts) (*EmbeddedConfig, error) { //nolint:staticcheck // SA1019
	if probe.Embedded != nil {
		cfg, ok := probe.Embedded.(*EmbeddedConfig)
		if !ok {
			return nil, fmt.Errorf("unexpected embedded config type %T", probe.Embedded)
		}
		return cfg, nil
	}
	if url := os.Getenv(EmbeddedDatabaseURLEnv); url != "" {
		return &EmbeddedConfig{DatabaseURL: url}, nil
	}
	return nil, nil
}

// EmbeddedConfig configures the embedded engine started by WithEmbedded. The
// WithEmbedded* options set these fields.
type EmbeddedConfig struct {
	// GRPCPort overrides the port the embedded engine's gRPC server listens on.
	GRPCPort *int
	// APIPort overrides the port the embedded API server listens on.
	APIPort *int
	// StartAPI controls whether the embedded API server starts. Defaults to true.
	StartAPI *bool
	// RunMigrations controls whether the engine applies database migrations at
	// startup. Defaults to true.
	RunMigrations *bool
	// RabbitMQURL points the engine at a RabbitMQ instance for its message queue.
	RabbitMQURL *string
	// LogLevel sets the embedded engine's log level.
	LogLevel *string
	// DatabaseURL points the engine at an existing Postgres database instead of
	// the bundled one.
	DatabaseURL string
}

// EmbeddedOption configures the embedded engine started by WithEmbedded.
type EmbeddedOption func(*EmbeddedConfig)

// EmbeddedBackend boots the embedded engine and returns a shutdown function.
// The hatchet-embedded package provides the implementation and registers it via
// RegisterEmbeddedBackend when imported.
type EmbeddedBackend func(ctx context.Context, cfg EmbeddedConfig) (shutdown func(context.Context) error, err error)

var embeddedBackend EmbeddedBackend

// RegisterEmbeddedBackend registers the function that boots the embedded
// engine. The hatchet-embedded package calls it from an init function, so a
// blank import of that package is all an application needs.
func RegisterEmbeddedBackend(b EmbeddedBackend) {
	embeddedBackend = b
}

// WithEmbedded runs a full Hatchet engine in-process for local development, with
// no API token or Docker required. By default it starts a bundled Postgres;
// pass WithEmbeddedDatabaseURL to point it at your own instead.
//
// Requires a blank import of the hatchet-embedded package
// (github.com/hatchet-dev/hatchet-embedded), which registers the engine
// backend. See the [Embedded Hatchet guide] for usage.
//
// [Embedded Hatchet guide]: https://docs.hatchet.run/v1/embedded
func WithEmbedded(opts ...EmbeddedOption) v0Client.ClientOpt { //nolint:staticcheck // SA1019
	cfg := EmbeddedConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	return func(co *v0Client.ClientOpts) { co.Embedded = &cfg } //nolint:staticcheck // SA1019
}

// WithEmbeddedDatabaseURL points embedded mode at an existing Postgres instead
// of the bundled one.
func WithEmbeddedDatabaseURL(url string) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.DatabaseURL = url }
}

// WithEmbeddedGRPCPort overrides the port the embedded engine's gRPC server
// listens on.
func WithEmbeddedGRPCPort(port int) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.GRPCPort = &port }
}

// WithEmbeddedAPIPort overrides the port the embedded API server listens on.
func WithEmbeddedAPIPort(port int) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.APIPort = &port }
}

// WithoutEmbeddedAPI disables the embedded API server.
func WithoutEmbeddedAPI() EmbeddedOption {
	return func(c *EmbeddedConfig) { off := false; c.StartAPI = &off }
}

// WithoutEmbeddedMigrations disables database migrations at engine startup. Use
// this on all engines except one when running a fleet against a shared
// database.
func WithoutEmbeddedMigrations() EmbeddedOption {
	return func(c *EmbeddedConfig) { off := false; c.RunMigrations = &off }
}

// WithEmbeddedRabbitMQ points the engine at a RabbitMQ instance for its message
// queue.
func WithEmbeddedRabbitMQ(url string) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.RabbitMQURL = &url }
}

// WithEmbeddedLogLevel sets the embedded engine's log level.
func WithEmbeddedLogLevel(level string) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.LogLevel = &level }
}
