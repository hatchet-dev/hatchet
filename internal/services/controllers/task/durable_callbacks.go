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

	workerIds := make([]uuid.UUID, 0, len(callbacks))
	for _, cb := range callbacks {
		if cb.WorkerId != nil {
			workerIds = append(workerIds, *cb.WorkerId)
		}
	}

	workerIdToDispatcherId, _, err := tc.repov1.Workers().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)
	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	dispatcherToMsgs := make(map[uuid.UUID][]*msgqueue.Message)

	for _, cb := range callbacks {
		if cb.WorkerId == nil {
			tc.l.Error().Msgf("callback has no worker_id set for task %s node %d", cb.DurableTaskExternalId, cb.NodeId)
			continue
		}

		dispatcherId, ok := workerIdToDispatcherId[*cb.WorkerId]
		if !ok {
			tc.l.Error().Msgf("no dispatcher found for worker %s (task %s node %d)", cb.WorkerId, cb.DurableTaskExternalId, cb.NodeId)
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
