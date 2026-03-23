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

		dispatcherLookup, ok := idInsertedAtToDispatcherId[key]

		if !ok {
			tc.l.Warn().Msgf("no runtime/dispatcher lookup row for task %d, skipping callback delivery", cb.DurableTaskId)
			continue
		}

		if dispatcherLookup.IsEvicted {
			tc.l.Debug().Msgf("task %d is evicted, publishing restore message", cb.DurableTaskId)

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

		dispatcherId := dispatcherLookup.DispatcherId
		if dispatcherId == nil {
			tc.l.Warn().Msgf("task %d has runtime but no durable dispatcher id, skipping callback delivery", cb.DurableTaskId)
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

		dispatcherToMsgs[*dispatcherId] = append(dispatcherToMsgs[*dispatcherId], msg)
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

	externalIds := make([]uuid.UUID, 0, len(msgs))
	reasonByExternalId := make(map[uuid.UUID]string, len(msgs))
	for _, msg := range msgs {
		externalIds = append(externalIds, msg.TaskExternalId)
		reasonByExternalId[msg.TaskExternalId] = msg.Reason
	}

	flatTasks, err := tc.repov1.Tasks().FlattenExternalIds(ctx, tenantId, externalIds)
	if err != nil {
		return fmt.Errorf("failed to batch-lookup tasks for restore: %w", err)
	}

	if len(flatTasks) == 0 {
		return nil
	}

	tasksToRestore := make([]v1.TaskIdInsertedAtRetryCount, 0, len(flatTasks))
	for _, t := range flatTasks {
		tasksToRestore = append(tasksToRestore, v1.TaskIdInsertedAtRetryCount{
			Id:         t.ID,
			InsertedAt: t.InsertedAt,
			RetryCount: t.RetryCount,
		})
	}

	restoredRows, err := tc.repov1.Tasks().RestoreEvictedTasks(ctx, tenantId, tasksToRestore)
	if err != nil {
		return fmt.Errorf("failed to batch-restore evicted tasks: %w", err)
	}

	restoredByTaskId := make(map[int64]*sqlcv1.RestoreEvictedTasksRow, len(restoredRows))
	for _, r := range restoredRows {
		restoredByTaskId[r.TaskID] = r
	}

	invCountOpts := make([]v1.IdInsertedAt, 0, len(flatTasks))
	for _, t := range flatTasks {
		invCountOpts = append(invCountOpts, v1.IdInsertedAt{ID: t.ID, InsertedAt: t.InsertedAt})
	}

	invocationCounts, err := tc.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, invCountOpts)
	if err != nil {
		return fmt.Errorf("failed to get durable task invocation counts for restoring tasks: %w", err)
	}

	queues := make(map[string]struct{})

	for _, t := range flatTasks {
		restored, ok := restoredByTaskId[t.ID]
		if !ok || !restored.Queued {
			tc.l.Warn().Msgf("task %s was not requeued (not evicted or already queued)", t.ExternalID)
			continue
		}

		var durableInvCount int32
		if count, ok := invocationCounts[v1.IdInsertedAt{ID: t.ID, InsertedAt: t.InsertedAt}]; ok && count != nil {
			durableInvCount = *count
		}

		reason := reasonByExternalId[t.ExternalID]

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:                 t.ID,
				RetryCount:             t.RetryCount,
				DurableInvocationCount: durableInvCount,
				EventTimestamp:         time.Now(),
				EventType:              sqlcv1.V1EventTypeOlapDURABLERESTORING,
				EventMessage:           fmt.Sprintf("Restoring evicted task: %s", reason),
			},
		)
		if err == nil {
			if pubErr := tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false); pubErr != nil {
				tc.l.Warn().Err(pubErr).Msg("failed to publish DURABLE_RESTORING to OLAP")
			}
		}

		if restored.Queue != "" {
			queues[restored.Queue] = struct{}{}
		} else {
			tc.l.Warn().Str("task_id", t.ExternalID.String()).Msg("restored task has empty queue, skipping scheduler notification")
		}
	}

	if len(queues) > 0 {
		if err := tc.notifySchedulerQueues(ctx, tenantId, queues); err != nil {
			tc.l.Error().Err(err).Msg("failed to notify scheduler queues")
		}
	}

	return nil
}

func (tc *TasksControllerImpl) notifySchedulerQueues(ctx context.Context, tenantId uuid.UUID, queues map[string]struct{}) error {
	tenant, err := tc.repov1.Tenant().GetTenantByID(ctx, tenantId)
	if err != nil {
		return fmt.Errorf("could not get tenant for scheduler notification: %w", err)
	}

	if !tenant.SchedulerPartitionId.Valid {
		return nil
	}

	queueNames := make([]string, 0, len(queues))
	for q := range queues {
		queueNames = append(queueNames, q)
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		tasktypes.CheckTenantQueuesPayload{
			QueueNames: queueNames,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build check-tenant-queue message for queues %v: %w", queueNames, err)
	}

	if err := tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler), msg); err != nil {
		return fmt.Errorf("failed to notify scheduler for queues %v: %w", queueNames, err)
	}

	return nil
}
