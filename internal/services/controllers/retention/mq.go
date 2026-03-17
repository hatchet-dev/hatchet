package retention

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (rc *RetentionControllerImpl) runDeleteMessageQueueItems(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		rc.l.Debug().Ctx(ctx).Msgf("retention controller: deleting queue items")

		// get internal tenant
		tenant, err := rc.p.GetInternalTenantForController(ctx)

		if err != nil {
			rc.l.Error().Ctx(ctx).Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			return
		}

		err = rc.runDeleteMessageQueueItemsQueries(ctx)

		if err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("could not run delete mq queue items")
		}
	}
}

func (rc *RetentionControllerImpl) runDeleteMessageQueueItemsQueries(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "delete-queue-items-tenant")
	defer span.End()

	err := rc.repo.MessageQueue().CleanupQueues(ctx)

	if err != nil {
		return err
	}

	return rc.repo.MessageQueue().CleanupMessageQueueItems(ctx)
}
