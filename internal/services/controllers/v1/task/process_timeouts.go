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

func (tc *TasksControllerImpl) runTenantTimeoutTasks(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running timeout for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.timeoutTaskOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.timeoutTaskOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processTaskTimeouts(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-timeout")
	defer span.End()

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
			WorkerId:   sqlchelpers.UUIDToStr(task.WorkerID),
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
