package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (rc *RetentionControllerImpl) runDeleteExpiredEvents(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting expired events")

		errChan := make(chan error, 2)

		go func() {
			errChan <- rc.ForTenants(ctx, rc.runDeleteExpiredEventsTenant)
		}()

		go func() {
			errChan <- rc.ForTenants(ctx, rc.runClearDeletedEventsPayloadTenant)
		}()

		err1 := <-errChan
		err2 := <-errChan

		if err1 != nil {
			rc.l.Err(err1).Msg("could not run delete expired events")
		}
		if err2 != nil {
			rc.l.Err(err2).Msg("could not clear deleted event payload")
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteExpiredEventsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-expired-events")
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
		hasMore, err := wc.repo.Event().SoftDeleteExpiredEvents(ctx, tenantId, createdBefore)

		if err != nil {
			return fmt.Errorf("could not delete expired events: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}

func (wc *RetentionControllerImpl) runClearDeletedEventsPayloadTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-expired-events")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// keep deleting until the context is done
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// delete expired workflow runs
		hasMore, err := wc.repo.Event().ClearEventPayloadData(ctx, tenantId)

		if err != nil {
			return fmt.Errorf("could not clear deleted event payload: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}
