package prometheus

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
)

// entitlementCacheTTL bounds how stale a tenant's prometheus_metrics
// entitlement can be on the hot emission paths. Entitlements change rarely
// (only on plan changes fanned out from the control plane), so a short TTL
// keeps collection off the database without meaningfully delaying gating
// changes.
const entitlementCacheTTL = 1 * time.Minute

// EntitlementChecker reports whether Prometheus metrics are entitled for a
// tenant. It is satisfied structurally by
// repository.TenantEntitlementRepository; defining it locally avoids importing
// pkg/repository (and the resulting import cycle).
type EntitlementChecker interface {
	IsPrometheusMetricsEnabled(ctx context.Context, tenantId uuid.UUID) (bool, error)
}

// Gate decides whether per-tenant Prometheus metrics should be collected. When
// tenantScoped is false (self-hosted/OSS default) every tenant is enabled and
// the gate is a no-op. When tenantScoped is true, collection is gated on each
// tenant's prometheus_metrics entitlement, read through a short-lived cache to
// keep the hot emission paths off the database.
type Gate struct {
	checker      EntitlementChecker
	cache        cache.Cacheable
	l            *zerolog.Logger
	tenantScoped bool
}

// NewGate builds a Gate. When tenantScoped is false, Enabled always returns
// true and the checker is never consulted.
func NewGate(checker EntitlementChecker, tenantScoped bool, l *zerolog.Logger) *Gate {
	return &Gate{
		checker:      checker,
		tenantScoped: tenantScoped,
		cache:        cache.New(entitlementCacheTTL),
		l:            l,
	}
}

// Enabled reports whether per-tenant Prometheus metrics should be collected for
// the tenant. It returns true when the gate is nil or not tenant-scoped. On a
// lookup error it fails closed (returns false) so unentitled data is not
// collected.
func (g *Gate) Enabled(ctx context.Context, tenantId uuid.UUID) bool {
	if g == nil || !g.tenantScoped {
		return true
	}

	enabled, err := cache.MakeCacheable(g.cache, tenantId.String(), func() (*bool, error) {
		v, err := g.checker.IsPrometheusMetricsEnabled(ctx, tenantId)
		if err != nil {
			return nil, err
		}

		return &v, nil
	})

	if err != nil {
		if g.l != nil {
			g.l.Error().Err(err).Str("tenant_id", tenantId.String()).Msg("failed to check prometheus metrics entitlement, skipping collection")
		}

		return false
	}

	return *enabled
}
