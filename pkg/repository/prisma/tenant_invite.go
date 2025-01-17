package prisma

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantInviteRepository struct {
	client *db.PrismaClient
	v      validator.Validator
	l      *zerolog.Logger
}

func NewTenantInviteRepository(client *db.PrismaClient, v validator.Validator, l *zerolog.Logger) repository.TenantInviteRepository {
	return &tenantInviteRepository{
		client: client,
		v:      v,
		l:      l,
	}
}

func (r *tenantInviteRepository) CreateTenantInvite(tenantId string, opts *repository.CreateTenantInviteOpts) (*db.TenantInviteLinkModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}
	if opts.MaxPending != 0 {
		invites, err := r.client.TenantInviteLink.FindMany(
			db.TenantInviteLink.Status.Equals(db.InviteLinkStatusPending),
			db.TenantInviteLink.Expires.Gt(time.Now()),
			db.TenantInviteLink.TenantID.Equals(tenantId)).Exec(context.Background())

		if err != nil {
			r.l.Error().Err(err).Msg("error counting pending invites")
			return nil, err
		}

		if len(invites) >= opts.MaxPending {
			r.l.Error().Msg("max pending invites reached")
			return nil, fmt.Errorf("max pending invites reached")
		}
	}

	til, err := r.client.TenantInviteLink.FindMany(
		db.TenantInviteLink.InviteeEmail.Equals(opts.InviteeEmail),
		db.TenantInviteLink.TenantID.Equals(tenantId),
		db.TenantInviteLink.Status.Equals(db.InviteLinkStatusPending),
		db.TenantInviteLink.Expires.Gt(time.Now()),
		db.TenantInviteLink.Role.Equals(db.TenantMemberRole(opts.Role)),
	).Exec(context.Background())

	if err != nil {
		r.l.Error().Err(err).Msg("error checking for existing invite")
		return nil, err
	}

	if len(til) > 0 {
		r.l.Error().Msg("invite already exists")
		return nil, fmt.Errorf("invite already exists")
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
