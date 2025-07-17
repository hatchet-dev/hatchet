package olap

import (
	"context"
	"strings"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
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

		tenantIds := make([]string, len(tenants))
		for i, tenant := range tenants {
			tenantIds[i] = sqlchelpers.UUIDToStr(tenant.ID)
		}

		if len(tenantIds) == 0 {
			o.l.Warn().Msg("no tenants found for task status updates")
			return
		}

		underscoreDelimitedTenantIds := strings.Join(tenantIds, "_")

		o.updateTaskStatusOperations.RunOrContinue(underscoreDelimitedTenantIds)
	}
}

func (o *OLAPControllerImpl) updateTaskStatuses(ctx context.Context, underscoreDelimitedTenantIds string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-task-statuses")
	defer span.End()

	if len(underscoreDelimitedTenantIds) == 0 {
		o.l.Warn().Msg("no tenant IDs provided for updating task statuses")
		return false, nil
	}

	tenantIds := strings.Split(underscoreDelimitedTenantIds, "_")

	shouldContinue, rows, err := o.repo.OLAP().UpdateTaskStatuses(ctx, tenantIds)

	if err != nil {
		return false, err
	}

	tenantIdToPayloads := make(map[pgtype.UUID][]tasktypes.NotifyFinalizedPayload)
	workflowIds := make([]pgtype.UUID, 0, len(rows))
	workerIds := make([]pgtype.UUID, 0, len(rows))
	taskIds := make([]int64, 0, len(rows))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(rows))
	readableStatuses := make([]sqlcv1.V1ReadableStatusOlap, 0, len(rows))

	for _, row := range rows {
		if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
			continue
		}

		tenantIdToPayloads[row.TenantId] = append(tenantIdToPayloads[row.TenantId], tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		taskIds = append(taskIds, row.TaskId)
		taskInsertedAts = append(taskInsertedAts, row.TaskInsertedAt)
		workflowIds = append(workflowIds, row.WorkflowId)
		workerIds = append(workerIds, row.LatestWorkerId) // TODO(mnafees): use this information in workflow metrics below
		readableStatuses = append(readableStatuses, row.ReadableStatus)
	}

	if o.prometheusMetricsEnabled {
		for _, tenantId := range tenantIds {
			workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
			if err != nil {
				return false, err
			}

			taskDurations, err := o.repo.OLAP().GetTaskDurationsByTaskIds(ctx, tenantId, taskIds, taskInsertedAts, readableStatuses)
			if err != nil {
				return false, err
			}

			for _, row := range rows {
				// Only track metrics for standalone tasks, not tasks within DAGs
				// DAG-level metrics are tracked in process_dag_status_updates.go
				if !row.IsDAGTask {
					workflowName := workflowNames[row.WorkflowId]
					if workflowName == "" {
						continue
					}

					taskDuration := taskDurations[row.TaskId]
					if taskDuration == nil || !taskDuration.StartedAt.Valid || !taskDuration.FinishedAt.Valid {
						continue
					}

					prometheus.TenantWorkflowDurationBuckets.WithLabelValues(tenantId, workflowName, string(row.ReadableStatus)).Observe(float64(taskDuration.FinishedAt.Time.Sub(taskDuration.StartedAt.Time).Milliseconds()))
				}
			}
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
				return false, err
			}

			q := msgqueue.TenantEventConsumerQueue(tenantId.String())

			err = o.mq.SendMessage(ctx, q, msg)

			if err != nil {
				return false, err
			}
		}
	}

	return shouldContinue, nil
}
