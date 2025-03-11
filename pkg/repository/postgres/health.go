package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

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

func (a *healthAPIRepository) IsHealthy() bool {
	_, err := a.queries.Health(context.Background(), a.pool)

	if err != nil { //nolint:gosimple
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
}

func NewHealthEngineRepository(pool *pgxpool.Pool) repository.HealthRepository {
	queries := dbsqlc.New()

	return &healthEngineRepository{
		queries: queries,
		pool:    pool,
	}
}

func (a *healthEngineRepository) IsHealthy() bool {
	_, err := a.queries.Health(context.Background(), a.pool)

	if err != nil { //nolint:gosimple
		return false
	}

	return true
}

func (a *healthEngineRepository) PgStat() *pgxpool.Stat {
	stat := a.pool.Stat()
	return stat
}
