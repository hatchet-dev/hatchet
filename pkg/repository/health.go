package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepository interface {
	IsHealthy(ctx context.Context) bool
	PgStat() *pgxpool.Stat
}

type healthRepository struct {
	*sharedRepository
}

func newHealthRepository(shared *sharedRepository) HealthRepository {
	return &healthRepository{
		sharedRepository: shared,
	}
}

func (a *healthRepository) IsHealthy(ctx context.Context) bool {
	// Readiness reads on the raw pool so a shared bucket at its cap does not take the process out of service.
	_, err := a.queries.Health(ctx, a.pool.Unwrap())

	if err != nil { //nolint:staticcheck
		a.l.Err(err).Ctx(ctx).Msg("health check failed")
		return false
	}

	return true
}

func (a *healthRepository) PgStat() *pgxpool.Stat {
	stat := a.pool.Stat()
	return stat
}
