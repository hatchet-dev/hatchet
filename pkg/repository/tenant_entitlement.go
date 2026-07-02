package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TenantEntitlementRepository is a minimal repository for per-tenant feature
// entitlements. Entitlements are owned upstream (e.g. Hatchet Cloud's control
// plane) and fanned out into the engine database so that engine and API
// components can gate features locally without reaching across databases.
type TenantEntitlementRepository interface {
	// IsAuditLogsEnabled reports whether audit logging is entitled for the tenant.
	// Tenants without an entitlement row are treated as not entitled.
	IsAuditLogsEnabled(ctx context.Context, tenantId uuid.UUID) (bool, error)

	// AnyTenantHasAuditLogs reports whether any of the given tenants is entitled
	// to audit logging.
	AnyTenantHasAuditLogs(ctx context.Context, tenantIds []uuid.UUID) (bool, error)

	// IsPrometheusMetricsEnabled reports whether Prometheus metrics are entitled
	// for the tenant. Tenants without an entitlement row are treated as not entitled.
	IsPrometheusMetricsEnabled(ctx context.Context, tenantId uuid.UUID) (bool, error)

	// SetEntitlements upserts the full set of feature entitlements for the tenant.
	SetEntitlements(ctx context.Context, tenantId uuid.UUID, entitlements TenantEntitlements) error
}

// TenantEntitlements is the full set of per-tenant feature entitlements that are
// fanned out from upstream into the engine database in a single upsert.
type TenantEntitlements struct {
	AuditLogs         bool
	PrometheusMetrics bool
}

type tenantEntitlementRepository struct {
	*sharedRepository
}

func newTenantEntitlementRepository(shared *sharedRepository) TenantEntitlementRepository {
	return &tenantEntitlementRepository{
		sharedRepository: shared,
	}
}

func (t *tenantEntitlementRepository) IsAuditLogsEnabled(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	entitlement, err := t.queries.GetTenantEntitlement(ctx, t.pool, tenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return entitlement.AuditLogs, nil
}

func (t *tenantEntitlementRepository) AnyTenantHasAuditLogs(ctx context.Context, tenantIds []uuid.UUID) (bool, error) {
	if len(tenantIds) == 0 {
		return false, nil
	}

	return t.queries.AnyTenantHasAuditLogs(ctx, t.pool, tenantIds)
}

func (t *tenantEntitlementRepository) IsPrometheusMetricsEnabled(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	entitlement, err := t.queries.GetTenantEntitlement(ctx, t.pool, tenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return entitlement.PrometheusMetrics, nil
}

func (t *tenantEntitlementRepository) SetEntitlements(ctx context.Context, tenantId uuid.UUID, entitlements TenantEntitlements) error {
	_, err := t.queries.UpsertTenantEntitlement(ctx, t.pool, sqlcv1.UpsertTenantEntitlementParams{
		Tenantid:          tenantId,
		Auditlogs:         entitlements.AuditLogs,
		Prometheusmetrics: entitlements.PrometheusMetrics,
	})

	return err
}
