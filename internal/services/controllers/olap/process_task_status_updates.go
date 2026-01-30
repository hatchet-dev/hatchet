package olap

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (o *OLAPControllerImpl) runTaskStatusUpdates(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		shouldContinue := true

		for shouldContinue {
			o.l.Debug().Msgf("partition: running status updates for tasks")

			// list all tenants
			tenants, err := o.p.ListTenantsForController(ctx, sqlcv1.TenantMajorEngineVersionV1)

			if err != nil {
				o.l.Error().Err(err).Msg("could not list tenants")
				return
			}

			tenantIds := make([]string, 0, len(tenants))

			for _, tenant := range tenants {
				tenantId := tenant.ID
				tenantIds = append(tenantIds, tenantId)
			}

			var rows []v1.UpdateTaskStatusRow

			shouldContinue, rows, err = o.repo.OLAP().UpdateTaskStatuses(ctx, tenantIds)

			if err != nil {
				o.l.Error().Err(err).Msg("could not update task statuses")
				return
			}

			err = o.notifyTasksUpdated(ctx, rows)

			if err != nil {
				o.l.Error().Err(err).Msg("failed to notify updated task statuses")
				return
			}
		}
	}
}

func (o *OLAPControllerImpl) notifyTasksUpdated(ctx context.Context, rows []v1.UpdateTaskStatusRow) error {
	tenantIdToPayloads := make(map[uuid.UUID][]tasktypes.NotifyFinalizedPayload)

	for _, row := range rows {
		if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
			continue
		}

		tenantIdToPayloads[row.TenantId] = append(tenantIdToPayloads[row.TenantId], tasktypes.NotifyFinalizedPayload{
			ExternalId: row.ExternalId.String(),
			Status:     row.ReadableStatus,
		})
	}

	// Send prometheus updates asynchronously
	if o.prometheusMetricsEnabled && o.taskPrometheusUpdateCh != nil {
		for _, row := range rows {
			if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
				continue
			}

			update := taskPrometheusUpdate{
				tenantId:       row.TenantId.String(),
				taskId:         row.TaskId,
				taskInsertedAt: row.TaskInsertedAt,
				readableStatus: row.ReadableStatus,
				workflowId:     row.WorkflowId,
				isDAGTask:      row.IsDAGTask,
			}

			select {
			case o.taskPrometheusUpdateCh <- update:
				// Successfully sent
			default:
				// Channel full, discard with warning
				o.l.Warn().Msgf("task prometheus update channel full, discarding update for task %d", row.TaskId)
			}
		}
	}

	// send to the tenant queue
	if len(tenantIdToPayloads) > 0 {
		for tenantId, payloads := range tenantIdToPayloads {
			msg, err := msgqueue.NewTenantMessage(
				tenantId.String(),
				msgqueue.MsgIDWorkflowRunFinished,
				true,
				false,
				payloads...,
			)

			if err != nil {
				return err
			}

			q := msgqueue.TenantEventConsumerQueue(tenantId.String())

			err = o.mq.SendMessage(ctx, q, msg)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
