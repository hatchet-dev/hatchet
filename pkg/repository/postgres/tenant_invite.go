package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type tenantInviteRepository struct {
	*sharedRepository
}

func NewTenantInviteRepository(shared *sharedRepository) repository.TenantInviteRepository {
	return &tenantInviteRepository{
		sharedRepository: shared,
	}
}

func (r *tenantInviteRepository) CreateTenantInvite(ctx context.Context, tenantId string, opts *repository.CreateTenantInviteOpts) (*dbsqlc.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if opts.MaxPending != 0 {
		invites, err := r.queries.CountActiveInvites(
			ctx,
			tx,
			sqlchelpers.UUIDFromStr(tenantId),
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
		dbsqlc.GetExistingInviteParams{
			Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
			Inviteeemail: opts.InviteeEmail,
		},
	)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	invite, err := r.queries.CreateTenantInvite(
		ctx,
		tx,
		dbsqlc.CreateTenantInviteParams{
			Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
			Inviteremail: opts.InviterEmail,
			Inviteeemail: opts.InviteeEmail,
			Expires:      sqlchelpers.TimestampFromTime(opts.ExpiresAt),
			Role:         dbsqlc.TenantMemberRole(opts.Role),
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

func (r *tenantInviteRepository) GetTenantInvite(ctx context.Context, id string) (*dbsqlc.TenantInviteLink, error) {
	return r.queries.GetInviteById(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(id),
	)
}

func (r *tenantInviteRepository) ListTenantInvitesByEmail(ctx context.Context, email string) ([]*dbsqlc.TenantInviteLink, error) {
	return r.queries.ListTenantInvitesByEmail(
		ctx,
		r.pool,
		email,
	)
}

func (r *tenantInviteRepository) ListTenantInvitesByTenantId(ctx context.Context, tenantId string, opts *repository.ListTenantInvitesOpts) ([]*dbsqlc.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.ListInvitesByTenantIdParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.Status != nil {
		params.Status = dbsqlc.NullInviteLinkStatus{
			InviteLinkStatus: dbsqlc.InviteLinkStatus(*opts.Status),
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

func (r *tenantInviteRepository) UpdateTenantInvite(ctx context.Context, id string, opts *repository.UpdateTenantInviteOpts) (*dbsqlc.TenantInviteLink, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateTenantInviteParams{
		ID: sqlchelpers.UUIDFromStr(id),
	}

	if opts.Role != nil {
		params.Role = dbsqlc.NullTenantMemberRole{
			TenantMemberRole: dbsqlc.TenantMemberRole(*opts.Role),
			Valid:            true,
		}
	}

	if opts.Status != nil {
		params.Status = dbsqlc.NullInviteLinkStatus{
			InviteLinkStatus: dbsqlc.InviteLinkStatus(*opts.Status),
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
		sqlchelpers.UUIDFromStr(id),
	)
}
