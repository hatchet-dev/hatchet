package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type healthAPIRepository struct {
	*sharedRepository
}

func NewHealthAPIRepository(shared *sharedRepository) repository.HealthRepository {
	return &healthAPIRepository{
		sharedRepository: shared,
	}
}

func (a *healthAPIRepository) IsHealthy(ctx context.Context) bool {
	_, err := a.queries.Health(ctx, a.pool)

	if err != nil { //nolint:gosimple
		a.l.Err(err).Msg("health check failed")
		return false
	}

	return true
}

func (a *healthAPIRepository) PgStat() *pgxpool.Stat {
	stat := a.pool.Stat()
	return stat
}

type healthEngineRepository struct {
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
	l       *logger.Logger
}

func NewHealthEngineRepository(pool *pgxpool.Pool, l *logger.Logger) repository.HealthRepository {
	queries := dbsqlc.New()

	return &healthEngineRepository{
		queries: queries,
		pool:    pool,
		l:       l,
	}
}

func (a *healthEngineRepository) IsHealthy(ctx context.Context) bool {
	_, err := a.queries.Health(ctx, a.pool)

	if err != nil {
		a.l.Err(err).Msg("health check failed")
		return false
	}

	return true
}

func (a *healthEngineRepository) PgStat() *pgxpool.Stat {
	stat := a.pool.Stat()
	return stat
}
