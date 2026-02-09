package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (tc *TasksControllerImpl) handleRegisterDurableCallback(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.RegisterDurableCallbackPayload](payloads)

	for _, msg := range msgs {
		if err := tc.processRegisterDurableCallback(ctx, tenantId, msg); err != nil {
			tc.l.Error().Err(err).Msgf("failed to process register durable callback for task %s", msg.TaskExternalId)
		}
	}

	return nil
}

func (tc *TasksControllerImpl) processRegisterDurableCallback(ctx context.Context, tenantId uuid.UUID, msg *tasktypes.RegisterDurableCallbackPayload) error {
	taskExternalId, err := uuid.Parse(msg.TaskExternalId)
	if err != nil {
		return fmt.Errorf("invalid task external id: %w", err)
	}

	task, err := tc.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskExternalId, false)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	callbackKey := fmt.Sprintf("%s:%d", msg.TaskExternalId, msg.NodeId)

	existingCallback, err := tc.repov1.DurableEvents().GetEventLogCallback(ctx, tenantId, task.ID, task.InsertedAt, callbackKey)
	if err == nil && existingCallback != nil && existingCallback.Callback.IsSatisfied {
		cbMsg, err := tasktypes.DurableCallbackCompletedMessage(
			tenantId,
			msg.TaskExternalId,
			msg.NodeId,
			msg.InvocationCount,
			existingCallback.Result,
		)
		if err != nil {
			return fmt.Errorf("failed to create callback completed message: %w", err)
		}

		return tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(msg.DispatcherId), cbMsg)
	}

	entryWithData, err := tc.repov1.DurableEvents().GetEventLogEntry(ctx, tenantId, task.ID, task.InsertedAt, msg.NodeId)
	if err != nil {
		return fmt.Errorf("event log entry not found for node_id %d: %w", msg.NodeId, err)
	}

	var callbackKind sqlcv1.V1DurableEventLogCallbackKind
	switch entryWithData.Entry.Kind.V1DurableEventLogEntryKind {
	case sqlcv1.V1DurableEventLogEntryKindRUNTRIGGERED:
		callbackKind = sqlcv1.V1DurableEventLogCallbackKindRUNCOMPLETED
	default:
		callbackKind = sqlcv1.V1DurableEventLogCallbackKindWAITFORCOMPLETED
	}

	if existingCallback == nil {
		now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())
		externalId := uuid.New()

		_, err = tc.repov1.DurableEvents().CreateEventLogCallbacks(ctx, []v1.CreateEventLogCallbackOpts{{
			TenantId:              tenantId,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			InsertedAt:            now,
			Kind:                  callbackKind,
			Key:                   callbackKey,
			NodeId:                msg.NodeId,
			IsSatisfied:           false,
			ExternalId:            externalId,
			DispatcherId:          msg.DispatcherId,
		}})
		if err != nil {
			return fmt.Errorf("failed to create callback: %w", err)
		}
	}

	signalKey := fmt.Sprintf("durable:%s:%d", msg.TaskExternalId, msg.NodeId)

	err = tc.repov1.Matches().LinkCallbackToMatch(
		ctx, tenantId,
		task.ID, task.InsertedAt, signalKey,
		task.ID, task.InsertedAt, callbackKey,
	)
	if err != nil {
		tc.l.Warn().Err(err).Msgf("failed to link callback to match (match may not exist yet or already satisfied)")
	}

	return nil
}

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
	callback, err := tc.repov1.DurableEvents().UpdateEventLogCallbackSatisfied(
		ctx,
		tenantId,
		cb.DurableTaskId,
		cb.DurableTaskInsertedAt,
		cb.CallbackKey,
		true,
		cb.Data,
	)
	if err != nil {
		return fmt.Errorf("failed to update callback as satisfied: %w", err)
	}

	if callback.DispatcherID == nil {
		return fmt.Errorf("callback %s has no dispatcher_id set", cb.CallbackKey)
	}

	dispatcherId := *callback.DispatcherID

	taskExternalId, nodeId, err := parseCallbackKey(cb.CallbackKey)
	if err != nil {
		return fmt.Errorf("failed to parse callback key %s: %w", cb.CallbackKey, err)
	}

	msg, err := tasktypes.DurableCallbackCompletedMessage(
		tenantId,
		taskExternalId,
		nodeId,
		0,
		cb.Data,
	)
	if err != nil {
		return fmt.Errorf("failed to create callback completed message: %w", err)
	}

	return tc.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), msg)
}

func parseCallbackKey(key string) (string, int64, error) {
	var nodeId int64
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

	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid callback key format: %s", key)
	}

	_, err := fmt.Sscanf(parts[1], "%d", &nodeId)
	if err != nil {
		return "", 0, fmt.Errorf("invalid node id in callback key: %w", err)
	}

	return parts[0], nodeId, nil
}
