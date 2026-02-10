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
			tc.l.Error().Err(err).Msgf("failed to process satisfied callback %s", cb.CallbackKey)
		}
	}
	return nil
}

func (tc *TasksControllerImpl) processSingleSatisfiedCallback(ctx context.Context, tenantId uuid.UUID, cb v1.SatisfiedCallback) error {
	if cb.DispatcherId == nil {
		return fmt.Errorf("callback %s has no dispatcher_id set", cb.CallbackKey)
	}

	dispatcherId := *cb.DispatcherId

	callbackKey, err := tc.repov1.DurableEvents().ParseCallbackKey(ctx, cb.CallbackKey)
	if err != nil {
		return fmt.Errorf("failed to parse callback key %s: %w", cb.CallbackKey, err)
	}

	msg, err := tasktypes.DurableCallbackCompletedMessage(
		tenantId,
		*callbackKey,
		1,
		cb.Data,
	)
	if err != nil {
		return fmt.Errorf("failed to create callback completed message: %w", err)
	}

	return tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), msg)
}
