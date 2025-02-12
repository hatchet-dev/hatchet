package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
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

	tasks, shouldContinue, err := tc.repov2.Tasks().ProcessTaskTimeouts(ctx, tenantId)

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
				TaskId:         taskId,
				RetryCount:     task.RetryCount,
				EventType:      timescalev2.V2EventTypeOlapTIMEDOUT,
				EventTimestamp: time.Now(),
			},
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
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
