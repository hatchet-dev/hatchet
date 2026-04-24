package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// SyncRepository provides database access for syncing tenants, users,
// tenant memberships, and invites from an external source.
//
// Callers manage their own transactions via Pool():
//
//	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, repo.Pool(), l)
//	defer rollback()
//	repo.SyncUpsertTenant(ctx, tx, ...)
//	repo.SyncUpsertUser(ctx, tx, ...)
//	commit(ctx)
//
// All queries require explicit id, createdAt, and updatedAt values —
// no gen_random_uuid() or NOW() defaults are called server-side.
type SyncRepository interface {
	// Pool returns the connection pool so callers can start their own transactions.
	Pool() *pgxpool.Pool

	SyncUpsertTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantParams) (*sqlcv1.Tenant, error)
	SyncUpdateTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantParams) (*sqlcv1.Tenant, error)
	SyncSoftDeleteTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncSoftDeleteTenantParams) error

	SyncUpsertUser(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertUserParams) (*sqlcv1.User, error)

	SyncUpsertTenantInvite(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantInviteParams) (*sqlcv1.TenantInviteLink, error)
	SyncUpdateTenantInvite(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantInviteParams) (*sqlcv1.TenantInviteLink, error)
	SyncDeleteTenantInvite(ctx context.Context, db sqlcv1.DBTX, id uuid.UUID) error

	SyncUpsertTenantMember(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantMemberParams) (*sqlcv1.TenantMember, error)
	SyncUpdateTenantMember(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantMemberParams) (*sqlcv1.TenantMember, error)
	SyncDeleteTenantMember(ctx context.Context, db sqlcv1.DBTX, id uuid.UUID) error

	SyncUpsertTenantAlertingSettings(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantAlertingSettingsParams) (*sqlcv1.TenantAlertingSettings, error)
}

type syncRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcv1.Queries
	l       *zerolog.Logger
}

func NewSyncRepository(pool *pgxpool.Pool, l *zerolog.Logger) SyncRepository {
	return &syncRepository{
		pool:    pool,
		queries: sqlcv1.New(),
		l:       l,
	}
}

func (r *syncRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *syncRepository) SyncUpsertTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantParams) (*sqlcv1.Tenant, error) {
	return r.queries.SyncUpsertTenant(ctx, db, arg)
}

func (r *syncRepository) SyncUpdateTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantParams) (*sqlcv1.Tenant, error) {
	return r.queries.SyncUpdateTenant(ctx, db, arg)
}

func (r *syncRepository) SyncSoftDeleteTenant(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncSoftDeleteTenantParams) error {
	return r.queries.SyncSoftDeleteTenant(ctx, db, arg)
}

func (r *syncRepository) SyncUpsertUser(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertUserParams) (*sqlcv1.User, error) {
	return r.queries.SyncUpsertUser(ctx, db, arg)
}

func (r *syncRepository) SyncUpsertTenantInvite(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantInviteParams) (*sqlcv1.TenantInviteLink, error) {
	return r.queries.SyncUpsertTenantInvite(ctx, db, arg)
}

func (r *syncRepository) SyncUpdateTenantInvite(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantInviteParams) (*sqlcv1.TenantInviteLink, error) {
	return r.queries.SyncUpdateTenantInvite(ctx, db, arg)
}

func (r *syncRepository) SyncDeleteTenantInvite(ctx context.Context, db sqlcv1.DBTX, id uuid.UUID) error {
	return r.queries.DeleteTenantInvite(ctx, db, id)
}

func (r *syncRepository) SyncUpsertTenantMember(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantMemberParams) (*sqlcv1.TenantMember, error) {
	return r.queries.SyncUpsertTenantMember(ctx, db, arg)
}

func (r *syncRepository) SyncUpdateTenantMember(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpdateTenantMemberParams) (*sqlcv1.TenantMember, error) {
	return r.queries.SyncUpdateTenantMember(ctx, db, arg)
}

func (r *syncRepository) SyncDeleteTenantMember(ctx context.Context, db sqlcv1.DBTX, id uuid.UUID) error {
	return r.queries.DeleteTenantMember(ctx, db, id)
}

func (r *syncRepository) SyncUpsertTenantAlertingSettings(ctx context.Context, db sqlcv1.DBTX, arg sqlcv1.SyncUpsertTenantAlertingSettingsParams) (*sqlcv1.TenantAlertingSettings, error) {
	return r.queries.SyncUpsertTenantAlertingSettings(ctx, db, arg)
}
