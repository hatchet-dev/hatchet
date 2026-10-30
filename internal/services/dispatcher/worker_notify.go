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
func (d *DispatcherImpl) NotifyNewWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID) {
	d.publishToSchedulerPartition(ctx, tenant, "new worker", func() (*msgqueue.Message, error) {
		return tasktypes.NotifyNewWorker(tenant.ID, workerId)
	})
}

// NotifyWorkerDispatcherChanged only logs a failed publish: the scheduler's dispatcher id cache TTL
// covers it.
func (d *DispatcherImpl) NotifyWorkerDispatcherChanged(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID) {
	d.publishToSchedulerPartition(ctx, tenant, "worker dispatcher change", func() (*msgqueue.Message, error) {
		return tasktypes.NotifyWorkerDispatcherChanged(tenant.ID, workerId, d.dispatcherId)
	})
}

// publishToSchedulerPartition runs detached from ctx so the publish outlives the calling handler, and
// is a no-op for tenants without a scheduler partition.
func (d *DispatcherImpl) publishToSchedulerPartition(ctx context.Context, tenant *sqlcv1.Tenant, what string, newMessage func() (*msgqueue.Message, error)) {
	if !tenant.SchedulerPartitionId.Valid {
		return
	}

	go func() {
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		msg, err := newMessage()

		if err != nil {
			d.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msgf("could not create message for notifying %s", what)
			return
		}

		err = d.pubsub.Pub(
			notifyCtx,
			msgqueue.SchedulerPartitionTopic(tenant.SchedulerPartitionId.String),
			msg,
		)

		if err != nil {
			d.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msgf("could not publish %s to scheduler partition topic", what)
		}
	}()
}
