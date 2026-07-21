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

type EmbeddedConfig struct {
	GRPCPort      *int
	APIPort       *int
	StartAPI      *bool
	RunMigrations *bool
	RabbitMQURL   *string
	LogLevel      *string
	DatabaseURL   string
}

type EmbeddedOption func(*EmbeddedConfig)

type EmbeddedBackend func(ctx context.Context, cfg EmbeddedConfig) (shutdown func(context.Context) error, err error)

var embeddedBackend EmbeddedBackend

func RegisterEmbeddedBackend(b EmbeddedBackend) {
	embeddedBackend = b
}

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
