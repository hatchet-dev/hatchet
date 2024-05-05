package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/cache"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type tenantAPIRepository struct {
	client *db.PrismaClient
	v      validator.Validator
	cache  cache.Cacheable
}

func NewTenantAPIRepository(client *db.PrismaClient, v validator.Validator, cache cache.Cacheable) repository.TenantAPIRepository {
	return &tenantAPIRepository{
		client: client,
		v:      v,
		cache:  cache,
	}
}

func (r *tenantAPIRepository) CreateTenant(opts *repository.CreateTenantOpts) (*db.TenantModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.Tenant.CreateOne(
		db.Tenant.Name.Set(opts.Name),
		db.Tenant.Slug.Set(opts.Slug),
		db.Tenant.ID.SetIfPresent(opts.ID),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) UpdateTenant(id string, opts *repository.UpdateTenantOpts) (*db.TenantModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.Tenant.FindUnique(
		db.Tenant.ID.Equals(id),
	).Update(
		db.Tenant.AnalyticsOptOut.SetIfPresent(opts.AnalyticsOptOut),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantByID(id string) (*db.TenantModel, error) {
	return cache.MakeCacheable[db.TenantModel](r.cache, id, func() (*db.TenantModel, error) {
		return r.client.Tenant.FindUnique(
			db.Tenant.ID.Equals(id),
		).Exec(context.Background())
	})
}

func (r *tenantAPIRepository) GetTenantBySlug(slug string) (*db.TenantModel, error) {
	return r.client.Tenant.FindUnique(
		db.Tenant.Slug.Equals(slug),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) CreateTenantMember(tenantId string, opts *repository.CreateTenantMemberOpts) (*db.TenantMemberModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.TenantMember.CreateOne(
		db.TenantMember.Tenant.Link(db.Tenant.ID.Equals(tenantId)),
		db.TenantMember.User.Link(db.User.ID.Equals(opts.UserId)),
		db.TenantMember.Role.Set(db.TenantMemberRole(opts.Role)),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByID(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByUserID(tenantId string, userId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.TenantIDUserID(
			db.TenantMember.TenantID.Equals(tenantId),
			db.TenantMember.UserID.Equals(userId),
		),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) ListTenantMembers(tenantId string) ([]db.TenantMemberModel, error) {
	return r.client.TenantMember.FindMany(
		db.TenantMember.TenantID.Equals(tenantId),
	).With(
		db.TenantMember.User.Fetch(),
		db.TenantMember.Tenant.Fetch(),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByEmail(tenantId string, email string) (*db.TenantMemberModel, error) {
	user, err := r.client.User.FindUnique(
		db.User.Email.Equals(email),
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return r.client.TenantMember.FindUnique(
		db.TenantMember.TenantIDUserID(
			db.TenantMember.TenantID.Equals(tenantId),
			db.TenantMember.UserID.Equals(user.ID),
		),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) UpdateTenantMember(memberId string, opts *repository.UpdateTenantMemberOpts) (*db.TenantMemberModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.TenantMemberSetParam{}

	if opts.Role != nil {
		params = append(params, db.TenantMember.Role.Set(db.TenantMemberRole(*opts.Role)))
	}

	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Update(
		params...,
	).Exec(context.Background())
}

func (r *tenantAPIRepository) DeleteTenantMember(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Delete().Exec(context.Background())
}

type tenantEngineRepository struct {
	cache   cache.Cacheable
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewTenantEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cache cache.Cacheable) repository.TenantEngineRepository {
	queries := dbsqlc.New()

	return &tenantEngineRepository{
		cache:   cache,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (r *tenantEngineRepository) ListTenants(ctx context.Context) ([]*dbsqlc.Tenant, error) {
	return r.queries.ListTenants(ctx, r.pool)
}

func (r *tenantEngineRepository) GetTenantByID(ctx context.Context, tenantId string) (*dbsqlc.Tenant, error) {
	return cache.MakeCacheable[dbsqlc.Tenant](r.cache, tenantId, func() (*dbsqlc.Tenant, error) {
		return r.queries.GetTenantByID(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))
	})
}
