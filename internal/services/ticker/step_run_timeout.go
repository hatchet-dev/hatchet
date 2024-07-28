package ticker

// import (
// 	"context"
// 	"time"

// 	"github.com/hatchet-dev/hatchet/internal/datautils"
// 	"github.com/hatchet-dev/hatchet/internal/msgqueue"
// 	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
// )

// func (t *TickerImpl) runPollStepRuns(ctx context.Context) func() {
// 	return func() {
// 		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
// 		defer cancel()

// 		t.l.Debug().Msgf("ticker: polling step runs")

// 		stepRuns, err := t.repo.Ticker().PollStepRuns(ctx, t.tickerId)

// 		if err != nil {
// 			t.l.Err(err).Msg("could not poll step runs")
// 			return
// 		}

// 		for _, stepRun := range stepRuns {
// 			tenantId := sqlchelpers.UUIDToStr(stepRun.TenantId)
// 			stepRunId := sqlchelpers.UUIDToStr(stepRun.ID)

// 			err := t.mq.AddMessage(
// 				ctx,
// 				msgqueue.JOB_PROCESSING_QUEUE,
// 				taskStepRunTimedOut(
// 					tenantId,
// 					stepRunId,
// 					sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
// 					&stepRun.Retries,
// 					&stepRun.RetryCount,
// 				),
// 			)

// 			if err != nil {
// 				t.l.Err(err).Msg("could not add step run timeout task")
// 			}
// 		}
// 	}
// }

// func taskStepRunTimedOut(tenantId, stepRunId string, workflowRunId string, retries *int32, retryCount *int32) *msgqueue.Message {
// 	payload, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskPayload{
// 		WorkflowRunId: workflowRunId,
// 		StepRunId:     stepRunId,
// 		StepRetries:   retries,
// 		RetryCount:    retryCount,
// 	})

// 	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunTimedOutTaskMetadata{
// 		TenantId: tenantId,
// 	})

// 	return &msgqueue.Message{
// 		ID:       "step-run-timed-out",
// 		Payload:  payload,
// 		Metadata: metadata,
// 		Retries:  3,
// 	}
// }
