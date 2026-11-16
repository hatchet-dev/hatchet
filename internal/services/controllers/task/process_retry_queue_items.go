package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskRetryQueueItems(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-retry-queue-items")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID := uuid.MustParse(tenantId)

	retryQueueItems, shouldContinue, err := tc.repov1.Tasks().ProcessTaskRetryQueueItems(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	if num := len(retryQueueItems); num > 0 {
		tc.l.Info().Ctx(ctx).Msgf("reassigning %d step runs", num)

		// a trigger requeues the released items, which cannot wake the scheduler, so without this
		// they wait for the queuer's next poll, which backs off to tens of seconds on a quiet queue
		if notifyErr := tc.notifySchedulerOfReleasedRetries(ctx, tenantIdUUID, retryQueueItems); notifyErr != nil {
			tc.l.Error().Ctx(ctx).Err(notifyErr).Msg("could not notify scheduler of released retry queue items")
		}
	}

	for _, task := range retryQueueItems {
		taskId := task.TaskID

		monitoringEvent := tasktypes.CreateMonitoringEventPayload{
			TaskId:         taskId,
			RetryCount:     task.TaskRetryCount,
			EventType:      sqlcv1.V1EventTypeOlapQUEUED,
			EventTimestamp: time.Now(),
		}

		olapMsg, innerErr := tasktypes.MonitoringEventMessageFromInternal(
			tenantIdUUID,
			monitoringEvent,
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not create monitoring event message: %w", err))
			continue
		}

		innerErr = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not publish monitoring event message: %w", err))
		}
	}

	if err != nil {
		return false, fmt.Errorf("could not list retry queue items for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}

func (tc *TasksControllerImpl) notifySchedulerOfReleasedRetries(ctx context.Context, tenantId uuid.UUID, retryQueueItems []*sqlcv1.V1RetryQueueItem) error {
	taskIds := make([]int64, 0, len(retryQueueItems))

	for _, item := range retryQueueItems {
		taskIds = append(taskIds, item.TaskID)
	}

	tasks, err := tc.repov1.Tasks().ListTasks(ctx, tenantId, taskIds)

	if err != nil {
		return fmt.Errorf("could not list released tasks: %w", err)
	}

	queueNames := make([]string, 0, len(tasks))
	concurrencyStrategyIds := make([]int64, 0)

	for _, task := range tasks {
		// a task with a concurrency strategy is released into its first strategy's slots rather than
		// onto its queue, and that strategy notifies the queue once it lets the task through
		if len(task.ConcurrencyStrategyIds) > 0 {
			concurrencyStrategyIds = append(concurrencyStrategyIds, task.ConcurrencyStrategyIds[0])
			continue
		}

		queueNames = append(queueNames, task.Queue)
	}

	queueNames = enumerables.Uniq(queueNames)
	concurrencyStrategyIds = enumerables.Uniq(concurrencyStrategyIds)

	if len(queueNames) == 0 && len(concurrencyStrategyIds) == 0 {
		return nil
	}

	return tc.notifySchedulerOfRequeuedItems(ctx, tenantId, queueNames, concurrencyStrategyIds)
}
