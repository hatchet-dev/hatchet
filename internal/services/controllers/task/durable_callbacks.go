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

	tc.l.Warn().Msgf("processSatisfiedEventLogEntry: processing %d satisfied callbacks", len(callbacks))

	idInsertedAtTuples := make([]v1.IdInsertedAt, 0)

	for _, cb := range callbacks {
		tc.l.Warn().Msgf("  satisfied entry: durableTaskExternalId=%s durableTaskId=%d nodeId=%d", cb.DurableTaskExternalId, cb.DurableTaskId, cb.NodeId)
		idInsertedAtTuples = append(idInsertedAtTuples, v1.IdInsertedAt{
			ID:         cb.DurableTaskId,
			InsertedAt: cb.DurableTaskInsertedAt,
		})
	}

	idInsertedAtToDispatcherId, err := tc.repov1.Workers().GetDurableDispatcherIdsForTasks(ctx, tenantId, idInsertedAtTuples)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for tasks: %w", err)
	}

	tc.l.Warn().Msgf("  found %d dispatcher mappings for %d tasks", len(idInsertedAtToDispatcherId), len(idInsertedAtTuples))

	dispatcherToMsgs := make(map[uuid.UUID][]*msgqueue.Message)

	for _, cb := range callbacks {
		key := v1.IdInsertedAt{
			ID:         cb.DurableTaskId,
			InsertedAt: cb.DurableTaskInsertedAt,
		}

		dispatcherId, ok := idInsertedAtToDispatcherId[key]

		if !ok {
			tc.l.Warn().Msgf("could not find dispatcher id for task %d inserted at %s, publishing restore", cb.DurableTaskId, cb.DurableTaskInsertedAt.Time)

			restoreMsg, err := tasktypes.DurableRestoreTaskMessage(tenantId, cb.DurableTaskExternalId, "callback satisfied, no dispatcher")
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

	tc.l.Warn().Msgf("handleDurableRestoreTask: received %d restore messages", len(msgs))

	for _, msg := range msgs {
		tc.l.Warn().Msgf("  restoring task externalId=%s reason=%s", msg.TaskExternalId, msg.Reason)

		task, err := tc.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, msg.TaskExternalId, true)
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to look up task %s for durable restore", msg.TaskExternalId)
			continue
		}

		tc.l.Warn().Msgf("  looked up task: id=%d retryCount=%d", task.ID, task.RetryCount)

		requeued, queue, err := tc.repov1.Tasks().RestoreEvictedTask(ctx, tenantId, v1.TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to restore evicted task %s", msg.TaskExternalId)
			continue
		}

		tc.l.Warn().Msgf("  RestoreEvictedTask result: requeued=%v queue=%s", requeued, queue)

		if !requeued {
			tc.l.Warn().Msgf("  task %s was not requeued (not evicted or already restored)", msg.TaskExternalId)
			continue
		}

		tc.l.Info().Msgf("restored evicted task %s (reason: %s)", msg.TaskExternalId, msg.Reason)

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventTimestamp: time.Now(),
				EventType:      sqlcv1.V1EventTypeOlapDURABLERESTORING,
				EventMessage:   msg.Reason,
			},
		)
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to create DURABLE_RESTORING monitoring event for task %s", msg.TaskExternalId)
			continue
		}

		if err := tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false); err != nil {
			tc.l.Error().Err(err).Msgf("failed to publish DURABLE_RESTORING event for task %s", msg.TaskExternalId)
		}

		if queue != "" {
			tc.notifySchedulerQueue(ctx, tenantId, queue)
		}
	}

	return nil
}

func (tc *TasksControllerImpl) notifySchedulerQueue(ctx context.Context, tenantId uuid.UUID, queue string) {
	tenant, err := tc.repov1.Tenant().GetTenantByID(ctx, tenantId)
	if err != nil {
		tc.l.Error().Err(err).Msg("could not get tenant for scheduler notification")
		return
	}

	if !tenant.SchedulerPartitionId.Valid {
		return
	}

	payload := tasktypes.CheckTenantQueuesPayload{
		QueueNames: []string{queue},
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		payload,
	)
	if err != nil {
		tc.l.Error().Err(err).Msg("could not create check-tenant-queue message")
		return
	}

	if err := tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler), msg); err != nil {
		tc.l.Error().Err(err).Msg("could not send check-tenant-queue message to scheduler")
	}
}
