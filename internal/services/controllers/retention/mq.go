package retention

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (rc *RetentionControllerImpl) runDeleteMessageQueueItems(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Msgf("retention controller: deleting queue items")

		err := rc.ForTenants(ctx, rc.runDeleteMessageQueueItemsTenant)

		if err != nil {
			rc.l.Err(err).Msg("could not run delete queue items")
		}
	}
}

func (rc *RetentionControllerImpl) runDeleteMessageQueueItemsTenant(ctx context.Context, tenant dbsqlc.Tenant) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-queue-items-tenant")
	defer span.End()

	if tenant.Name != "internal" {
		return nil
	}

	err := rc.repo.MessageQueue().CleanupQueues(ctx)

	if err != nil {
		return err
	}

	return rc.repo.MessageQueue().CleanupMessageQueueItems(ctx)
}
