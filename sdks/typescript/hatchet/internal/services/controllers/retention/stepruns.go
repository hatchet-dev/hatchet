package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (wc *RetentionControllerImpl) runDeleteExpiredStepRuns(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		wc.l.Debug().Msgf("retention controller: deleting expired step runs")

		err := wc.ForTenants(ctx, wc.runDeleteExpireStepRunsTenant)

		if err != nil {
			wc.l.Err(err).Msg("could not run delete expired step runs")
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteExpireStepRunsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-expired-step-runs")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// keep deleting until the context is done
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		hasMore, err := wc.repo.StepRun().ClearStepRunPayloadData(ctx, tenantId)

		if err != nil {
			return fmt.Errorf("could not delete expired step runs: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}
