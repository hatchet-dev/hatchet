package ticker

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/hatchet-dev/hatchet/internal/datautils"
// 	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
// 	"github.com/hatchet-dev/hatchet/internal/taskqueue"
// )

// func (t *TickerImpl) handleScheduleStepRunRequeue(ctx context.Context, task *taskqueue.Task) error {
// 	t.l.Debug().Msg("ticker: scheduling step run timeout")

// 	payload := tasktypes.ScheduleStepRunRequeueTaskPayload{}
// 	metadata := tasktypes.ScheduleStepRunRequeueTaskMetadata{}

// 	err := t.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode ticker task payload: %w", err)
// 	}

// 	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode ticker task metadata: %w", err)
// 	}

// 	requeueAfter, err := time.Parse(time.RFC3339, payload.RequeueAfter)

// 	if err != nil {
// 		return fmt.Errorf("could not parse timeout at: %w", err)
// 	}

// 	// schedule the timeout
// 	childCtx, cancel := context.WithDeadline(context.Background(), requeueAfter)

// 	go func() {
// 		<-childCtx.Done()
// 		t.runStepRunRequeue(metadata.TenantId, payload.JobRunId, payload.StepRunId)
// 	}()

// 	// store the schedule in the step run map
// 	t.stepRunRequeues.Store(payload.StepRunId, &timeoutCtx{
// 		ctx:    childCtx,
// 		cancel: cancel,
// 	})

// 	return nil
// }

// func (t *TickerImpl) runStepRunRequeue(tenantId, jobRunId, stepRunId string) {
// 	defer t.stepRunRequeues.Delete(stepRunId)

// 	childTimeoutCtxVal, ok := t.stepRunRequeues.Load(stepRunId)

// 	if !ok {
// 		t.l.Debug().Msgf("ticker: could not find step run %s", stepRunId)
// 		return
// 	}

// 	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

// 	var isCancelled bool

// 	if cancelledVal := childTimeoutCtx.ctx.Value("cancelled"); cancelledVal != nil {
// 		isCancelled = cancelledVal.(bool)
// 	}

// 	if isCancelled {
// 		t.l.Debug().Msgf("ticker: timeout of %s was cancelled", stepRunId)
// 		return
// 	}

// 	t.l.Debug().Msgf("ticker: step run %s timed out", stepRunId)

// 	// signal the jobs controller that the step timed out
// 	err := t.tq.AddTask(
// 		context.Background(),
// 		taskqueue.JOB_PROCESSING_QUEUE,
// 		taskStepRunRequeue(tenantId, jobRunId, stepRunId),
// 	)

// 	if err != nil {
// 		t.l.Err(err).Msg("could not add step run requeue task")
// 	}
// }

// func taskStepRunRequeue(tenantId, jobRunId, stepRunId string) *taskqueue.Task {
// 	payload, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskPayload{
// 		StepRunId: stepRunId,
// 		JobRunId:  jobRunId,
// 	})

// 	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskMetadata{
// 		TenantId: tenantId,
// 	})

// 	return &taskqueue.Task{
// 		ID:       "step-run-requeue",
// 		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
// 		Payload:  payload,
// 		Metadata: metadata,
// 	}
// }

// func (t *TickerImpl) handleCancelStepRunRequeue(ctx context.Context, task *taskqueue.Task) error {
// 	t.l.Debug().Msg("ticker: canceling job run timeout")

// 	payload := tasktypes.CancelStepRunRequeueTaskPayload{}
// 	metadata := tasktypes.CancelStepRunRequeueTaskMetadata{}

// 	err := t.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode ticker task payload: %w", err)
// 	}

// 	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode ticker task metadata: %w", err)
// 	}

// 	// get the cancel function
// 	childTimeoutCtxVal, ok := t.jobRuns.Load(payload.JobRunId)

// 	if !ok {
// 		return fmt.Errorf("could not find job run %s", payload.JobRunId)
// 	}

// 	// cancel the timeout
// 	childTimeoutCtx := childTimeoutCtxVal.(*timeoutCtx)

// 	childTimeoutCtx.ctx = context.WithValue(childTimeoutCtx.ctx, "cancelled", true)

// 	childTimeoutCtx.cancel()

// 	return nil
// }
