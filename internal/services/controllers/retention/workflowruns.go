package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (wc *RetentionControllerImpl) runDeleteExpiredWorkflowRuns(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		wc.l.Debug().Msgf("retention controller: deleting expired workflow runs")

		err := wc.ForTenants(ctx, wc.runDeleteExpiredWorkflowRunsTenant)

		if err != nil {
			wc.l.Err(err).Msg("could not run delete expired workflow runs")
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteExpiredWorkflowRunsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-expired-workflow-runs")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createdBefore, err := GetDataRetentionExpiredTime(tenant.DataRetentionPeriod)

	if err != nil {
		return fmt.Errorf("could not get data retention expired time: %w", err)
	}

	// keep deleting until the context is done
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// delete expired workflow runs
		hasMore, err := wc.repo.WorkflowRun().SoftDeleteExpiredWorkflowRuns(ctx, tenantId, []dbsqlc.WorkflowRunStatus{
			dbsqlc.WorkflowRunStatusSUCCEEDED,
			dbsqlc.WorkflowRunStatusFAILED,
		}, createdBefore)

		if err != nil {
			return fmt.Errorf("could not delete expired workflow runs: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}
