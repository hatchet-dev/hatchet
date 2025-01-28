package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (tc *TasksControllerImpl) runTenantReassignTasks(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running reassign for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.reassignTaskOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.reassignTaskOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processTaskReassignments(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-reassignments")
	defer span.End()

	tasks, shouldContinue, err := tc.repov2.ProcessTaskReassignments(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	if num := len(tasks); num > 0 {
		tc.l.Info().Msgf("reassigning %d step runs", num)
	}

	for _, task := range tasks {
		workerId := sqlchelpers.UUIDToStr(task.WorkerID)
		taskId := task.ID

		monitoringEvent := tasktypes.CreateMonitoringEventPayload{
			TaskId:         &taskId,
			RetryCount:     task.RetryCount,
			WorkerId:       &workerId,
			EventTimestamp: time.Now(),
		}

		switch task.Operation {
		case "REASSIGNED":
			monitoringEvent.EventType = olap.EVENT_TYPE_REASSIGNED
		case "FAILED":
			monitoringEvent.EventType = olap.EVENT_TYPE_FAILED
			monitoringEvent.EventPayload = "Worker became inactive, and we reached the maximum number of internal retries"
		default:
			tc.l.Error().Msgf("unknown operation %s", task.Operation)
			continue
		}

		olapMsg, innerErr := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			monitoringEvent,
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not create monitoring event message: %w", err))
			continue
		}

		tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
		)
	}

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
