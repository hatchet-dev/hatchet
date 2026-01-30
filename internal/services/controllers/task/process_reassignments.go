package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskReassignments(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-reassignments")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskReassignments(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	retriedTasks := make(map[int64]bool)

	for _, task := range res.RetriedTasks {
		retriedTasks[task.Id] = true
	}

	prometheus.ReassignedTasks.Add(float64(len(res.RetriedTasks)))
	prometheus.TenantReassignedTasks.WithLabelValues(tenantId).Add(float64(len(res.RetriedTasks)))

	for _, task := range res.ReleasedTasks {
		var workerId *string

		if task.WorkerID != uuid.Nil {
			workerIdStr := task.WorkerID.String()
			workerId = &workerIdStr
		}

		// send failed tasks to the olap repository
		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapREASSIGNED,
				EventTimestamp: time.Now(),
				EventMessage:   "Worker did not send a heartbeat for 30 seconds",
				WorkerId:       workerId,
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

		if _, ok := retriedTasks[task.ID]; !ok {
			// if the task was not retried, we should fail it
			// send failed tasks to the olap repository
			olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         task.ID,
					RetryCount:     task.RetryCount,
					EventType:      sqlcv1.V1EventTypeOlapFAILED,
					EventTimestamp: time.Now(),
					EventMessage:   "Task reached its maximum reassignment count",
					EventPayload:   "Task reached its maximum reassignment count",
					WorkerId:       workerId,
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
	}

	err = tc.processFailTasksResponse(ctx, tenantId, res)

	if err != nil {
		return false, fmt.Errorf("could not process fail tasks response: %w", err)
	}

	return shouldContinue, nil
}
