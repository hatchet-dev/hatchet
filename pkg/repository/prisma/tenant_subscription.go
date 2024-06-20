package prisma

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantSubscriptionRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	config  *server.ConfigFileRuntime
}

func NewTenantSubscriptionRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, s *server.ConfigFileRuntime) repository.TenantSubscriptionRepository {
	queries := dbsqlc.New()

	return &tenantSubscriptionRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
		config:  s,
	}
}

func (r *tenantSubscriptionRepository) GetSubscription(ctx context.Context, tenantId string) (*dbsqlc.TenantSubscription, error) {

	subscription, err := r.queries.GetTenantSubscription(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return subscription, nil
}

func (r *tenantSubscriptionRepository) UpsertSubscription(ctx context.Context, opts dbsqlc.UpsertTenantSubscriptionParams) (bool, *dbsqlc.TenantSubscription, error) {

	sub, err := r.queries.UpsertTenantSubscription(ctx, r.pool, opts)

	if err != nil {
		return false, nil, err
	}

	return sub.Status == dbsqlc.TenantSubscriptionStatusActive, sub, nil
}
