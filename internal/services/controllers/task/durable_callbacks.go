package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (tc *TasksControllerImpl) processSatisfiedCallbacks(ctx context.Context, tenantId uuid.UUID, callbacks []v1.SatisfiedCallback) error {
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
			tc.l.Warn().Msgf("could not find dispatcher id for task %d inserted at %s, skipping callback", cb.DurableTaskId, cb.DurableTaskInsertedAt.Time)
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
