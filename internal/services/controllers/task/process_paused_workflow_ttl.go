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

func (tc *TasksControllerImpl) processPausedWorkflowQueueItemTTL(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-paused-workflow-queue-item-ttl")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID := uuid.MustParse(tenantId)

	res, shouldContinue, err := tc.repov1.Tasks().ExpirePausedWorkflowQueueItems(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not expire paused workflow queue items for tenant %s: %w", tenantId, err)
	}

	if len(res.ReleasedTasks) == 0 {
		return shouldContinue, nil
	}

	tc.notifyQueuesOnCompletion(ctx, tenantIdUUID, res.ReleasedTasks)

	if err := tc.signaler.SendInternalEvents(ctx, tenantIdUUID, res.InternalEvents); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not send internal events for cleaned up tasks")
	}

	for _, releasedTask := range res.ReleasedTasks {
		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantIdUUID,
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
