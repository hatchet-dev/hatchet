package task

import (
	"context"
	"fmt"
	"time"

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

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskReassignments(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	for _, task := range res.ReleasedTasks {
		var workerId *string

		if task.WorkerID.Valid {
			workerIdStr := sqlchelpers.UUIDToStr(task.WorkerID)
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
	}

	err = tc.processFailTasksResponse(ctx, tenantId, res)

	if err != nil {
		return false, fmt.Errorf("could not process fail tasks response: %w", err)
	}

	return shouldContinue, nil
}
