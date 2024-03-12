package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *TickerImpl) handleScheduleGetGroupKeyRunTimeout(ctx context.Context, task *msgqueue.Message) error {
	t.l.Debug().Msg("ticker: scheduling get group key run timeout")

	payload := tasktypes.ScheduleGetGroupKeyRunTimeoutTaskPayload{}
	metadata := tasktypes.ScheduleGetGroupKeyRunTimeoutTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	timeoutAt, err := time.Parse(time.RFC3339, payload.TimeoutAt)

	if err != nil {
		return fmt.Errorf("could not parse timeout at: %w", err)
	}

	// schedule the timeout
	childCtx, cancel := context.WithDeadline(context.Background(), timeoutAt)

	go func() {
		<-childCtx.Done()
		t.runGetGroupKeyRunTimeout(metadata.TenantId, payload.WorkflowRunId, payload.GetGroupKeyRunId)
	}()

	// store the schedule in the get group key run map
	t.getGroupKeyRuns.Store(payload.GetGroupKeyRunId, &timeoutCtx{
		ctx:    childCtx,
		cancel: cancel,
	})

	return nil
}

func (t *TickerImpl) handleCancelGetGroupKeyRunTimeout(ctx context.Context, task *msgqueue.Message) error {
	t.l.Debug().Msg("ticker: canceling get group key run timeout")

	payload := tasktypes.CancelGetGroupKeyRunTimeoutTaskPayload{}
	metadata := tasktypes.CancelGetGroupKeyRunTimeoutTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	// get the cancel function
	childTimeoutCtxVal, ok := t.getGroupKeyRuns.Load(payload.GetGroupKeyRunId)

	if !ok {
		return nil
	}

	// cancel the timeout
	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	childTimeoutCtx.ctx = context.WithValue(childTimeoutCtx.ctx, "cancelled", true)

	childTimeoutCtx.cancel()

	return nil
}

func (t *TickerImpl) runGetGroupKeyRunTimeout(tenantId, workflowRunId, getGroupKeyRunId string) {
	defer t.getGroupKeyRuns.Delete(getGroupKeyRunId)

	childTimeoutCtxVal, ok := t.getGroupKeyRuns.Load(getGroupKeyRunId)

	if !ok {
		t.l.Debug().Msgf("ticker: could not find get group key run %s", getGroupKeyRunId)
		return
	}

	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	var isCancelled bool

	if cancelledVal := childTimeoutCtx.ctx.Value("cancelled"); cancelledVal != nil {
		isCancelled = cancelledVal.(bool)
	}

	if isCancelled {
		t.l.Debug().Msgf("ticker: timeout of %s was cancelled", getGroupKeyRunId)
		return
	}

	t.l.Debug().Msgf("ticker: get group key run %s timed out", getGroupKeyRunId)

	// signal the jobs controller that the group key run timed out
	err := t.mq.AddMessage(
		context.Background(),
		msgqueue.JOB_PROCESSING_QUEUE,
		taskGetGroupKeyRunTimedOut(tenantId, workflowRunId, getGroupKeyRunId),
	)

	if err != nil {
		t.l.Err(err).Msg("could not add get group key run requeue task")
	}
}

func taskGetGroupKeyRunTimedOut(tenantId, workflowRunId, getGroupKeyRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunTimedOutTaskPayload{
		GetGroupKeyRunId: getGroupKeyRunId,
		WorkflowRunId:    workflowRunId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunTimedOutTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "get-group-key-run-timed-out",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
