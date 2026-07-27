package durable

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func DispatchCallbacks(ctx context.Context, l *zerolog.Logger, mq msgqueue.MessageQueue, repo v1.Repository, tenantId uuid.UUID, callbacks []v1.SatisfiedEntry) error {
	if len(callbacks) == 0 {
		return nil
	}

	idInsertedAtTuples := make([]v1.IdInsertedAt, 0, len(callbacks))

	for _, cb := range callbacks {
		idInsertedAtTuples = append(idInsertedAtTuples, v1.IdInsertedAt{
			ID:         cb.DurableTaskId,
			InsertedAt: cb.DurableTaskInsertedAt,
		})
	}

	idInsertedAtToDispatcherId, err := repo.Workers().GetDurableDispatcherIdsForTasks(ctx, tenantId, idInsertedAtTuples)

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
			l.Warn().Msgf("no runtime/dispatcher lookup row for task %d, skipping callback delivery", cb.DurableTaskId)
			continue
		}

		if dispatcherLookup.IsEvicted {
			l.Debug().Msgf("task %d is evicted, publishing restore message", cb.DurableTaskId)

			restoreMsg, err := tasktypes.DurableRestoreTaskMessage(tenantId, cb.DurableTaskExternalId, "callback satisfied while task evicted")
			if err != nil {
				l.Error().Err(err).Msgf("failed to create restore message for task %s", cb.DurableTaskExternalId.String())
				continue
			}

			if err := mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, restoreMsg); err != nil {
				l.Error().Err(err).Msgf("failed to publish restore message for task %s", cb.DurableTaskExternalId.String())
			}
			continue
		}

		dispatcherId := dispatcherLookup.DispatcherId
		if dispatcherId == nil {
			l.Warn().Msgf("task %d has runtime but no durable dispatcher id, skipping callback delivery", cb.DurableTaskId)
			continue
		}

		msg, err := tasktypes.DurableCallbackCompletedMessage(
			tenantId,
			cb.DurableTaskExternalId,
			cb.InvocationCount,
			cb.BranchId,
			cb.NodeId,
			cb.Data,
			cb.ChildTaskIsFailure,
			cb.ChildTaskErrorMessage,
		)
		if err != nil {
			l.Error().Err(err).Msgf("failed to create callback completed message for task %s node %d", cb.DurableTaskExternalId.String(), cb.NodeId)
			continue
		}

		dispatcherToMsgs[*dispatcherId] = append(dispatcherToMsgs[*dispatcherId], msg)
	}

	for dispatcherId, msgs := range dispatcherToMsgs {
		for _, m := range msgs {
			if err := mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), m); err != nil {
				l.Error().Err(err).Msgf("failed to send callback completed message to dispatcher %s", dispatcherId.String())
			}
		}
	}

	return nil
}
