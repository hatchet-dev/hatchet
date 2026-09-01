package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processPausedWorkflowQueueItemTTL(ctx context.Context, tenantIdStr string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-paused-workflow-queue-item-ttl")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantIdStr})
	tenantId, err := uuid.Parse(tenantIdStr)

	if err != nil {
		return false, fmt.Errorf("could not parse tenant id %s: %w", tenantIdStr, err)
	}

	// this is a reconciliation step (running here just because this is a cron), which
	// requeues any queue items that were stranded in the paused table when we unpaused,
	// since there's a race condition on unpause between moving queue items out of this table
	// and continuing to write into it before the pause flag propagates
	if err := tc.requeueItemsForUnpausedWorkflows(ctx, tenantId); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not requeue paused queue items for unpaused workflows")
	}

	res, shouldContinue, err := tc.repov1.Tasks().ExpirePausedWorkflowQueueItems(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not expire paused workflow queue items for tenant %s: %w", tenantId, err)
	}

	if len(res.ReleasedTasks) == 0 {
		return shouldContinue, nil
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	if err := tc.signaler.SendInternalEvents(ctx, tenantId, res.InternalEvents); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not send internal events for cleaned up tasks")
	}

	for _, releasedTask := range res.ReleasedTasks {
		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         releasedTask.ID,
				RetryCount:     releasedTask.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapCANCELLED,
				EventTimestamp: time.Now(),
				EventMessage:   "paused workflow queue item TTL expired.",
			},
		)

		if err != nil {
			tc.l.Error().Ctx(ctx).Err(err).Msg("could not create monitoring event message for cleaned up task")
			continue
		}

		if err := tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false); err != nil {
			tc.l.Error().Ctx(ctx).Err(err).Msg("could not publish monitoring event message for cleaned up task")
		}
	}

	return shouldContinue, nil
}

func (tc *TasksControllerImpl) requeueItemsForUnpausedWorkflows(ctx context.Context, tenantId uuid.UUID) error {
	workflowIds, err := tc.repov1.Workflows().ListUnpausedWorkflowsWithPausedQueueItems(ctx, tenantId)

	if err != nil {
		return fmt.Errorf("could not list unpaused workflows with paused queue items: %w", err)
	}

	if len(workflowIds) == 0 {
		return nil
	}

	queueNames, concurrencyStrategyIds, err := tc.repov1.Workflows().RequeuePausedWorkflowQueueItems(ctx, tenantId, workflowIds)

	if err != nil {
		return fmt.Errorf("could not requeue paused queue items for unpaused workflows: %w", err)
	}

	if len(queueNames) > 0 || len(concurrencyStrategyIds) > 0 {
		return tc.notifySchedulerOfRequeuedItems(ctx, tenantId, queueNames, concurrencyStrategyIds)
	}

	return nil
}
