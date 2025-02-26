package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type apiTokenRepository struct {
	*sharedRepository

	cache cache.Cacheable
}

func NewAPITokenRepository(shared *sharedRepository, cache cache.Cacheable) repository.APITokenRepository {
	return &apiTokenRepository{
		sharedRepository: shared,
		cache:            cache,
	}
}

func (a *apiTokenRepository) RevokeAPIToken(ctx context.Context, id string) error {
	return a.queries.RevokeAPIToken(ctx, a.pool, sqlchelpers.UUIDFromStr(id))
}

func (a *apiTokenRepository) ListAPITokensByTenant(ctx context.Context, tenantId string) ([]*dbsqlc.APIToken, error) {
	return a.queries.ListAPITokensByTenant(ctx, a.pool, sqlchelpers.UUIDFromStr(tenantId))
}

func (a *apiTokenRepository) CreateAPIToken(ctx context.Context, opts *repository.CreateAPITokenOpts) (*dbsqlc.APIToken, error) {
	if err := a.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateAPITokenParams{
		ID:        sqlchelpers.UUIDFromStr(opts.ID),
		Expiresat: sqlchelpers.TimestampFromTime(opts.ExpiresAt),
		Internal:  sqlchelpers.BoolFromBoolean(opts.Internal),
	}

	if opts.TenantId != nil {
		createParams.TenantId = sqlchelpers.UUIDFromStr(*opts.TenantId)
	}

	if opts.Name != nil {
		createParams.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	return a.queries.CreateAPIToken(ctx, a.pool, createParams)
}

func (a *apiTokenRepository) GetAPITokenById(ctx context.Context, id string) (*dbsqlc.APIToken, error) {
	return cache.MakeCacheable[dbsqlc.APIToken](a.cache, id, func() (*dbsqlc.APIToken, error) {
		return a.queries.GetAPITokenById(ctx, a.pool, sqlchelpers.UUIDFromStr(id))
	})
}

func (a *apiTokenRepository) DeleteAPIToken(ctx context.Context, tenantId, id string) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, a.pool, a.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	err = a.queries.DeleteAPIToken(ctx, tx, dbsqlc.DeleteAPITokenParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(id),
	})

	if err != nil {
		return err
	}

	return commit(ctx)
}
