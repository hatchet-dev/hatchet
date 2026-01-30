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

func (o *OLAPControllerImpl) runDAGStatusUpdates(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		shouldContinue := true

		for shouldContinue {
			o.l.Debug().Msgf("partition: running status updates for dags")

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

			var rows []v1.UpdateDAGStatusRow

			shouldContinue, rows, err = o.repo.OLAP().UpdateDAGStatuses(ctx, tenantIds)

			if err != nil {
				o.l.Error().Err(err).Msg("could not update DAG statuses")
				return
			}

			err = o.notifyDAGsUpdated(ctx, rows)

			if err != nil {
				o.l.Error().Err(err).Msg("failed to notify updated DAG statuses")
				return
			}
		}

	}
}

func (o *OLAPControllerImpl) notifyDAGsUpdated(ctx context.Context, rows []v1.UpdateDAGStatusRow) error {
	tenantIdToPayloads := make(map[uuid.UUID][]tasktypes.NotifyFinalizedPayload)

	for _, row := range rows {
		tenantIdToPayloads[row.TenantId] = append(tenantIdToPayloads[row.TenantId], tasktypes.NotifyFinalizedPayload{
			ExternalId: row.ExternalId.String(),
			Status:     row.ReadableStatus,
		})

		if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED {
			o.processTenantAlertOperations.RunOrContinue(row.TenantId.String())
		}
	}

	// Send prometheus updates asynchronously
	if o.prometheusMetricsEnabled && o.dagPrometheusUpdateCh != nil {
		for _, row := range rows {
			if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
				continue
			}

			update := dagPrometheusUpdate{
				tenantId:       row.TenantId.String(),
				dagExternalId:  row.ExternalId,
				dagInsertedAt:  row.DagInsertedAt,
				readableStatus: row.ReadableStatus,
				workflowId:     row.WorkflowId,
			}

			select {
			case o.dagPrometheusUpdateCh <- update:
				// Successfully sent
			default:
				// Channel full, discard with warning
				o.l.Warn().Msgf("dag prometheus update channel full, discarding update for dag %s", row.ExternalId.String())
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
