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

	taskExternalId, nodeId, invocationCount, err := parseCallbackKey(cb.CallbackKey)
	if err != nil {
		return fmt.Errorf("failed to parse callback key %s: %w", cb.CallbackKey, err)
	}

	msg, err := tasktypes.DurableCallbackCompletedMessage(
		tenantId,
		taskExternalId,
		nodeId,
		invocationCount,
		cb.Data,
	)
	if err != nil {
		return fmt.Errorf("failed to create callback completed message: %w", err)
	}

	return tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), msg)
}

func parseCallbackKey(key string) (string, int64, int64, error) {
	var nodeId, invocationCount int64
	parts := make([]string, 0)
	current := ""
	for _, c := range key {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)

	if len(parts) != 3 {
		return "", 0, 0, fmt.Errorf("invalid callback key format: %s (expected taskExternalId:nodeId:invocationCount)", key)
	}

	_, err := fmt.Sscanf(parts[1], "%d", &nodeId)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid node id in callback key: %w", err)
	}

	_, err = fmt.Sscanf(parts[2], "%d", &invocationCount)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid invocation count in callback key: %w", err)
	}

	return parts[0], nodeId, invocationCount, nil
}
