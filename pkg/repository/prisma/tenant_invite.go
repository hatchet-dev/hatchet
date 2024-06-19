package prisma

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantInviteRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewTenantInviteRepository(client *db.PrismaClient, v validator.Validator) repository.TenantInviteRepository {
	return &tenantInviteRepository{
		client: client,
		v:      v,
	}
}

func (r *tenantInviteRepository) CreateTenantInvite(tenantId string, opts *repository.CreateTenantInviteOpts) (*db.TenantInviteLinkModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.TenantInviteLink.CreateOne(
		db.TenantInviteLink.Tenant.Link(db.Tenant.ID.Equals(tenantId)),
		db.TenantInviteLink.InviterEmail.Set(opts.InviterEmail),
		db.TenantInviteLink.InviteeEmail.Set(opts.InviteeEmail),
		db.TenantInviteLink.Expires.Set(opts.ExpiresAt),
		db.TenantInviteLink.Role.Set(db.TenantMemberRole(opts.Role)),
	).Exec(context.Background())
}

func (r *tenantInviteRepository) GetTenantInvite(id string) (*db.TenantInviteLinkModel, error) {
	return r.client.TenantInviteLink.FindUnique(
		db.TenantInviteLink.ID.Equals(id),
	).Exec(context.Background())
}

func (r *tenantInviteRepository) ListTenantInvitesByEmail(email string) ([]db.TenantInviteLinkModel, error) {
	return r.client.TenantInviteLink.FindMany(
		db.TenantInviteLink.InviteeEmail.Equals(email),
		db.TenantInviteLink.Status.Equals(db.InviteLinkStatusPending),
		db.TenantInviteLink.Expires.Gt(time.Now()),
	).With(
		db.TenantInviteLink.Tenant.Fetch(),
	).Exec(context.Background())
}

func (r *tenantInviteRepository) ListTenantInvitesByTenantId(tenantId string, opts *repository.ListTenantInvitesOpts) ([]db.TenantInviteLinkModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.TenantInviteLinkWhereParam{
		db.TenantInviteLink.TenantID.Equals(tenantId),
	}

	if opts.Status != nil {
		params = append(params, db.TenantInviteLink.Status.Equals(db.InviteLinkStatus(*opts.Status)))
	}

	if opts.Expired != nil {
		if *opts.Expired {
			params = append(params, db.TenantInviteLink.Expires.Lt(time.Now()))
		} else {
			params = append(params, db.TenantInviteLink.Expires.Gt(time.Now()))
		}
	}

	return r.client.TenantInviteLink.FindMany(
		params...,
	).Exec(context.Background())
}

func (r *tenantInviteRepository) UpdateTenantInvite(id string, opts *repository.UpdateTenantInviteOpts) (*db.TenantInviteLinkModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	optionals := []db.TenantInviteLinkSetParam{}

	if opts.Role != nil {
		optionals = append(optionals, db.TenantInviteLink.Role.Set(db.TenantMemberRole(*opts.Role)))
	}

	if opts.Status != nil {
		optionals = append(optionals, db.TenantInviteLink.Status.Set(db.InviteLinkStatus(*opts.Status)))
	}

	return r.client.TenantInviteLink.FindUnique(
		db.TenantInviteLink.ID.Equals(id),
	).Update(
		optionals...,
	).Exec(context.Background())
}

func (r *tenantInviteRepository) DeleteTenantInvite(id string) error {
	_, err := r.client.TenantInviteLink.FindUnique(
		db.TenantInviteLink.ID.Equals(id),
	).Delete().Exec(context.Background())

	return err
}
