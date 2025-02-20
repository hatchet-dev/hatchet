package prisma

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type apiTokenRepository struct {
	client *db.PrismaClient
	v      validator.Validator
	cache  cache.Cacheable
}

func NewAPITokenRepository(client *db.PrismaClient, v validator.Validator, cache cache.Cacheable) repository.APITokenRepository {
	return &apiTokenRepository{
		client: client,
		v:      v,
		cache:  cache,
	}
}

func (a *apiTokenRepository) GetAPITokenById(id string) (*db.APITokenModel, error) {
	return cache.MakeCacheable[db.APITokenModel](a.cache, id, func() (*db.APITokenModel, error) {
		return a.client.APIToken.FindUnique(
			db.APIToken.ID.Equals(id),
		).Exec(context.Background())
	})
}

func (a *apiTokenRepository) CreateAPIToken(opts *repository.CreateAPITokenOpts) (*db.APITokenModel, error) {
	if err := a.v.Validate(opts); err != nil {
		return nil, err
	}

	optionals := []db.APITokenSetParam{
		db.APIToken.ID.Set(opts.ID),
		db.APIToken.ExpiresAt.Set(opts.ExpiresAt),
		db.APIToken.Internal.Set(opts.Internal),
	}

	if opts.TenantId != nil {
		optionals = append(optionals, db.APIToken.Tenant.Link(
			db.Tenant.ID.Equals(*opts.TenantId),
		))
	}

	if opts.Name != nil {
		optionals = append(optionals, db.APIToken.Name.Set(*opts.Name))
	}

	return a.client.APIToken.CreateOne(
		optionals...,
	).Exec(context.Background())
}

func (a *apiTokenRepository) RevokeAPIToken(id string) error {
	_, err := a.client.APIToken.FindUnique(
		db.APIToken.ID.Equals(id),
	).Update(
		db.APIToken.ExpiresAt.Set(time.Now().Add(-1*time.Second)),
		db.APIToken.Revoked.Set(true),
	).Exec(context.Background())

	return err
}

func (a *apiTokenRepository) ListAPITokensByTenant(tenantId string) ([]db.APITokenModel, error) {
	return a.client.APIToken.FindMany(
		db.APIToken.TenantID.Equals(tenantId),
		db.APIToken.Revoked.Equals(false),
		db.APIToken.Internal.Equals(false),
	).Exec(context.Background())
}

type engineTokenRepository struct {
	cache   cache.Cacheable
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewEngineTokenRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cache cache.Cacheable) repository.EngineTokenRepository {
	queries := dbsqlc.New()

	return &engineTokenRepository{
		cache:   cache,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (a *engineTokenRepository) CreateAPIToken(ctx context.Context, opts *repository.CreateAPITokenOpts) (*dbsqlc.APIToken, error) {
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

func (a *engineTokenRepository) GetAPITokenById(ctx context.Context, id string) (*dbsqlc.APIToken, error) {
	return cache.MakeCacheable[dbsqlc.APIToken](a.cache, id, func() (*dbsqlc.APIToken, error) {
		return a.queries.GetAPITokenById(ctx, a.pool, sqlchelpers.UUIDFromStr(id))
	})
}
