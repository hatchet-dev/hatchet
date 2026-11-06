package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type RevokeSupportGrantOpts struct {
	GrantID  uuid.UUID
	TenantID uuid.UUID

	// ExpiresAt is when the grant would have ended on its own; the row is useless after that
	ExpiresAt time.Time
}

// SupportGrantRepository records support grants the control plane revoked early so a still-valid
// support token is refused before it expires. Staff have no rows on the shard, so this is the only
// support state a shard holds.
type SupportGrantRepository interface {
	Revoke(ctx context.Context, opts RevokeSupportGrantOpts) error
	IsRevoked(ctx context.Context, grantID uuid.UUID) (bool, error)
}

type supportGrantRepository struct {
	*sharedRepository
}

func newSupportGrantRepository(shared *sharedRepository) SupportGrantRepository {
	return &supportGrantRepository{sharedRepository: shared}
}

func (r *supportGrantRepository) Revoke(ctx context.Context, opts RevokeSupportGrantOpts) error {
	db := r.pool.ForTenant(opts.TenantID)

	// expired rows are swept on the way in: revocations are rare, so a ticker would be overkill
	if err := r.queries.DeleteExpiredSupportGrantRevocations(ctx, db); err != nil {
		return err
	}

	return r.queries.RevokeSupportGrant(ctx, db, sqlcv1.RevokeSupportGrantParams{
		Grantid:   opts.GrantID,
		Tenantid:  opts.TenantID,
		Expiresat: sqlchelpers.TimestamptzFromTime(opts.ExpiresAt),
	})
}

// IsRevoked reads the table directly: the per-process cache would hide a revocation for its TTL
func (r *supportGrantRepository) IsRevoked(ctx context.Context, grantID uuid.UUID) (bool, error) {
	return r.queries.IsSupportGrantRevoked(ctx, r.pool.ForShared(), grantID)
}
