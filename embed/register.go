package embed

import (
	"context"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func init() {
	hatchet.RegisterEmbeddedBackend(func(ctx context.Context, cfg hatchet.EmbeddedConfig) (func(context.Context) error, error) {
		inst, err := start(ctx, embeddedConfigToOptions(cfg)...)
		if err != nil {
			return nil, err
		}
		return inst.Shutdown, nil
	})
}

func embeddedConfigToOptions(cfg hatchet.EmbeddedConfig) []Option {
	opts := []Option{WithPostgres(cfg.DatabaseURL)}
	if cfg.GRPCPort != nil {
		opts = append(opts, WithGRPCPort(*cfg.GRPCPort))
	}
	if cfg.APIPort != nil {
		opts = append(opts, WithAPIPort(*cfg.APIPort))
	}
	if cfg.StartAPI != nil && !*cfg.StartAPI {
		opts = append(opts, WithoutAPI())
	}
	if cfg.RunMigrations != nil && !*cfg.RunMigrations {
		opts = append(opts, WithoutMigrations())
	}
	if cfg.RabbitMQURL != nil {
		opts = append(opts, WithRabbitMQ(*cfg.RabbitMQURL))
	}
	if cfg.LogLevel != nil {
		opts = append(opts, WithLogLevel(*cfg.LogLevel))
	}
	return opts
}
