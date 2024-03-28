package ticker

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *TickerImpl) runPollGetGroupKeyRuns(ctx context.Context) func() {
	return func() {
		t.l.Debug().Msgf("ticker: polling get group key runs")

		getGroupKeyRuns, err := t.repo.Ticker().PollGetGroupKeyRuns(t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not poll get group key runs")
			return
		}

		for _, getGroupKeyRun := range getGroupKeyRuns {
			tenantId := sqlchelpers.UUIDToStr(getGroupKeyRun.TenantId)
			getGroupKeyRunId := sqlchelpers.UUIDToStr(getGroupKeyRun.ID)

			err := t.mq.AddMessage(
				ctx,
				msgqueue.JOB_PROCESSING_QUEUE,
				taskGetGroupKeyRunTimedOut(tenantId, getGroupKeyRunId),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add step run timeout task")
			}
		}
	}
}

func taskGetGroupKeyRunTimedOut(tenantId, getGroupKeyRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunTimedOutTaskPayload{
		GetGroupKeyRunId: getGroupKeyRunId,
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
