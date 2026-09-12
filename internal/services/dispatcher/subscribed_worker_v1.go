package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

var (
	errFlowControlActive = errors.New("could not acquire worker send mutex, flow control is active")
	errSessionReleased   = errors.New("worker session has been released")
	errWorkerPaused      = errors.New("worker session is paused, the task is returned to the queue")
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

	action.ActionType = contracts.ActionType_START_STEP_RUN
	action.ActionPayload = string(inputBytes)

	err := worker.sendToWorker(ctx, action)

	if err != nil {
		// if the context is done, we return nil, because the worker took too long to receive the message, and we're not
		// sure if the worker received it or not. this is equivalent to a network drop, and would be resolved by worker-side
		// acks, which we don't currently have.
		if errors.Is(err, context.DeadlineExceeded) {
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

	return worker.sendToWorker(ctx, action)
}

func (worker *subscribedWorker) sendToWorker(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
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

	var msg proto.Message = action

	if worker.wrap != nil {
		msg = worker.wrap(action)
	}

	return worker.sendMsg(ctx, msg)
}

// sendMsg encodes msg and writes it on the stream. Writes on one stream are serialised by
// sendLock, which is held until the SendMsg call itself exits: gRPC forbids concurrent SendMsg
// calls on a stream, so a caller whose ctx ends while its send is still blocked by flow
// control returns without releasing the lock, and the next caller fails fast with
// errFlowControlActive once the lock timeout elapses rather than starting an overlapping
// send.
func (worker *subscribedWorker) sendMsg(ctx context.Context, msg proto.Message) error {
	select {
	case <-worker.done:
		return errSessionReleased
	default:
	}

	_, span := telemetry.NewSpan(ctx, "send-worker-message")
	defer span.End()

	_, encodeSpan := telemetry.NewSpan(ctx, "encode-action")

	prepared := &grpc.PreparedMsg{}
	err := prepared.Encode(worker.stream, msg)
	if err != nil {
		encodeSpan.RecordError(err)
		encodeSpan.End()
		return fmt.Errorf("could not encode action: %w", err)
	}

	encodeSpan.End()

	lockBegin := time.Now()

	_, lockSpan := telemetry.NewSpan(ctx, "acquire-worker-stream-lock")

	if !worker.sendLock.Acquire() {
		lockSpan.End()
		span.RecordError(errFlowControlActive)
		span.SetStatus(codes.Error, "flow control is active")
		return errFlowControlActive
	}

	lockSpan.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{
		Key:   "lock.duration_ms",
		Value: time.Since(lockBegin).Milliseconds(),
	})

	_, streamSpan := telemetry.NewSpan(ctx, "send-worker-stream")
	defer streamSpan.End()

	sendMsgBegin := time.Now()

	sentCh := make(chan error, 1)

	go func() {
		// the lock is released only once SendMsg has returned, whether or not the caller is
		// still waiting for the result
		defer worker.sendLock.Release()

		err := worker.stream.SendMsg(prepared)

		if err != nil {
			span.RecordError(err)
		}

		if time.Since(sendMsgBegin) > 50*time.Millisecond {
			span.SetStatus(codes.Error, "flow control detected")
			span.RecordError(fmt.Errorf("send took too long, we may be in flow control: %s", time.Since(sendMsgBegin)))
		}

		sentCh <- err
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done before send could complete: %w", ctx.Err())
	case err = <-sentCh:
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

	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	err := worker.sendToWorker(ctx, action)

	if err != nil {
		// if the context is done, we return nil, because the worker took too long to receive the message, and we're not
		// sure if the worker received it or not. this is equivalent to a network drop, and would be resolved by worker-side
		// acks, which we don't currently have.
		if errors.Is(err, context.DeadlineExceeded) {
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
