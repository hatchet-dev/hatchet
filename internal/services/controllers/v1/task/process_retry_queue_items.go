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

func (tc *TasksControllerImpl) runTenantRetryQueueItems(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running retry queue items for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.retryTaskOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.retryTaskOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processTaskRetryQueueItems(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-retry-queue-items")
	defer span.End()

	retryQueueItems, shouldContinue, err := tc.repov1.Tasks().ProcessTaskRetryQueueItems(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	if num := len(retryQueueItems); num > 0 {
		tc.l.Info().Msgf("reassigning %d step runs", num)
	}

	for _, task := range retryQueueItems {
		taskId := task.TaskID

		monitoringEvent := tasktypes.CreateMonitoringEventPayload{
			TaskId:         taskId,
			RetryCount:     task.TaskRetryCount,
			EventType:      sqlcv1.V1EventTypeOlapQUEUED,
			EventTimestamp: time.Now(),
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
		return false, fmt.Errorf("could not list retry queue items for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
