package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (worker *subscribedWorker) StartTaskFromBulk(
	ctx context.Context,
	tenantId string,
	task *v1.V1TaskWithPayload,
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

	action := populateAssignedAction(tenantId, task.V1Task, task.Runtime, task.RetryCount)

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

func (worker *subscribedWorker) incBacklogSize(delta int64) bool {
	worker.backlogSizeMu.Lock()
	defer worker.backlogSizeMu.Unlock()

	if worker.backlogSize+delta > worker.maxBacklogSize {
		return false
	}

	worker.backlogSize += delta

	return true
}

func (worker *subscribedWorker) decBacklogSize(delta int64) int64 {
	worker.backlogSizeMu.Lock()
	defer worker.backlogSizeMu.Unlock()

	worker.backlogSize -= delta

	if worker.backlogSize < 0 {
		worker.backlogSize = 0
	}

	return worker.backlogSize
}

func (worker *subscribedWorker) sendToWorker(
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

	_, encodeSpan := telemetry.NewSpan(ctx, "encode-action")

	msg := &grpc.PreparedMsg{}
	err := msg.Encode(worker.stream, action)

	if err != nil {
		encodeSpan.RecordError(err)
		encodeSpan.End()
		return fmt.Errorf("could not encode action: %w", err)
	}

	encodeSpan.End()

	incSuccess := worker.incBacklogSize(1)

	if !incSuccess {
		err := fmt.Errorf("worker backlog size exceeded max of %d", worker.maxBacklogSize)
		span.RecordError(err)
		span.SetStatus(codes.Error, "worker backlog size exceeded max")
		return err
	}

	lockBegin := time.Now()

	_, lockSpan := telemetry.NewSpan(ctx, "acquire-worker-stream-lock")

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

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
		defer close(sentCh)
		defer worker.decBacklogSize(1)

		err = worker.stream.SendMsg(msg)

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
	tenantId string,
	task *sqlcv1.V1Task,
	retryCount int32,
) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context done before cancelling task: %w", ctx.Err())
	}

	ctx, span := telemetry.NewSpan(ctx, "cancel-task") // nolint:ineffassign
	defer span.End()

	action := populateAssignedAction(tenantId, task, nil, retryCount)

	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	sentCh := make(chan error, 1)
	incSuccess := worker.incBacklogSize(1)

	if !incSuccess {
		err := fmt.Errorf("worker backlog size exceeded max of %d", worker.maxBacklogSize)
		span.RecordError(err)
		span.SetStatus(codes.Error, "worker backlog size exceeded max")
		return err
	}

	go func() {
		defer close(sentCh)
		defer worker.decBacklogSize(1)

		worker.sendMu.Lock()
		defer worker.sendMu.Unlock()

		sentCh <- worker.stream.Send(action)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done before send could complete: %w", ctx.Err())
	case err := <-sentCh:
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("could not send cancel action to worker: %w", err)
		}
	}

	return nil
}

func populateAssignedAction(tenantID string, task *sqlcv1.V1Task, runtime *sqlcv1.V1TaskRuntime, retryCount int32) *contracts.AssignedAction {
	workflowId := sqlchelpers.UUIDToStr(task.WorkflowID)
	workflowVersionId := sqlchelpers.UUIDToStr(task.WorkflowVersionID)

	action := &contracts.AssignedAction{
		TenantId:          tenantID,
		JobId:             sqlchelpers.UUIDToStr(task.StepID), // FIXME
		JobName:           task.StepReadableID,
		JobRunId:          sqlchelpers.UUIDToStr(task.ExternalID), // FIXME
		StepId:            sqlchelpers.UUIDToStr(task.StepID),
		StepRunId:         sqlchelpers.UUIDToStr(task.ExternalID),
		ActionId:          task.ActionID,
		StepName:          task.StepReadableID,
		WorkflowRunId:     sqlchelpers.UUIDToStr(task.WorkflowRunID),
		RetryCount:        retryCount,
		Priority:          task.Priority.Int32,
		WorkflowId:        &workflowId,
		WorkflowVersionId: &workflowVersionId,
	}

	if task.AdditionalMetadata != nil {
		metadataStr := string(task.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if task.ParentTaskExternalID.Valid {
		parentId := sqlchelpers.UUIDToStr(task.ParentTaskExternalID)
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
		if runtime.BatchID.Valid {
			batchID := sqlchelpers.UUIDToStr(runtime.BatchID)
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

	return action
}
