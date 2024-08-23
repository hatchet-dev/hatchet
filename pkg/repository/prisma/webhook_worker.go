package prisma

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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

func (r *webhookWorkerEngineRepository) ListWebhookWorkersByPartitionId(ctx context.Context, partitionId string) ([]*dbsqlc.WebhookWorker, error) {
	if partitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListWebhookWorkersByPartitionId(ctx, r.pool, partitionId)
}

func (r *webhookWorkerEngineRepository) ListActiveWebhookWorkers(ctx context.Context, tenantId string) ([]*dbsqlc.WebhookWorker, error) {
	return r.queries.ListActiveWebhookWorkers(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))
}

func (r *webhookWorkerEngineRepository) ListWebhookWorkerRequests(ctx context.Context, webhookWorkerId string) ([]*dbsqlc.WebhookWorkerRequest, error) {
	return r.queries.ListWebhookWorkerRequests(ctx, r.pool, sqlchelpers.UUIDFromStr(webhookWorkerId))
}

func (r *webhookWorkerEngineRepository) InsertWebhookWorkerRequest(ctx context.Context, webhookWorkerId string, method string, statusCode int32) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return r.queries.InsertWebhookWorkerRequest(ctx, r.pool, dbsqlc.InsertWebhookWorkerRequestParams{
		Webhookworkerid: sqlchelpers.UUIDFromStr(webhookWorkerId),
		Method:          dbsqlc.WebhookWorkerRequestMethod(method),
		Statuscode:      statusCode,
	})
}

func (r *webhookWorkerEngineRepository) CreateWebhookWorker(ctx context.Context, opts *repository.CreateWebhookWorkerOpts) (*dbsqlc.WebhookWorker, error) {

	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.CreateWebhookWorkerParams{
		Tenantid: sqlchelpers.UUIDFromStr(opts.TenantId),
		Name:     opts.Name,
		Secret:   opts.Secret,
		Url:      opts.URL,
	}

	if opts.Deleted != nil {
		params.Deleted = sqlchelpers.BoolFromBoolean(*opts.Deleted)
	}

	if opts.TokenID != nil {
		params.TokenId = sqlchelpers.UUIDFromStr(*opts.TokenID)
	}

	if opts.TokenValue != nil {
		params.TokenValue = sqlchelpers.TextFromStr(*opts.TokenValue)
	}

	worker, err := r.queries.CreateWebhookWorker(ctx, r.pool, params)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return nil, repository.ErrDuplicateKey
		}
		return nil, err
	}

	return worker, nil
}

func (r *webhookWorkerEngineRepository) UpdateWebhookWorkerToken(ctx context.Context, id string, tenantId string, opts *repository.UpdateWebhookWorkerTokenOpts) (*dbsqlc.WebhookWorker, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateWebhookWorkerTokenParams{
		ID:       sqlchelpers.UUIDFromStr(id),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.TokenID != nil {
		params.TokenId = sqlchelpers.UUIDFromStr(*opts.TokenID)
	}

	if opts.TokenValue != nil {
		params.TokenValue = sqlchelpers.TextFromStr(*opts.TokenValue)
	}

	worker, err := r.queries.UpdateWebhookWorkerToken(ctx, r.pool, params)

	if err != nil {
		return nil, err
	}

	return worker, nil
}

func (r *webhookWorkerEngineRepository) SoftDeleteWebhookWorker(ctx context.Context, id string, tenantId string) error {
	return r.queries.SoftDeleteWebhookWorker(ctx, r.pool, dbsqlc.SoftDeleteWebhookWorkerParams{
		ID:       sqlchelpers.UUIDFromStr(id),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (r *webhookWorkerEngineRepository) HardDeleteWebhookWorker(ctx context.Context, id string, tenantId string) error {
	return r.queries.HardDeleteWebhookWorker(ctx, r.pool, dbsqlc.HardDeleteWebhookWorkerParams{
		ID:       sqlchelpers.UUIDFromStr(id),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}
