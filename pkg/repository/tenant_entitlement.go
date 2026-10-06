package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TenantEntitlementRepository is a minimal repository for per-tenant feature
// entitlements. Entitlements are owned upstream (e.g. Hatchet Cloud's control
// plane) and fanned out into the engine database so that engine and API
// components can gate features locally without reaching across databases.
type TenantEntitlementRepository interface {
	// HasEntitlement is false for tenants without an entitlement row. Reads are
	// cached for up to entitlementCacheTTL.
	HasEntitlement(ctx context.Context, tenantId uuid.UUID, entitlement Entitlement) (bool, error)

	// AnyTenantHasAuditLogs reports whether any of the given tenants is entitled
	// to audit logging.
	AnyTenantHasAuditLogs(ctx context.Context, tenantIds []uuid.UUID) (bool, error)

	// GetEntitlements is uncached and all false for tenants without an entitlement row.
	GetEntitlements(ctx context.Context, tenantId uuid.UUID) (TenantEntitlements, error)

	// SetEntitlements upserts the full set of feature entitlements for the tenant.
	SetEntitlements(ctx context.Context, tenantId uuid.UUID, entitlements TenantEntitlements) error
}

// Entitlement names a per-tenant feature; values match the tenant_entitlement columns.
type Entitlement string

const (
	EntitlementAuditLogs                       Entitlement = "audit_logs"
	EntitlementPrometheusMetrics               Entitlement = "prometheus_metrics"
	EntitlementStrictAdditionalMetadataFilters Entitlement = "strict_additional_metadata_filters"
	EntitlementDAGOperator                     Entitlement = "dag_operator"
	EntitlementDurableStreams                  Entitlement = "durable_streams"
)

// Entitlements only change on plan changes fanned out from the control plane,
// so this bounds how long a change takes to apply on hot paths.
const entitlementCacheTTL = 5 * time.Minute

const entitlementCacheSize = 10000

// TenantEntitlements is the full set of per-tenant feature entitlements that are
// fanned out from upstream into the engine database in a single upsert.
type TenantEntitlements struct {
	AuditLogs                       bool
	PrometheusMetrics               bool
	StrictAdditionalMetadataFilters bool
	DAGOperator                     bool
	DurableStreams                  bool
}

func (e TenantEntitlements) Has(entitlement Entitlement) (bool, error) {
	switch entitlement {
	case EntitlementAuditLogs:
		return e.AuditLogs, nil
	case EntitlementPrometheusMetrics:
		return e.PrometheusMetrics, nil
	case EntitlementStrictAdditionalMetadataFilters:
		return e.StrictAdditionalMetadataFilters, nil
	case EntitlementDAGOperator:
		return e.DAGOperator, nil
	case EntitlementDurableStreams:
		return e.DurableStreams, nil
	default:
		return false, fmt.Errorf("unknown entitlement %q", entitlement)
	}
}

type tenantEntitlementRepository struct {
	*sharedRepository

	// expiry is checked on read so there's no sweeper goroutine to outlive the repo
	cache *lru.Cache[uuid.UUID, cachedEntitlements]
}

type cachedEntitlements struct {
	entitlements TenantEntitlements
	expiresAt    time.Time
}

func newTenantEntitlementRepository(shared *sharedRepository) TenantEntitlementRepository {
	c, err := lru.New[uuid.UUID, cachedEntitlements](entitlementCacheSize)

	if err != nil {
		panic(err)
	}

	return &tenantEntitlementRepository{
		sharedRepository: shared,
		cache:            c,
	}
}

func (t *tenantEntitlementRepository) HasEntitlement(ctx context.Context, tenantId uuid.UUID, entitlement Entitlement) (bool, error) {
	if cached, ok := t.cache.Get(tenantId); ok && time.Now().Before(cached.expiresAt) {
		return cached.entitlements.Has(entitlement)
	}

	entitlements, err := t.GetEntitlements(ctx, tenantId)

	if err != nil {
		return false, err
	}

	t.cache.Add(tenantId, cachedEntitlements{
		entitlements: entitlements,
		expiresAt:    time.Now().Add(entitlementCacheTTL),
	})

	return entitlements.Has(entitlement)
}

func (t *tenantEntitlementRepository) AnyTenantHasAuditLogs(ctx context.Context, tenantIds []uuid.UUID) (bool, error) {
	if len(tenantIds) == 0 {
		return false, nil
	}

	return t.queries.AnyTenantHasAuditLogs(ctx, t.pool, tenantIds)
}

func (t *tenantEntitlementRepository) GetEntitlements(ctx context.Context, tenantId uuid.UUID) (TenantEntitlements, error) {
	entitlement, err := t.queries.GetTenantEntitlement(ctx, t.pool, tenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TenantEntitlements{}, nil
		}

		return TenantEntitlements{}, err
	}

	return TenantEntitlements{
		AuditLogs:                       entitlement.AuditLogs,
		PrometheusMetrics:               entitlement.PrometheusMetrics,
		StrictAdditionalMetadataFilters: entitlement.StrictAdditionalMetadataFilters,
		DAGOperator:                     entitlement.DagOperator,
		DurableStreams:                  entitlement.DurableStreams,
	}, nil
}

func (t *tenantEntitlementRepository) SetEntitlements(ctx context.Context, tenantId uuid.UUID, entitlements TenantEntitlements) error {
	_, err := t.queries.UpsertTenantEntitlement(ctx, t.pool, sqlcv1.UpsertTenantEntitlementParams{
		Tenantid:                        tenantId,
		Auditlogs:                       entitlements.AuditLogs,
		Prometheusmetrics:               entitlements.PrometheusMetrics,
		Strictadditionalmetadatafilters: entitlements.StrictAdditionalMetadataFilters,
		Dagoperator:                     entitlements.DAGOperator,
		Durablestreams:                  entitlements.DurableStreams,
	})

	if err != nil {
		return err
	}

	t.cache.Remove(tenantId)

	return nil
}
