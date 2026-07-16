package hatchet

import (
	"context"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: bridges to the internal v0 client option type
)

// EmbeddedConfig describes an in-process Hatchet engine requested via WithEmbeddedPostgres.
type EmbeddedConfig struct {
	GRPCPort      *int
	APIPort       *int
	StartAPI      *bool
	RunMigrations *bool
	RabbitMQURL   *string
	LogLevel      *string
	DatabaseURL   string
}

// EmbeddedOption customizes an embedded engine.
type EmbeddedOption func(*EmbeddedConfig)

// EmbeddedBackend boots an in-process engine for cfg and returns a shutdown function.
// It is registered by importing github.com/hatchet-dev/hatchet/embed.
type EmbeddedBackend func(ctx context.Context, cfg EmbeddedConfig) (shutdown func(context.Context) error, err error)

var embeddedBackend EmbeddedBackend

// RegisterEmbeddedBackend wires the embed package into the SDK. Callers do not invoke
// this directly; it runs from the embed package's init when it is imported.
func RegisterEmbeddedBackend(b EmbeddedBackend) {
	embeddedBackend = b
}

// WithEmbeddedPostgres runs a full Hatchet engine in-process against the given Postgres
// URL, with the database as the only shared source of truth. Requires a blank import of
// github.com/hatchet-dev/hatchet/embed so the engine is linked into the binary.
func WithEmbeddedPostgres(databaseURL string, opts ...EmbeddedOption) v0Client.ClientOpt { //nolint:staticcheck // SA1019
	cfg := EmbeddedConfig{DatabaseURL: databaseURL}
	for _, o := range opts {
		o(&cfg)
	}
	return func(co *v0Client.ClientOpts) { co.Embedded = &cfg } //nolint:staticcheck // SA1019
}

func WithEmbeddedGRPCPort(port int) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.GRPCPort = &port }
}

func WithEmbeddedAPIPort(port int) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.APIPort = &port }
}

func WithoutEmbeddedAPI() EmbeddedOption {
	return func(c *EmbeddedConfig) { off := false; c.StartAPI = &off }
}

func WithoutEmbeddedMigrations() EmbeddedOption {
	return func(c *EmbeddedConfig) { off := false; c.RunMigrations = &off }
}

func WithEmbeddedRabbitMQ(url string) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.RabbitMQURL = &url }
}

func WithEmbeddedLogLevel(level string) EmbeddedOption {
	return func(c *EmbeddedConfig) { c.LogLevel = &level }
}
