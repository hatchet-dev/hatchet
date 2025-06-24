package olap

import (
	"context"
	"errors"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5"
)

func (o *OLAPControllerImpl) runTenantTaskStatusUpdates(ctx context.Context) func() {
	return func() {
		o.l.Debug().Msgf("partition: running status updates for tasks")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			o.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		o.updateTaskStatusOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			o.updateTaskStatusOperations.RunOrContinue(tenantId)
		}
	}
}

func (o *OLAPControllerImpl) updateTaskStatuses(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-task-statuses")
	defer span.End()

	shouldContinue, rows, err := o.repo.OLAP().UpdateTaskStatuses(ctx, tenantId)

	if err != nil {
		return false, err
	}

	payloads := make([]tasktypes.NotifyFinalizedPayload, 0, len(rows))

	for _, row := range rows {
		payloads = append(payloads, tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		hasDAG, err := o.repo.OLAP().TaskHasDAG(ctx, row.TaskId)
		if err != nil {
			return false, err
		}

		if !hasDAG {
			// instrumentation
			if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCANCELLED {
				workflow, err := o.repo.OLAP().GetWorkflowByExternalId(ctx, tenantId, row.ExternalId)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						continue
					}

					return false, err
				}

				tenantMetric := prometheus.WithTenant(tenantId)
				tenantMetric.WorkflowCompleted.WithLabelValues(tenantId, workflow.WorkflowName.String, string(row.ReadableStatus), "").Inc()
			}
		}
	}

	// send to the tenant queue
	if len(payloads) > 0 {
		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			"workflow-run-finished",
			true,
			false,
			payloads...,
		)

		if err != nil {
			return false, err
		}

		q := msgqueue.TenantEventConsumerQueue(tenantId)

		err = o.mq.SendMessage(ctx, q, msg)

		if err != nil {
			return false, err
		}
	}

	return shouldContinue, nil
}
