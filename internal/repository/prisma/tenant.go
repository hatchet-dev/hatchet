package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type tenantRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewTenantRepository(client *db.PrismaClient, v validator.Validator) repository.TenantRepository {
	return &tenantRepository{
		client: client,
		v:      v,
	}
}

func (r *tenantRepository) CreateTenant(opts *repository.CreateTenantOpts) (*db.TenantModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.Tenant.CreateOne(
		db.Tenant.Name.Set(opts.Name),
		db.Tenant.Slug.Set(opts.Slug),
	).Exec(context.Background())
}

func (r *tenantRepository) ListTenants() ([]db.TenantModel, error) {
	return r.client.Tenant.FindMany().Exec(context.Background())
}

func (r *tenantRepository) GetTenantByID(id string) (*db.TenantModel, error) {
	return r.client.Tenant.FindUnique(
		db.Tenant.ID.Equals(id),
	).Exec(context.Background())
}

func (r *tenantRepository) GetTenantBySlug(slug string) (*db.TenantModel, error) {
	return r.client.Tenant.FindUnique(
		db.Tenant.Slug.Equals(slug),
	).Exec(context.Background())
}

func (r *tenantRepository) CreateTenantMember(tenantId string, opts *repository.CreateTenantMemberOpts) (*db.TenantMemberModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.TenantMember.CreateOne(
		db.TenantMember.Tenant.Link(db.Tenant.ID.Equals(tenantId)),
		db.TenantMember.User.Link(db.User.ID.Equals(opts.UserId)),
		db.TenantMember.Role.Set(db.TenantMemberRole(opts.Role)),
	).Exec(context.Background())
}

func (r *tenantRepository) GetTenantMemberByID(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Exec(context.Background())
}

func (r *tenantRepository) GetTenantMemberByUserID(tenantId string, userId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.TenantIDUserID(
			db.TenantMember.TenantID.Equals(tenantId),
			db.TenantMember.UserID.Equals(userId),
		),
	).Exec(context.Background())
}

func (r *tenantRepository) UpdateTenantMember(memberId string, opts *repository.UpdateTenantMemberOpts) (*db.TenantMemberModel, error) {
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

func (r *tenantRepository) DeleteTenantMember(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Delete().Exec(context.Background())
}
