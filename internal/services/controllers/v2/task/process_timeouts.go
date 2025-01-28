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

func (tc *TasksControllerImpl) runTenantTimeoutTasks(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running timeout for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx)

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

	tasks, shouldContinue, err := tc.repov2.ProcessTaskTimeouts(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	if num := len(tasks); num > 0 {
		tc.l.Info().Msgf("timed out %d step runs", num)
	}

	for _, task := range tasks {
		taskId := task.ID

		olapMsg, innerErr := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         &taskId,
				RetryCount:     task.RetryCount,
				EventType:      olap.EVENT_TYPE_TIMED_OUT,
				EventTimestamp: time.Now(),
			},
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
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
