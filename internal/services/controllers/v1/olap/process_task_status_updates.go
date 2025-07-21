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

func (o *OLAPControllerImpl) runTaskStatusUpdates(ctx context.Context) func() {
	return func() {
		shouldContinue := true

		for shouldContinue {
			o.l.Debug().Msgf("partition: running status updates for tasks")

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

			var rows []v1.UpdateTaskStatusRow

			shouldContinue, rows, err = o.repo.OLAP().UpdateTaskStatuses(ctx, tenantIds)

			if err != nil {
				o.l.Error().Err(err).Msg("could not update DAG statuses")
				return
			}

			err = o.notifyTasksUpdated(ctx, rows)

			if err != nil {
				o.l.Error().Err(err).Msg("failed to notify updated DAG statuses")
				return
			}
		}
	}
}

func (o *OLAPControllerImpl) notifyTasksUpdated(ctx context.Context, rows []v1.UpdateTaskStatusRow) error {
	tenantIdToPayloads := make(map[pgtype.UUID][]tasktypes.NotifyFinalizedPayload)
	tenantIdToWorkflowIds := make(map[string][]pgtype.UUID)

	for _, row := range rows {
		if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
			continue
		}

		tenantIdToPayloads[row.TenantId] = append(tenantIdToPayloads[row.TenantId], tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		tenantId := sqlchelpers.UUIDToStr(row.TenantId)
		tenantIdToWorkflowIds[tenantId] = append(tenantIdToWorkflowIds[tenantId], row.WorkflowId)
	}

	if o.prometheusMetricsEnabled {
		var eg errgroup.Group

		for currentTenantId, workflowIds := range tenantIdToWorkflowIds {
			tenantId := currentTenantId
			workflowIds := workflowIds

			var tenantTaskIds []int64
			var tenantTaskInsertedAts []pgtype.Timestamptz
			var tenantReadableStatuses []sqlcv1.V1ReadableStatusOlap
			var tenantRows []v1.UpdateTaskStatusRow

			for _, row := range rows {
				if sqlchelpers.UUIDToStr(row.TenantId) == tenantId {
					tenantTaskIds = append(tenantTaskIds, row.TaskId)
					tenantTaskInsertedAts = append(tenantTaskInsertedAts, row.TaskInsertedAt)
					tenantReadableStatuses = append(tenantReadableStatuses, row.ReadableStatus)
					tenantRows = append(tenantRows, row)
				}
			}

			eg.Go(func() error {
				workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
				if err != nil {
					return err
				}

				taskDurations, err := o.repo.OLAP().GetTaskDurationsByTaskIds(ctx, tenantId, tenantTaskIds, tenantTaskInsertedAts, tenantReadableStatuses)
				if err != nil {
					return err
				}

				for _, row := range tenantRows {
					if !row.IsDAGTask {
						workflowName := workflowNames[row.WorkflowId]
						if workflowName == "" {
							continue
						}

						taskDuration := taskDurations[row.TaskId]
						if taskDuration == nil || !taskDuration.StartedAt.Valid || !taskDuration.FinishedAt.Valid {
							continue
						}

						duration := int(taskDuration.FinishedAt.Time.Sub(taskDuration.StartedAt.Time).Milliseconds())
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
