package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/codes"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

var (
	errFlowControlActive     = errors.New("could not acquire worker send mutex, flow control is active")
	errSessionReleased       = errors.New("worker session has been released")
	errWorkerPaused          = errors.New("worker session is paused, the task is returned to the queue")
	errWorkerSendInterrupted = errors.New("caller stopped waiting for an in-flight worker send; delivery is uncertain")
)

func (worker *subscribedWorker) StartTaskFromBulk(
	ctx context.Context,
	tenantId uuid.UUID,
	task *v1.V1TaskWithPayload,
	durableInvocationCount *int32,
) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context done before starting task: %w", ctx.Err())
	}

	ctx, span := telemetry.NewSpan(ctx, "start-step-run-from-bulk") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if task.Payload != nil {
		inputBytes = task.Payload
	}

	action := populateAssignedAction(tenantId, task.V1Task, task.Runtime, task.RetryCount, durableInvocationCount)
	worker.setOperatorTaskInsertedAt(action, task.V1Task)

	action.ActionType = contracts.ActionType_START_STEP_RUN
	action.ActionPayload = string(inputBytes)

	err := worker.sendToWorker(ctx, action)

	if err != nil {
		// An in-flight send stays assigned: the worker may already be running it, and a
		// later Send error is dropped. A deadline before admission is returned so the
		// caller requeues. An operator handler has no stream, so its deadline stays
		// assigned too.
		if errors.Is(err, errWorkerSendInterrupted) || (worker.handler != nil && errors.Is(err, context.DeadlineExceeded)) {
			return nil
		}

		return fmt.Errorf("could not send start action to worker: %w", err)
	}

	return nil
}

func (worker *subscribedWorker) StartBatch(ctx context.Context, action *contracts.AssignedAction) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context done before starting batch: %w", ctx.Err())
	}

	err := worker.sendToWorker(ctx, action)
	// An in-flight send stays assigned so a retry cannot run the batch twice; see
	// StartTaskFromBulk.
	if errors.Is(err, errWorkerSendInterrupted) {
		return nil
	}
	return err
}

func (worker *subscribedWorker) sendToWorker(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// a paused operator session refuses starts the way a failed send does, so the caller
	// requeues the task; see subscribedWorker.paused
	if action.ActionType != contracts.ActionType_CANCEL_STEP_RUN && worker.paused.Load() {
		return errWorkerPaused
	}

	if worker.handler != nil {
		return worker.sendToWorkerWithOperator(ctx, action)
	}

	return worker.sendToWorkerWithStream(ctx, action)
}

func (worker *subscribedWorker) sendToWorkerWithOperator(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
	ctx, span := telemetry.NewSpan(ctx, "send-to-worker-operator") // nolint:ineffassign
	defer span.End()

	telemetry.WithAttributes(
		span,
		telemetry.AttributeKV{
			Key:   "worker.id",
			Value: worker.workerId,
		},
	)

	return worker.handler.HandleAction(ctx, action)
}

func (worker *subscribedWorker) sendToWorkerWithStream(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
	ctx, span := telemetry.NewSpan(ctx, "send-to-worker") // nolint:ineffassign
	defer span.End()

	telemetry.WithAttributes(
		span,
		telemetry.AttributeKV{
			Key:   "worker.id",
			Value: worker.workerId,
		},
	)

	telemetry.WithAttributes(
		span,
		telemetry.AttributeKV{
			Key:   "payload.size_bytes",
			Value: len(action.ActionPayload),
		},
	)

	select {
	case <-worker.done:
		return errSessionReleased
	default:
	}

	lockBegin := time.Now()
	_, lockSpan := telemetry.NewSpan(ctx, "acquire-worker-stream-lock")
	lockErr := worker.sendLock.AcquireContext(ctx)
	lockSpan.End()
	telemetry.WithAttributes(span, telemetry.AttributeKV{
		Key: "lock.duration_ms", Value: time.Since(lockBegin).Milliseconds(),
	})
	if errors.Is(lockErr, timeout_lock.ErrTimeout) {
		span.RecordError(errFlowControlActive)
		span.SetStatus(codes.Error, "flow control is active")
		return errFlowControlActive
	}
	if lockErr != nil {
		return lockErr
	}

	_, holdSpan := telemetry.NewSpan(ctx, "hold-worker-stream-lock")
	queuedAt := time.Now()
	// Cancellation and the sender atomically decide whether an action can still be
	// withdrawn. Once sending starts, retrying it could execute the task twice.
	const (
		pending uint32 = iota
		sending
		withdrawn
	)
	var state atomic.Uint32
	sentCh := make(chan error, 1)

	go func() {
		// The transport may outlive the caller. Retain admission until it actually
		// returns so later actions cannot accumulate behind Sender's unbounded mutex.
		defer worker.sendLock.Release()
		defer holdSpan.End()
		if ctx.Err() != nil {
			state.CompareAndSwap(pending, withdrawn)
		}
		if !state.CompareAndSwap(pending, sending) {
			sentCh <- ctx.Err()
			return
		}
		select {
		case <-worker.done:
			sentCh <- errSessionReleased
			return
		default:
		}
		_, streamSpan := telemetry.NewSpan(ctx, "send-worker-stream")
		defer streamSpan.End()
		telemetry.WithAttributes(streamSpan, telemetry.AttributeKV{Key: "schedule.duration_ms", Value: time.Since(queuedAt).Milliseconds()})
		sendBegin := time.Now()
		err := worker.stream.Send(action)
		duration := time.Since(sendBegin)
		telemetry.WithAttributes(streamSpan, telemetry.AttributeKV{Key: "send.duration_ms", Value: duration.Milliseconds()})
		if err != nil {
			streamSpan.RecordError(err)
			streamSpan.SetStatus(codes.Error, "worker send failed")
		} else if duration > 50*time.Millisecond {
			streamSpan.AddEvent("slow worker send")
		}
		sentCh <- err
	}()

	select {
	case <-ctx.Done():
		// Prefer a completed send when cancellation races with its result.
		select {
		case err := <-sentCh:
			return err
		default:
		}
		if state.CompareAndSwap(pending, withdrawn) || state.Load() == withdrawn {
			return ctx.Err()
		}
		span.AddEvent("worker send delivery uncertain")
		return fmt.Errorf("%w: %w", errWorkerSendInterrupted, ctx.Err())
	case err := <-sentCh:
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "worker send failed")
		}
		return err
	}
}

