package v2

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type workerRepo interface {
	ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]pgtype.UUID, error)
}

type workerDbQueries struct {
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool

	l *zerolog.Logger
}

func newWorkerDbQueries(queries *dbsqlc.Queries, pool *pgxpool.Pool, l *zerolog.Logger) *workerDbQueries {
	return &workerDbQueries{
		queries: queries,
		pool:    pool,
		l:       l,
	}
}

func (d *workerDbQueries) ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]pgtype.UUID, error) {
	return d.queries.ListActiveWorkerIds(ctx, d.pool, tenantId)
}
