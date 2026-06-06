package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (rc *RetentionControllerImpl) runCleanupOldData(parentCtx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(parentCtx).Msg("retention controller: cleaning up old data")

		ctx, cancel := context.WithTimeout(parentCtx, 30*time.Minute)
		defer cancel()

		if err := rc.ForTenants(ctx, 5*time.Minute, rc.cleanupOldDataForTenant); err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("could not cleanup old data")
		}
	}
}

func (rc *RetentionControllerImpl) cleanupOldDataForTenant(ctx context.Context, tenantId uuid.UUID) error {
	ctx, span := telemetry.NewSpan(ctx, "cleanup-old-data-tenant")
	defer span.End()

	tenant, err := rc.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		return fmt.Errorf("could not get tenant %s: %w", tenantId.String(), err)
	}

	cutoff, err := GetDataRetentionExpiredTime(tenant.DataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("could not get cutoff for tenant %s: %w", tenant.ID.String(), err)
	}

	shouldContinue := true

	for shouldContinue {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		shouldContinue, err = rc.repo.OLAP().CleanupOldTaskEvents(ctx, tenant.ID, cutoff)
		if err != nil {
			return fmt.Errorf("could not cleanup old task events for tenant %s: %w", tenant.ID.String(), err)
		}
	}

	return nil
}
