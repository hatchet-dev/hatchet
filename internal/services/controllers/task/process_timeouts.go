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

func (tc *TasksControllerImpl) processTaskTimeouts(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-timeout")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskTimeouts(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	err = tc.processFailTasksResponse(ctx, tenantId, res.FailTasksResponse)

	if err != nil {
		return false, fmt.Errorf("could not process fail tasks response: %w", err)
	}

	cancellationSignals := make([]tasktypes.SignalTaskCancelledPayload, 0, len(res.TimeoutTasks))

	for _, task := range res.TimeoutTasks {
		cancellationSignals = append(cancellationSignals, tasktypes.SignalTaskCancelledPayload{
			TaskId:     task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
			WorkerId:   task.WorkerID.String(),
		})

		// send failed tasks to the olap repository
		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapTIMEDOUT,
				EventTimestamp: time.Now(),
				EventMessage:   fmt.Sprintf("Task exceeded timeout of %s", task.StepTimeout.String),
			},
		)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create monitoring event message")
			continue
		}

		err = tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create monitoring event message")
			continue
		}
	}

	if len(cancellationSignals) > 0 {
		err = tc.sendTaskCancellationsToDispatcher(ctx, tenantId, cancellationSignals)

		if err != nil {
			return false, fmt.Errorf("could not send task cancellations to dispatcher: %w",
				err)

		}
	}

	return shouldContinue, nil
}
