package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (rc *RetentionControllerImpl) runDeleteOldWorkers(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting old workers")

		err := rc.ForTenants(ctx, rc.runDeleteOldWorkerDataTenant)

		if err != nil {
			rc.l.Err(err).Msg("could not run delete old workers")
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteOldWorkerDataTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	// simultenously delete old workers and worker assign events
	errCh := make(chan error, 2)

	go func() {
		errCh <- wc.runDeleteOldWorkersTenant(ctx, tenant)
	}()

	go func() {
		errCh <- wc.runDeleteOldWorkerAssignEventsTenant(ctx, tenant)
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh

		if err != nil {
			return err
		}
	}

	return nil
}

func (wc *RetentionControllerImpl) runDeleteOldWorkersTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-old-workers-tenant")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// hard-coded to last heartbeat before 24 hours
	lastHeartbeatBefore := time.Now().UTC().Add(-24 * time.Hour)

	// keep deleting until the context is done
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// delete expired workflow runs
		hasMore, err := wc.repo.Worker().DeleteOldWorkers(ctx, tenantId, lastHeartbeatBefore)

		if err != nil {
			return fmt.Errorf("could not delete expired events: %w", err)
		}

		if !hasMore {
			return nil
		}
	}
}

func (wc *RetentionControllerImpl) runDeleteOldWorkerAssignEventsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-old-workers-tenant")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// hard-coded to last heartbeat after 24 hours
	lastHeartbeatAfter := time.Now().UTC().Add(-24 * time.Hour)

	err := wc.repo.Worker().DeleteOldWorkerEvents(ctx, tenantId, lastHeartbeatAfter)

	if err != nil {
		return fmt.Errorf("could not delete expired events: %w", err)
	}

	return nil
}
