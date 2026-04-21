package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (rc *RetentionControllerImpl) runCleanupOldWorkers(ctx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(ctx).Msg("retention controller: cleaning up old workers")

		ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()

		if err := rc.ForTenants(ctx, rc.cleanupOldWorkersForTenant); err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("could not cleanup old workers")
		}
	}
}

func (rc *RetentionControllerImpl) cleanupOldWorkersForTenant(ctx context.Context, tenant sqlcv1.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "cleanup-old-workers-tenant")
	defer span.End()

	cutoff, err := GetDataRetentionExpiredTime(tenant.DataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("could not get cutoff for tenant %s: %w", tenant.ID.String(), err)
	}

	shouldContinue := true

	for shouldContinue {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		shouldContinue, err = rc.repo.Workers().CleanupOldWorkers(ctx, tenant.ID, cutoff)
		if err != nil {
			return fmt.Errorf("could not cleanup old workers for tenant %s: %w", tenant.ID.String(), err)
		}
	}

	return nil
}
