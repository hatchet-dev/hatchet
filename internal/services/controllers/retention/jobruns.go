package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (wc *RetentionControllerImpl) runDeleteExpiredJobRuns(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		wc.l.Debug().Msgf("retention controller: deleting expired job runs")

		err := wc.ForTenants(ctx, wc.runDeleteExpireJobRunsTenant)

		if err != nil {
			wc.l.Err(err).Msg("could not run delete expired job runs")
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteExpireJobRunsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
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

		hasMore, err := wc.repo.JobRun().ClearJobRunPayloadData(ctx, tenantId)

		if err != nil {
			return fmt.Errorf("could not delete expired job runs: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}
