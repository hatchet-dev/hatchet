package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (tc *TasksControllerImpl) processSatisfiedEventLogEntry(ctx context.Context, tenantId uuid.UUID, callbacks []v1.SatisfiedEntry) error {
	if len(callbacks) == 0 {
		return nil
	}

	idInsertedAtTuples := make([]v1.IdInsertedAt, 0)

	for _, cb := range callbacks {
		idInsertedAtTuples = append(idInsertedAtTuples, v1.IdInsertedAt{
			ID:         cb.DurableTaskId,
			InsertedAt: cb.DurableTaskInsertedAt,
		})
	}

	idInsertedAtToDispatcherId, err := tc.repov1.Workers().GetDurableDispatcherIdsForTasks(ctx, tenantId, idInsertedAtTuples)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for tasks: %w", err)
	}

	dispatcherToMsgs := make(map[uuid.UUID][]*msgqueue.Message)

	for _, cb := range callbacks {
		key := v1.IdInsertedAt{
			ID:         cb.DurableTaskId,
			InsertedAt: cb.DurableTaskInsertedAt,
		}

		dispatcherId, ok := idInsertedAtToDispatcherId[key]

		if !ok {
			// TODO-DURABLE: does invocation count matter here?
			tc.l.Warn().Msgf("no dispatcher for task %d (evicted or worker gone), publishing restore message", cb.DurableTaskId)

			restoreMsg, err := tasktypes.DurableRestoreTaskMessage(tenantId, cb.DurableTaskExternalId, "callback satisfied while task evicted")
			if err != nil {
				tc.l.Error().Err(err).Msgf("failed to create restore message for task %s", cb.DurableTaskExternalId)
				continue
			}

			if err := tc.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, restoreMsg); err != nil {
				tc.l.Error().Err(err).Msgf("failed to publish restore message for task %s", cb.DurableTaskExternalId)
			}
			continue
		}

		msg, err := tasktypes.DurableCallbackCompletedMessage(
			tenantId,
			cb.DurableTaskExternalId,
			cb.InvocationCount,
			cb.BranchId,
			cb.NodeId,
			cb.Data,
		)
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to create callback completed message for task %s node %d", cb.DurableTaskExternalId, cb.NodeId)
			continue
		}

		dispatcherToMsgs[dispatcherId] = append(dispatcherToMsgs[dispatcherId], msg)
	}

	for dispatcherId, msgs := range dispatcherToMsgs {
		for _, m := range msgs {
			if err := tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), m); err != nil {
				tc.l.Error().Err(err).Msgf("failed to send callback completed message to dispatcher %s", dispatcherId)
			}
		}
	}

	return nil
}

func (tc *TasksControllerImpl) handleDurableRestoreTask(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.DurableRestoreTaskPayload](payloads)

	for _, msg := range msgs {
		task, err := tc.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, msg.TaskExternalId, true)
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to look up task %s for restore", msg.TaskExternalId)
			continue
		}

		requeued, queue, err := tc.repov1.Tasks().RestoreEvictedTask(ctx, tenantId, v1.TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to restore task %s", msg.TaskExternalId)
			continue
		}

		if !requeued {
			tc.l.Warn().Msgf("task %s was not requeued (not evicted or already queued)", msg.TaskExternalId)
			continue
		}

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventTimestamp: time.Now(),
				EventType:      sqlcv1.V1EventTypeOlapDURABLERESTORING,
				EventMessage:   fmt.Sprintf("Restoring evicted task: %s", msg.Reason),
			},
		)
		if err == nil {
			if pubErr := tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false); pubErr != nil {
				tc.l.Warn().Err(pubErr).Msg("failed to publish DURABLE_RESTORING to OLAP")
			}
		}

		tc.notifySchedulerQueue(ctx, tenantId, queue)
	}

	return nil
}

func (tc *TasksControllerImpl) notifySchedulerQueue(ctx context.Context, tenantId uuid.UUID, queue string) {
	if queue == "" {
		return
	}

	tenant, err := tc.repov1.Tenant().GetTenantByID(ctx, tenantId)
	if err != nil {
		tc.l.Error().Err(err).Msg("could not get tenant for scheduler notification")
		return
	}

	if !tenant.SchedulerPartitionId.Valid {
		return
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		tasktypes.CheckTenantQueuesPayload{
			QueueNames: []string{queue},
		},
	)
	if err != nil {
		tc.l.Error().Err(err).Msgf("failed to build check-tenant-queue message for queue %s", queue)
		return
	}

	if err := tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler), msg); err != nil {
		tc.l.Error().Err(err).Msgf("failed to notify scheduler for queue %s", queue)
	}
}
