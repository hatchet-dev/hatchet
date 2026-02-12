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
	for _, cb := range callbacks {
		err := tc.processSingleSatisfiedCallback(ctx, tenantId, cb)
		if err != nil {
			tc.l.Error().Err(err).Msgf("failed to process satisfied callback for task %s node %d", cb.DurableTaskExternalId, cb.NodeId)
		}
	}
	return nil
}

func (tc *TasksControllerImpl) processSingleSatisfiedCallback(ctx context.Context, tenantId uuid.UUID, cb v1.SatisfiedCallback) error {
	if cb.DispatcherId == nil {
		return fmt.Errorf("callback has no dispatcher_id set")
	}

	dispatcherId := *cb.DispatcherId

	msg, err := tasktypes.DurableCallbackCompletedMessage(
		tenantId,
		cb.DurableTaskExternalId,
		cb.NodeId,
		// todo: fix this - need the real invocation count here
		0,
		cb.Data,
	)
	if err != nil {
		return fmt.Errorf("failed to create callback completed message: %w", err)
	}

	return tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), msg)
}
