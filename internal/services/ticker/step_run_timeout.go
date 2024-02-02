package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *TickerImpl) handleScheduleStepRunTimeout(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: scheduling step run timeout")

	payload := tasktypes.ScheduleStepRunTimeoutTaskPayload{}
	metadata := tasktypes.ScheduleStepRunTimeoutTaskMetadata{}

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
		t.runStepRunTimeout(metadata.TenantId, payload.JobRunId, payload.StepRunId)
	}()

	// store the schedule in the step run map
	t.stepRuns.Store(payload.StepRunId, &timeoutCtx{
		ctx:    childCtx,
		cancel: cancel,
	})

	return nil
}

func (t *TickerImpl) handleCancelStepRunTimeout(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: canceling step run timeout")

	payload := tasktypes.CancelStepRunTimeoutTaskPayload{}
	metadata := tasktypes.CancelStepRunTimeoutTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	// get the cancel function
	childTimeoutCtxVal, ok := t.stepRuns.Load(payload.StepRunId)

	if !ok {
		return nil
	}

	// cancel the timeout
	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	childTimeoutCtx.ctx = context.WithValue(childTimeoutCtx.ctx, "cancelled", true)

	childTimeoutCtx.cancel()

	return nil
}

func (t *TickerImpl) runStepRunTimeout(tenantId, jobRunId, stepRunId string) {
	defer t.stepRuns.Delete(stepRunId)

	childTimeoutCtxVal, ok := t.stepRuns.Load(stepRunId)

	if !ok {
		t.l.Debug().Msgf("ticker: could not find step run %s", stepRunId)
		return
	}

	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	var isCancelled bool

	if cancelledVal := childTimeoutCtx.ctx.Value("cancelled"); cancelledVal != nil {
		isCancelled = cancelledVal.(bool)
	}

	if isCancelled {
		t.l.Debug().Msgf("ticker: timeout of %s was cancelled", stepRunId)
		return
	}

	t.l.Debug().Msgf("ticker: step run %s timed out", stepRunId)

	// signal the jobs controller that the step timed out
	err := t.tq.AddTask(
		context.Background(),
		taskqueue.JOB_PROCESSING_QUEUE,
		taskStepRunTimedOut(tenantId, jobRunId, stepRunId),
	)

	if err != nil {
		t.l.Err(err).Msg("could not add step run requeue task")
	}
}

func taskStepRunTimedOut(tenantId, jobRunId, stepRunId string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskPayload{
		StepRunId: stepRunId,
		JobRunId:  jobRunId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskMetadata{
		TenantId: tenantId,
	})

	return &taskqueue.Task{
		ID:       "step-run-timed-out",
		Payload:  payload,
		Metadata: metadata,
	}
}
