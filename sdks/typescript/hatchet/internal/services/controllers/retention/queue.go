package retention

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (rc *RetentionControllerImpl) runDeleteQueueItems(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting queue items")

		err := rc.ForTenants(ctx, rc.runDeleteQueueItemsTenant)

		if err != nil {
			rc.l.Err(err).Msg("could not run delete queue items")
		}
	}
}

func (rc *RetentionControllerImpl) runDeleteQueueItemsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-queue-items-tenant")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	return rc.repo.StepRun().CleanupQueueItems(ctx, tenantId)
}

func (rc *RetentionControllerImpl) runDeleteInternalQueueItems(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting internal queue items")

		err := rc.ForTenants(ctx, rc.runDeleteInternalQueueItemsTenant)

		if err != nil {
			rc.l.Err(err).Msg("could not run delete internal queue items")
		}
	}
}

func (rc *RetentionControllerImpl) runDeleteInternalQueueItemsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-internal-queue-items-tenant")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	return rc.repo.StepRun().CleanupInternalQueueItems(ctx, tenantId)
}

func (rc *RetentionControllerImpl) runDeleteRetryQueueItems(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting retry queue items")

		err := rc.ForTenants(ctx, rc.runDeleteRetryQueueItemsTenant)

		if err != nil {
			rc.l.Err(err).Msg("could not run delete retry queue items")
		}
	}
}

func (rc *RetentionControllerImpl) runDeleteRetryQueueItemsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-retry-queue-items-tenant")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	return rc.repo.StepRun().CleanupRetryQueueItems(ctx, tenantId)
}
