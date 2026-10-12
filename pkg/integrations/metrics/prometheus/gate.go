package prometheus

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// EntitlementChecker reports whether Prometheus metrics are entitled for a
// tenant. It's a func rather than repository.TenantEntitlementRepository to
// avoid importing pkg/repository (and the resulting import cycle).
type EntitlementChecker func(ctx context.Context, tenantId uuid.UUID) (bool, error)

// Gate decides whether per-tenant Prometheus metrics should be collected. When
// tenantScoped is false (self-hosted/OSS default) every tenant is enabled and
// the gate is a no-op. When tenantScoped is true, collection is gated on each
// tenant's prometheus_metrics entitlement.
type Gate struct {
	checker      EntitlementChecker
	l            *zerolog.Logger
	tenantScoped bool
}

// NewGate builds a Gate. When tenantScoped is false, Enabled always returns
// true and the checker is never consulted.
func NewGate(checker EntitlementChecker, tenantScoped bool, l *zerolog.Logger) *Gate {
	return &Gate{
		checker:      checker,
		tenantScoped: tenantScoped,
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

	enabled, err := g.checker(ctx, tenantId)

	if err != nil {
		if g.l != nil {
			g.l.Error().Err(err).Str("tenant_id", tenantId.String()).Msg("failed to check prometheus metrics entitlement, skipping collection")
		}

		return false
	}

	return enabled
}
