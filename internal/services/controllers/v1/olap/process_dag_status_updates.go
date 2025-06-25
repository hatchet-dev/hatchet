package olap

import (
	"context"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

func (o *OLAPControllerImpl) runTenantDAGStatusUpdates(ctx context.Context) func() {
	return func() {
		o.l.Debug().Msgf("partition: running status updates for dags")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			o.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		o.updateDAGStatusOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			o.updateDAGStatusOperations.RunOrContinue(tenantId)
		}
	}
}

func (o *OLAPControllerImpl) updateDAGStatuses(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-dag-statuses")
	defer span.End()

	shouldContinue, rows, err := o.repo.OLAP().UpdateDAGStatuses(ctx, tenantId)

	if err != nil {
		return false, err
	}

	payloads := make([]tasktypes.NotifyFinalizedPayload, 0, len(rows))

	var workflowIds []pgtype.UUID
	var dagIds []int64
	var dagInsertedAts []pgtype.Timestamptz

	for _, row := range rows {
		payloads = append(payloads, tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED {
			o.processTenantAlertOperations.RunOrContinue(tenantId)
		}

		workflowIds = append(workflowIds, row.WorkflowId)
		dagIds = append(dagIds, row.DagId)
		dagInsertedAts = append(dagInsertedAts, row.DagInsertedAt)
	}

	workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
	if err != nil {
		return false, err
	}

	dagDurations, err := o.repo.OLAP().GetDagDurationsByDagIds(ctx, tenantId, dagIds, dagInsertedAts)
	if err != nil {
		return false, err
	}

	for i, row := range rows {
		if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCANCELLED {
			workflowName := workflowNames[row.WorkflowId]
			if workflowName == "" {
				continue
			}

			dagDuration := dagDurations[i]

			tenantMetric := prometheus.WithTenant(tenantId)

			tenantMetric.WorkflowDuration.WithLabelValues(tenantId, workflowName, string(row.ReadableStatus)).Observe(float64(dagDuration.FinishedAt.Time.Sub(dagDuration.StartedAt.Time).Milliseconds()))
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
