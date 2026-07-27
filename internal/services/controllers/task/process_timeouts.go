package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskTimeouts(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-timeout")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID, err := uuid.Parse(tenantId)

	if err != nil {
		return false, fmt.Errorf("could not parse tenant id %s: %w", tenantId, err)
	}

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskTimeouts(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	err = tc.processFailTasksResponse(ctx, tenantIdUUID, res.FailTasksResponse)

	if err != nil {
		return false, fmt.Errorf("could not process fail tasks response: %w", err)
	}

	cancellationSignals := make([]tasktypes.SignalTaskCancelledPayload, 0, len(res.TimeoutTasks))
	timedOutPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(res.TimeoutTasks))

	for _, task := range res.TimeoutTasks {
		var workerId uuid.UUID
		if task.WorkerID != nil {
			workerId = *task.WorkerID
		}

		cancellationSignals = append(cancellationSignals, tasktypes.SignalTaskCancelledPayload{
			TaskId:     task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
			WorkerId:   workerId,
		})

		timedOutPayloads = append(timedOutPayloads, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapTIMEDOUT,
			EventTimestamp: time.Now(),
			EventMessage:   fmt.Sprintf("Task exceeded timeout of %s", task.StepTimeout.String),
		})
	}

	// send timed-out tasks to the olap repository
	if pubErr := tc.repov1.OLAPOutbox().MonitoringEvents(ctx, tenantIdUUID, timedOutPayloads...); pubErr != nil {
		tc.l.Error().Ctx(ctx).Err(pubErr).Msg("could not publish monitoring event message")
	}

	if len(cancellationSignals) > 0 {
		err = tc.sendTaskCancellationsToDispatcher(ctx, tenantIdUUID, cancellationSignals)

		if err != nil {
			return false, fmt.Errorf("could not send task cancellations to dispatcher: %w",
				err)

		}
	}

	return shouldContinue, nil
}
