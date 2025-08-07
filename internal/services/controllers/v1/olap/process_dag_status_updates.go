package olap

import (
	"context"
	"time"

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
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

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
	}

	if o.prometheusMetricsEnabled {
		var eg errgroup.Group

		for currentTenantId, workflowIds := range tenantIdToWorkflowIds {
			tenantId := currentTenantId
			workflowIds := workflowIds

			var tenantRows []v1.UpdateDAGStatusRow

			dagExternalIds := make([]pgtype.UUID, 0, len(rows))
			var minInsertedAt pgtype.Timestamptz

			for _, row := range rows {
				if sqlchelpers.UUIDToStr(row.TenantId) == tenantId {
					tenantRows = append(tenantRows, row)

					dagExternalIds = append(dagExternalIds, row.ExternalId)

					if !minInsertedAt.Valid || row.DagInsertedAt.Time.Before(minInsertedAt.Time) {
						minInsertedAt = row.DagInsertedAt
					}
				}
			}

			eg.Go(func() error {
				workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
				if err != nil {
					return err
				}

				dagDurations, err := o.repo.OLAP().GetDAGDurations(ctx, tenantId, dagExternalIds, minInsertedAt)
				if err != nil {
					return err
				}

				for _, row := range tenantRows {
					if row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED || row.ReadableStatus == sqlcv1.V1ReadableStatusOlapCANCELLED {
						workflowName := workflowNames[row.WorkflowId]
						if workflowName == "" {
							continue
						}

						dagDuration := dagDurations[sqlchelpers.UUIDToStr(row.ExternalId)]
						if dagDuration == nil || !dagDuration.StartedAt.Valid || !dagDuration.FinishedAt.Valid {
							continue
						}

						duration := int(dagDuration.FinishedAt.Time.Sub(dagDuration.StartedAt.Time).Milliseconds())
						prometheus.TenantWorkflowDurationBuckets.WithLabelValues(tenantId, workflowName, string(row.ReadableStatus)).Observe(float64(duration))
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
