package prisma

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type webhookWorkerEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWebhookWorkerEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WebhookWorkerEngineRepository {
	queries := dbsqlc.New()

	return &webhookWorkerEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *webhookWorkerEngineRepository) GetAllWebhookWorkers(ctx context.Context, tenantId string) ([]*dbsqlc.WebhookWorker, error) {
	rows, err := r.queries.GetAllWebhookWorkers(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))
	if err != nil {
		return nil, fmt.Errorf("could not get all webhook workers: %w", err)
	}

	return rows, nil
}
