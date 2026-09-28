package loader

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	serverlessoperatorconfig "github.com/hatchet-dev/hatchet/pkg/config/serverlessoperator"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// LoadServerlessOperatorConfigFile binds SERVERLESS_OPERATOR_* onto the binary's config and
// validates it; a value the binary cannot start with is refused here, before any connection
// is opened.
func LoadServerlessOperatorConfigFile() (*serverlessoperatorconfig.ConfigFile, error) {
	cf := &serverlessoperatorconfig.ConfigFile{}

	if _, err := loaderutils.LoadConfigFromViper(serverlessoperatorconfig.BindAllEnv, cf); err != nil {
		return nil, err
	}

	if err := cf.Validate(); err != nil {
		return nil, err
	}

	return cf, nil
}

// LoadServerlessOperatorRepository opens the binary's pool on the configured database and
// builds the serverless repository on it. The returned cleanup releases the repository's
// resources and closes the pool.
func LoadServerlessOperatorRepository(ctx context.Context, cf *serverlessoperatorconfig.ConfigFile, l *zerolog.Logger) (repository.ServerlessRepository, func() error, error) {
	pool, err := newServerlessOperatorPool(ctx, cf)

	if err != nil {
		return nil, nil, err
	}

	repo, cleanupRepo := repository.NewServerlessRepositoryFromPool(pool, l)

	return repo, func() error {
		err := cleanupRepo()
		pool.Close()

		return err
	}, nil
}

func newServerlessOperatorPool(ctx context.Context, cf *serverlessoperatorconfig.ConfigFile) (*pgxpool.Pool, error) {
	return NewPgxPool(ctx, cf.DatabaseUrl, PgxPoolOpts{
		ApplicationName: "hatchet-serverless-operator",
	})
}

// LoadServerlessOperatorTokenExchange builds the tenant token source the config selects
// (Validate has checked the mode and its variables). The returned close stops a local
// exchange's file watcher.
func LoadServerlessOperatorTokenExchange(cf *serverlessoperatorconfig.ConfigFile, l *zerolog.Logger) (hostgrpc.TokenSource, func(), error) {
	switch mode := cf.TokenExchangeMode(); mode {
	case serverlessoperatorconfig.TokenExchangeLocal:
		return newLocalTokenExchange(cf.TokenFile, l)
	case serverlessoperatorconfig.TokenExchangeStatic:
		return newStaticTokenExchange(cf.ClientToken, l)
	default:
		return nil, nil, fmt.Errorf("unknown token exchange %q", mode)
	}
}

func newLocalTokenExchange(path string, l *zerolog.Logger) (hostgrpc.TokenSource, func(), error) {
	ex, err := hostgrpc.NewLocalExchange(path, hostgrpc.WithExchangeLogger(l))

	if err != nil {
		return nil, nil, err
	}

	return ex, ex.Close, nil
}

// newStaticTokenExchange serves the one tenant the token belongs to. It is a development
// convenience: production deployments list their tenants in a token file.
func newStaticTokenExchange(token string, l *zerolog.Logger) (hostgrpc.TokenSource, func(), error) {
	ex, err := hostgrpc.NewStaticExchange(token)

	if err != nil {
		return nil, nil, err
	}

	l.Warn().Str("tenant_id", ex.TenantId().String()).Msgf("using HATCHET_CLIENT_TOKEN for a single tenant; set SERVERLESS_OPERATOR_TOKEN_EXCHANGE=%s with a token file for production", serverlessoperatorconfig.TokenExchangeLocal)

	return ex, func() {}, nil
}
