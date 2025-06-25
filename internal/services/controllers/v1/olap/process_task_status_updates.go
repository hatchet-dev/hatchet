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

	workflowIds := make([]pgtype.UUID, 0, len(rows))
	workerIds := make([]pgtype.UUID, 0, len(rows))
	taskIds := make([]int64, 0, len(rows))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(rows))

	for _, row := range rows {
		if row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCOMPLETED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapCANCELLED && row.ReadableStatus != sqlcv1.V1ReadableStatusOlapFAILED {
			continue
		}

		payloads = append(payloads, tasktypes.NotifyFinalizedPayload{
			ExternalId: sqlchelpers.UUIDToStr(row.ExternalId),
			Status:     row.ReadableStatus,
		})

		taskIds = append(taskIds, row.TaskId)
		taskInsertedAts = append(taskInsertedAts, row.TaskInsertedAt)
		workflowIds = append(workflowIds, row.WorkflowId)
		workerIds = append(workerIds, row.LatestWorkerId) // TODO(mnafees): use this information in workflow metrics below
	}

	workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(ctx, tenantId, workflowIds)
	if err != nil {
		return false, err
	}

	taskDurations, err := o.repo.OLAP().GetTaskDurationsByTaskIds(ctx, tenantId, taskIds, taskInsertedAts)
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

			tenantMetric := prometheus.WithTenant(tenantId)

			tenantMetric.WorkflowDuration.WithLabelValues(tenantId, workflowName, string(row.ReadableStatus)).Observe(float64(taskDuration.FinishedAt.Time.Sub(taskDuration.StartedAt.Time).Milliseconds()))
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
