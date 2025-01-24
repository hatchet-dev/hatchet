package workflowruntuils

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func CanShortCircuit(workflowRunRow *dbsqlc.GetWorkflowRunsInsertedInThisTxnRow) bool {

	return !(workflowRunRow.ConcurrencyLimitStrategy.Valid || workflowRunRow.ConcurrencyGroupExpression.Valid || workflowRunRow.GetGroupKeyRunId.Valid || workflowRunRow.WorkflowRun.ConcurrencyGroupId.Valid || workflowRunRow.DedupeValue.Valid || workflowRunRow.FailureJob)
}

func NotifyQueues(ctx context.Context, mq msgqueue.MessageQueue, l *zerolog.Logger, repo repository.EngineRepository, tenantId string, workflowRun *repository.CreatedWorkflowRun) error {
	tenant, err := repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		l.Err(err).Msg("could not add message to tenant partition queue")
		return fmt.Errorf("could not get tenant: %w", err)
	}

	if !CanShortCircuit(workflowRun.Row) {
		workflowRunId := sqlchelpers.UUIDToStr(workflowRun.Row.WorkflowRun.ID)

		err = mq.AddMessage(
			ctx,
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunQueuedToTask(
				tenantId,
				workflowRunId,
			),
		)
		if err != nil {
			return fmt.Errorf("could not add workflow run queued task: %w", err)
		}
	} else if tenant.SchedulerPartitionId.Valid {

		for _, queueName := range workflowRun.InitialStepRunQueueNames {

			err = mq.AddMessage(
				ctx,
				msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler),
				tasktypes.CheckTenantQueueToTask(tenantId, queueName, true, false),
			)

			if err != nil {
				l.Err(err).Msg("could not add message to scheduler partition queue")
			}
		}
	}

	return nil
}
