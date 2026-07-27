package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskReassignments(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-reassignments")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID := uuid.MustParse(tenantId)

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskReassignments(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	retriedTasks := make(map[int64]bool)

	for _, task := range res.RetriedTasks {
		retriedTasks[task.Id] = true
	}

	prometheus.ReassignedTasks.Add(float64(len(res.RetriedTasks)))
	if tc.promGate.Enabled(ctx, tenantIdUUID) {
		prometheus.TenantReassignedTasks.WithLabelValues(tenantId).Add(float64(len(res.RetriedTasks)))
	}

	reassignedPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(res.ReleasedTasks))

	for _, task := range res.ReleasedTasks {
		var workerId *uuid.UUID

		if task.WorkerID != uuid.Nil {
			workerId = &task.WorkerID
		}

		reassignedPayloads = append(reassignedPayloads, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapREASSIGNED,
			EventTimestamp: time.Now(),
			EventMessage:   "Worker did not send a heartbeat for 30 seconds",
			WorkerId:       workerId,
		})

		if _, ok := retriedTasks[task.ID]; !ok {
			// if the task was not retried, we should fail it
			reassignedPayloads = append(reassignedPayloads, tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapFAILED,
				EventTimestamp: time.Now(),
				EventMessage:   "Task reached its maximum reassignment count",
				EventPayload:   "Task reached its maximum reassignment count",
				WorkerId:       workerId,
			})
		}
	}

	// send reassigned/failed tasks to the olap repository
	if pubErr := tc.repov1.OLAPOutbox().MonitoringEvents(ctx, tenantIdUUID, reassignedPayloads...); pubErr != nil {
		tc.l.Error().Ctx(ctx).Err(pubErr).Msg("could not publish monitoring event message")
	}

	err = tc.processFailTasksResponse(ctx, tenantIdUUID, res)

	if err != nil {
		return false, fmt.Errorf("could not process fail tasks response: %w", err)
	}

	return shouldContinue, nil
}
