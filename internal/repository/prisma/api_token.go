package prisma

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/cache"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
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
	).Exec(context.Background())
}
