package task

import (
	"context"
	"fmt"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	"golang.org/x/sync/errgroup"
)

func (tc *TasksControllerImpl) runTenantTimeoutTasks(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running timeout for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.timeoutTaskOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.timeoutTaskOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processTaskTimeouts(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-timeout")
	defer span.End()

	tasks, shouldContinue, err := tc.repov1.Tasks().ProcessTaskTimeouts(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	if num := len(tasks); num > 0 {
		tc.l.Info().Msgf("timed out %d step runs", num)
	}

	cancellationSignals := make([]tasktypes.SignalTaskCancelledPayload, 0, len(tasks))

	cancelPayloads := make([]tasktypes.CancelledTaskPayload, 0, len(tasks))

	for _, task := range tasks {
		cancellationSignals = append(cancellationSignals, tasktypes.SignalTaskCancelledPayload{
			TaskId:     task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
			WorkerId:   sqlchelpers.UUIDToStr(task.WorkerID),
		})

		cancelPayloads = append(cancelPayloads, tasktypes.CancelledTaskPayload{
			TaskId:        task.ID,
			InsertedAt:    task.InsertedAt,
			RetryCount:    task.RetryCount,
			ExternalId:    sqlchelpers.UUIDToStr(task.ExternalID),
			WorkflowRunId: sqlchelpers.UUIDToStr(task.WorkflowRunID),
			EventType:     sqlcv1.V1EventTypeOlapTIMEDOUT,
			ShouldNotify:  true,
		})
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		if len(cancellationSignals) == 0 {
			return nil
		}

		return tc.sendTaskCancellationsToDispatcher(ctx, tenantId, cancellationSignals)
	})

	eg.Go(func() error {
		if len(cancelPayloads) == 0 {
			return nil
		}

		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			"task-cancelled",
			false,
			true,
			cancelPayloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create cancel tasks message: %w", err)
		}

		return tc.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)
	})

	return shouldContinue, eg.Wait()
}
