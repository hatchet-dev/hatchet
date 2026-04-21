package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const cleanupOldWorkersSleepBetweenBatches = time.Second

func (rc *RetentionControllerImpl) runCleanupOldWorkers(ctx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(ctx).Msg("retention controller: cleaning up old workers")

		err := rc.ForTenants(ctx, rc.runCleanupOldWorkersForTenant)
		if err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("could not cleanup old workers")
		}
	}
}

func (rc *RetentionControllerImpl) runCleanupOldWorkersForTenant(ctx context.Context, tenant sqlcv1.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "cleanup-old-workers-tenant")
	defer span.End()

	cutoff, err := GetDataRetentionExpiredTime(tenant.DataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("could not get cutoff for tenant %s: %w", tenant.ID.String(), err)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		hasMore, err := rc.repo.Workers().DeleteOldWorkers(ctx, tenant.ID, cutoff)
		if err != nil {
			return fmt.Errorf("could not delete old workers for tenant %s: %w", tenant.ID.String(), err)
		}

		if !hasMore {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cleanupOldWorkersSleepBetweenBatches):
		}
	}
}
