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
	_, err := a.queries.Health(ctx, a.pool)

	if err != nil { //nolint:gosimple
		a.l.Err(err).Msg("health check failed")
		return false
	}

	return true
}

func (a *healthRepository) PgStat() *pgxpool.Stat {
	stat := a.pool.Stat()
	return stat
}
