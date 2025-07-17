package olap

import (
	"context"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
)

func (o *OLAPControllerImpl) runDAGStatusUpdates(ctx context.Context) func() {
	return func() {
		shouldContinue := true

		for shouldContinue {
			o.l.Debug().Msgf("partition: running status updates for dags")

			// list all tenants
			tenants, err := o.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

			if err != nil {
				o.l.Error().Err(err).Msg("could not list tenants")
				return
			}

			tenantIds := make([]string, 0, len(tenants))

			for _, tenant := range tenants {
				tenantId := sqlchelpers.UUIDToStr(tenant.ID)
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
	tenantIdToPayloads := make(map[pgtype.UUID][]tasktypes.NotifyFinalizedPayload)
	tenantIdToWorkflowIds := make(map[string][]pgtype.UUID)
	dagIds := make([]int64, 0, len(rows))
	dagInsertedAts := make([]pgtype.Timestamptz, 0, len(rows))
	readableStatuses := make([]sqlcv1.V1ReadableStatusOlap, 0, len(rows))

	for _, row := range rows {
		tenantIdToPayloads[row.TenantId] = append(tenantIdToPayloads[row.TenantId], tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED {
			o.processTenantAlertOperations.RunOrContinue(row.TenantId.String())
		}

		tenantId := sqlchelpers.UUIDToStr(row.TenantId)

		tenantIdToWorkflowIds[tenantId] = append(tenantIdToWorkflowIds[tenantId], row.WorkflowId)
		dagIds = append(dagIds, row.DagId)
		dagInsertedAts = append(dagInsertedAts, row.DagInsertedAt)
		readableStatuses = append(readableStatuses, row.ReadableStatus)
	}

	if o.prometheusMetricsEnabled {
		var eg errgroup.Group

		for tenantId, workflowIds := range tenantIdToWorkflowIds {
			eg.Go(func() error {
				workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
				if err != nil {
					return err
				}

				dagDurations, err := o.repo.OLAP().GetDagDurationsByDagIds(ctx, tenantId, dagIds, dagInsertedAts, readableStatuses)
				if err != nil {
					return err
				}

				for i, row := range rows {
					if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCANCELLED {
						workflowName := workflowNames[row.WorkflowId]
						if workflowName == "" {
							continue
						}

						dagDuration := dagDurations[i]

						prometheus.TenantWorkflowDurationBuckets.WithLabelValues(tenantId, workflowName, string(row.ReadableStatus)).Observe(float64(dagDuration.FinishedAt.Time.Sub(dagDuration.StartedAt.Time).Milliseconds()))
					}
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	// send to the tenant queue
	if len(tenantIdToPayloads) > 0 {
		for tenantId, payloads := range tenantIdToPayloads {
			msg, err := msgqueue.NewTenantMessage(
				tenantId.String(),
				"workflow-run-finished",
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
