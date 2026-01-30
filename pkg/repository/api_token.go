package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateAPITokenOpts struct {
	// The id of the token
	ID string `validate:"required,uuid"`

	// When the token expires
	ExpiresAt time.Time

	// (optional) A tenant ID for this API token
	TenantId *string `validate:"omitempty,uuid"`

	// (optional) A name for this API token
	Name *string `validate:"omitempty,max=255"`

	Internal bool
}

type APITokenGenerator func(ctx context.Context, tenantId, name string, internal bool, expires *time.Time) (string, error)

type APITokenRepository interface {
	CreateAPIToken(ctx context.Context, opts *CreateAPITokenOpts) (*sqlcv1.APIToken, error)
	GetAPITokenById(ctx context.Context, id string) (*sqlcv1.APIToken, error)
	ListAPITokensByTenant(ctx context.Context, tenantId string) ([]*sqlcv1.APIToken, error)
	RevokeAPIToken(ctx context.Context, id string) error
	DeleteAPIToken(ctx context.Context, tenantId, id string) error
}

type apiTokenRepository struct {
	*sharedRepository

	c cache.Cacheable
}

func newAPITokenRepository(shared *sharedRepository, cacheDuration time.Duration) APITokenRepository {
	c := cache.New(cacheDuration)

	return &apiTokenRepository{
		sharedRepository: shared,
		c:                c,
	}
}

func (a *apiTokenRepository) RevokeAPIToken(ctx context.Context, id string) error {
	return a.queries.RevokeAPIToken(ctx, a.pool, uuid.MustParse(id))
}

func (a *apiTokenRepository) ListAPITokensByTenant(ctx context.Context, tenantId string) ([]*sqlcv1.APIToken, error) {
	return a.queries.ListAPITokensByTenant(ctx, a.pool, uuid.MustParse(tenantId))
}

func (a *apiTokenRepository) CreateAPIToken(ctx context.Context, opts *CreateAPITokenOpts) (*sqlcv1.APIToken, error) {
	if err := a.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := sqlcv1.CreateAPITokenParams{
		ID:        uuid.MustParse(opts.ID),
		Expiresat: sqlchelpers.TimestampFromTime(opts.ExpiresAt),
		Internal:  sqlchelpers.BoolFromBoolean(opts.Internal),
	}

	if opts.TenantId != nil {
		createParams.TenantId = uuid.MustParse(*opts.TenantId)
	}

	if opts.Name != nil {
		createParams.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	return a.queries.CreateAPIToken(ctx, a.pool, createParams)
}

func (a *apiTokenRepository) GetAPITokenById(ctx context.Context, id string) (*sqlcv1.APIToken, error) {
	return cache.MakeCacheable[sqlcv1.APIToken](a.c, id, func() (*sqlcv1.APIToken, error) {
		return a.queries.GetAPITokenById(ctx, a.pool, uuid.MustParse(id))
	})
}

func (a *apiTokenRepository) DeleteAPIToken(ctx context.Context, tenantId, id string) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, a.pool, a.l)

	if err != nil {
		return err
	}

	defer rollback()

	err = a.queries.DeleteAPIToken(ctx, tx, sqlcv1.DeleteAPITokenParams{
		Tenantid: uuid.MustParse(tenantId),
		ID:       uuid.MustParse(id),
	})

	if err != nil {
		return err
	}

	return commit(ctx)
}
