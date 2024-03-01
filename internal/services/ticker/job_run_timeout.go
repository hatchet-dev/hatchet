package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *TickerImpl) handleScheduleJobRunTimeout(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: scheduling job run timeout")

	payload := tasktypes.ScheduleJobRunTimeoutTaskPayload{}
	metadata := tasktypes.ScheduleJobRunTimeoutTaskMetadata{}

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

	childCtx, cancel := context.WithDeadline(context.Background(), timeoutAt)

	go func() {
		select {
		case <-childCtx.Done():
			t.runJobRunTimeout(metadata.TenantId, payload.JobRunId)
		case <-ctx.Done():
		}
	}()

	// store the schedule in the step run map
	t.jobRuns.Store(payload.JobRunId, &timeoutCtx{
		ctx:    childCtx,
		cancel: cancel,
	})

	return nil
}

func (t *TickerImpl) handleCancelJobRunTimeout(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: canceling job run timeout")

	payload := tasktypes.CancelJobRunTimeoutTaskPayload{}
	metadata := tasktypes.CancelJobRunTimeoutTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	// get the cancel function
	childTimeoutCtxVal, ok := t.jobRuns.Load(payload.JobRunId)

	if !ok {
		return fmt.Errorf("could not find job run %s", payload.JobRunId)
	}

	// cancel the timeout
	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	childTimeoutCtx.ctx = context.WithValue(childTimeoutCtx.ctx, "cancelled", true)

	childTimeoutCtx.cancel()

	return nil
}

func (t *TickerImpl) runJobRunTimeout(tenantId, jobRunId string) {
	defer t.jobRuns.Delete(jobRunId)

	childTimeoutCtxVal, ok := t.jobRuns.Load(jobRunId)

	if !ok {
		t.l.Debug().Msgf("ticker: could not find job run %s", jobRunId)
		return
	}

	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

	var isCancelled bool

	if cancelledVal := childTimeoutCtx.ctx.Value("cancelled"); cancelledVal != nil {
		isCancelled = cancelledVal.(bool)
	}

	if isCancelled {
		t.l.Debug().Msgf("ticker: timeout of job run %s was cancelled", jobRunId)
		return
	}

	t.l.Debug().Msgf("ticker: job run %s timed out", jobRunId)

	// signal the jobs controller that the job timed out
	err := t.tq.AddTask(
		context.Background(),
		taskqueue.JOB_PROCESSING_QUEUE,
		taskJobRunTimedOut(tenantId, jobRunId),
	)

	if err != nil {
		t.l.Err(err).Msg("could not add job run requeue task")
	}
}

func taskJobRunTimedOut(tenantId, jobRunId string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.JobRunTimedOutTaskPayload{
		JobRunId: jobRunId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.JobRunTimedOutTaskMetadata{
		TenantId: tenantId,
	})

	return &taskqueue.Task{
		ID:       "job-run-timed-out",
		Payload:  payload,
		Metadata: metadata,
	}
}
