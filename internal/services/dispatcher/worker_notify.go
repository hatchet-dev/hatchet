package dispatcher

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// NotifyNewWorker tells the tenant's scheduler partition that workerId is available for work.
// The publish runs detached from ctx so it outlives the calling handler, and is a no-op for
// tenants without a scheduler partition.
func (d *DispatcherImpl) NotifyNewWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID) {
	if !tenant.SchedulerPartitionId.Valid {
		return
	}

	go func() {
		// detached from the request so the notify outlives the handler, but keeps
		// the request's values for tracing
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		msg, err := tasktypes.NotifyNewWorker(tenant.ID, workerId)

		if err != nil {
			d.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not create message for notifying new worker")
			return
		}

		err = d.pubsub.Pub(
			notifyCtx,
			msgqueue.SchedulerPartitionTopic(tenant.SchedulerPartitionId.String),
			msg,
		)

		if err != nil {
			d.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not publish message to scheduler partition topic")
		}
	}()
}
