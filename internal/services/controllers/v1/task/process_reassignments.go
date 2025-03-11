package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (tc *TasksControllerImpl) runTenantReassignTasks(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running reassign for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

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

	tasks, shouldContinue, err := tc.repov1.Tasks().ProcessTaskReassignments(ctx, tenantId)

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
			TaskId:         taskId,
			RetryCount:     task.RetryCount,
			WorkerId:       &workerId,
			EventTimestamp: time.Now(),
		}

		switch task.Operation {
		case "REASSIGNED":
			monitoringEvent.EventType = sqlcv1.V1EventTypeOlapREASSIGNED
		case "FAILED":
			monitoringEvent.EventType = sqlcv1.V1EventTypeOlapFAILED
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
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
