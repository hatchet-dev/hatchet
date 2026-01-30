package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateTenantInviteOpts struct {
	// (required) the invitee email
	InviteeEmail string `validate:"required,email"`

	// (required) the inviter email
	InviterEmail string `validate:"required,email"`

	// (required) when the invite expires
	ExpiresAt time.Time `validate:"required,future"`

	// (required) the role of the invitee
	Role string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`

	// (optional) the maximum number pending of invites the inviter can have

	MaxPending int `validate:"omitempty"`
}

type UpdateTenantInviteOpts struct {
	Status *string `validate:"omitempty,oneof=ACCEPTED REJECTED"`

	// (optional) the role of the invitee
	Role *string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`
}

type ListTenantInvitesOpts struct {
	// (optional) the status of the invite
	Status *string `validate:"omitempty,oneof=PENDING ACCEPTED REJECTED"`

	// (optional) whether the invite has expired
	Expired *bool `validate:"omitempty"`
}

type TenantInviteRepository interface {
	// CreateTenantInvite creates a new tenant invite with the given options
	CreateTenantInvite(ctx context.Context, tenantId string, opts *CreateTenantInviteOpts) (*sqlcv1.TenantInviteLink, error)

	// GetTenantInvite returns the tenant invite with the given id
	GetTenantInvite(ctx context.Context, id string) (*sqlcv1.TenantInviteLink, error)

	// ListTenantInvitesByEmail returns the list of tenant invites for the given invitee email for invites
	// which are not expired
	ListTenantInvitesByEmail(ctx context.Context, email string) ([]*sqlcv1.ListTenantInvitesByEmailRow, error)

	// ListTenantInvitesByTenantId returns the list of tenant invites for the given tenant id
	ListTenantInvitesByTenantId(ctx context.Context, tenantId string, opts *ListTenantInvitesOpts) ([]*sqlcv1.TenantInviteLink, error)

	// UpdateTenantInvite updates the tenant invite with the given id
	UpdateTenantInvite(ctx context.Context, id string, opts *UpdateTenantInviteOpts) (*sqlcv1.TenantInviteLink, error)

	// DeleteTenantInvite deletes the tenant invite with the given id
	DeleteTenantInvite(ctx context.Context, id string) error
}

type tenantInviteRepository struct {
	*sharedRepository
}

func newTenantInviteRepository(shared *sharedRepository) TenantInviteRepository {
	return &tenantInviteRepository{
		sharedRepository: shared,
	}
}

func (r *tenantInviteRepository) CreateTenantInvite(ctx context.Context, tenantId string, opts *CreateTenantInviteOpts) (*sqlcv1.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if opts.MaxPending != 0 {
		invites, err := r.queries.CountActiveInvites(
			ctx,
			tx,
			uuid.MustParse(tenantId),
		)

		if err != nil {
			r.l.Error().Err(err).Msg("error counting pending invites")
			return nil, err
		}

		if invites >= int64(opts.MaxPending) {
			r.l.Error().Msg("max pending invites reached")
			return nil, fmt.Errorf("max pending invites reached")
		}
	}

	_, err = r.queries.GetExistingInvite(
		ctx,
		tx,
		sqlcv1.GetExistingInviteParams{
			Tenantid:     uuid.MustParse(tenantId),
			Inviteeemail: opts.InviteeEmail,
		},
	)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	invite, err := r.queries.CreateTenantInvite(
		ctx,
		tx,
		sqlcv1.CreateTenantInviteParams{
			Tenantid:     uuid.MustParse(tenantId),
			Inviteremail: opts.InviterEmail,
			Inviteeemail: opts.InviteeEmail,
			Expires:      sqlchelpers.TimestampFromTime(opts.ExpiresAt),
			Role:         sqlcv1.TenantMemberRole(opts.Role),
		},
	)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return invite, nil
}

func (r *tenantInviteRepository) GetTenantInvite(ctx context.Context, id string) (*sqlcv1.TenantInviteLink, error) {
	return r.queries.GetInviteById(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}

func (r *tenantInviteRepository) ListTenantInvitesByEmail(ctx context.Context, email string) ([]*sqlcv1.ListTenantInvitesByEmailRow, error) {
	return r.queries.ListTenantInvitesByEmail(
		ctx,
		r.pool,
		email,
	)
}

func (r *tenantInviteRepository) ListTenantInvitesByTenantId(ctx context.Context, tenantId string, opts *ListTenantInvitesOpts) ([]*sqlcv1.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.ListInvitesByTenantIdParams{
		Tenantid: uuid.MustParse(tenantId),
	}

	if opts.Status != nil {
		params.Status = sqlcv1.NullInviteLinkStatus{
			InviteLinkStatus: sqlcv1.InviteLinkStatus(*opts.Status),
			Valid:            true,
		}
	}

	if opts.Expired != nil {
		params.Expired = pgtype.Bool{
			Bool:  *opts.Expired,
			Valid: true,
		}
	}

	return r.queries.ListInvitesByTenantId(
		ctx,
		r.pool,
		params,
	)
}

func (r *tenantInviteRepository) UpdateTenantInvite(ctx context.Context, id string, opts *UpdateTenantInviteOpts) (*sqlcv1.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateTenantInviteParams{
		ID: uuid.MustParse(id),
	}

	if opts.Role != nil {
		params.Role = sqlcv1.NullTenantMemberRole{
			TenantMemberRole: sqlcv1.TenantMemberRole(*opts.Role),
			Valid:            true,
		}
	}

	if opts.Status != nil {
		params.Status = sqlcv1.NullInviteLinkStatus{
			InviteLinkStatus: sqlcv1.InviteLinkStatus(*opts.Status),
			Valid:            true,
		}
	}

	return r.queries.UpdateTenantInvite(
		ctx,
		r.pool,
		params,
	)
}

func (r *tenantInviteRepository) DeleteTenantInvite(ctx context.Context, id string) error {
	return r.queries.DeleteTenantInvite(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}