func (worker *subscribedWorker) CancelTask(
	ctx context.Context,
	tenantId uuid.UUID,
	task *sqlcv1.V1Task,
	retryCount int32,
	durableTaskInvocationCount *int32,
) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context done before cancelling task: %w", ctx.Err())
	}

	ctx, span := telemetry.NewSpan(ctx, "cancel-task") // nolint:ineffassign
	defer span.End()

	action := populateAssignedAction(tenantId, task, nil, retryCount, durableTaskInvocationCount)
	worker.setOperatorTaskInsertedAt(action, task)

	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	err := worker.sendToWorker(ctx, action)

	if err != nil {
		// An in-flight cancel may already have been delivered, so it stays assigned; see
		// StartTaskFromBulk.
		if errors.Is(err, errWorkerSendInterrupted) || (worker.handler != nil && errors.Is(err, context.DeadlineExceeded)) {
			return nil
		}

		if errors.Is(err, errFlowControlActive) {
			msg, merr := tasktypesv1.MonitoringEventMessageFromInternal(
				task.TenantID,
				tasktypesv1.CreateMonitoringEventPayload{
					TaskId:         task.ID,
					RetryCount:     task.RetryCount,
					WorkerId:       &worker.workerId,
					EventType:      sqlcv1.V1EventTypeOlapCOULDNOTSENDTOWORKER,
					EventTimestamp: time.Now().UTC(),
					EventMessage:   fmt.Sprintf("Could not acquire send lock before timeout of %s", worker.sendLock.Timeout),
				},
			)
			if merr != nil {
				return fmt.Errorf("could not create monitoring event for task %d: %w", task.ID, merr)
			}

			return worker.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)
		}

		return fmt.Errorf("could not send cancel action to worker: %w", err)
	}

	return nil
}

func populateAssignedAction(tenantID uuid.UUID, task *sqlcv1.V1Task, runtime *sqlcv1.V1TaskRuntime, retryCount int32, invocationCount *int32) *contracts.AssignedAction {
	workflowId := task.WorkflowID.String()
	workflowVersionId := task.WorkflowVersionID.String()

	action := &contracts.AssignedAction{
		TenantId:                   tenantID.String(),
		JobId:                      task.StepID.String(), // FIXME
		JobName:                    task.StepReadableID,
		JobRunId:                   task.ExternalID.String(), // FIXME
		TaskId:                     task.StepID.String(),
		TaskRunExternalId:          task.ExternalID.String(),
		ActionId:                   task.ActionID,
		TaskName:                   task.StepReadableID,
		WorkflowRunId:              task.WorkflowRunID.String(),
		RetryCount:                 retryCount,
		Priority:                   task.Priority.Int32,
		WorkflowId:                 &workflowId,
		WorkflowVersionId:          &workflowVersionId,
		DurableTaskInvocationCount: invocationCount,
	}

	if task.AdditionalMetadata != nil {
		metadataStr := string(task.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if task.ParentTaskExternalID != nil {
		parentId := task.ParentTaskExternalID.String()
		action.ParentWorkflowRunId = &parentId
	}

	if task.ChildIndex.Valid {
		i := int32(task.ChildIndex.Int64) // nolint: gosec
		action.ChildWorkflowIndex = &i
	}

	if task.ChildKey.Valid {
		key := task.ChildKey.String
		action.ChildWorkflowKey = &key
	}

	if runtime != nil {
		if runtime.BatchID != nil {
			batchID := runtime.BatchID.String()
			action.BatchId = &batchID
		}

		if runtime.BatchSize.Valid {
			size := runtime.BatchSize.Int32
			action.BatchSize = &size
		}

		if runtime.BatchIndex.Valid {
			index := runtime.BatchIndex.Int32
			action.BatchIndex = &index
		}

		if runtime.BatchKey.Valid {
			key := strings.TrimSpace(runtime.BatchKey.String)
			if key != "" {
				action.BatchKey = &key
			}
		}
	}

	if action.BatchKey == nil && task.BatchKey.Valid {
		key := strings.TrimSpace(task.BatchKey.String)
		if key != "" {
			action.BatchKey = &key
		}
	}

	if task.TriggeringEventExternalID != nil {
		triggeringEventExternalId := task.TriggeringEventExternalID.String()
		action.TriggeringEventExternalId = &triggeringEventExternalId
	}

	if task.TriggeringEventKey.Valid {
		triggeringEventKey := task.TriggeringEventKey.String
		action.TriggeringEventKey = &triggeringEventKey
	}

	return action
}

func (worker *subscribedWorker) setOperatorTaskInsertedAt(action *contracts.AssignedAction, task *sqlcv1.V1Task) {
	if worker.handler == nil || task == nil || !task.InsertedAt.Valid {
		return
	}

	action.TaskInsertedAt = timestamppb.New(task.InsertedAt.Time)
}
