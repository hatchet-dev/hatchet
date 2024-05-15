package worker

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

func (w *Worker) WebhookHandler(process func(ctx HatchetContext) interface{}) http.HandlerFunc {
	return func(writer http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		log.Printf("got webhook request!")

		var event dispatcher.WebhookEvent
		if err := json.Unmarshal(data, &event); err != nil {
			panic(err)
		}

		indent, _ := json.MarshalIndent(event, "", "  ")
		log.Printf("data: %s", string(indent))

		action := &client.Action{
			//WorkerId:         event.WorkerId,
			TenantId:         event.TenantId,
			WorkflowRunId:    event.WorkflowRunId,
			GetGroupKeyRunId: event.GetGroupKeyRunId,
			JobId:            event.JobId,
			JobName:          event.JobName,
			JobRunId:         event.JobRunId,
			StepId:           event.StepId,
			StepName:         event.StepName,
			StepRunId:        event.StepRunId,
			ActionId:         event.ActionId,
			//ActionPayload:    event.ActionPayload,
			//ActionType:       event.ActionType,
		}

		timestamp := time.Now().UTC()
		_, err = w.client.Dispatcher().SendStepActionEvent(context.TODO(),
			&client.ActionEvent{
				Action:         action,
				EventTimestamp: &timestamp,
				EventType:      client.ActionEventTypeStarted,
			},
		)
		if err != nil {
			panic(err)
		}

		ctx, err := newHatchetContext(context.TODO(), action, w.client, w.l)
		if err != nil {
			panic(err)
		}
		resp := process(ctx)

		log.Printf("got response from user: %+v", resp)

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))

		timestamp = time.Now().UTC()
		_, err = w.client.Dispatcher().SendStepActionEvent(context.TODO(),
			&client.ActionEvent{
				Action:         action,
				EventTimestamp: &timestamp,
				EventType:      client.ActionEventTypeCompleted,
				EventPayload:   resp,
			},
		)
		if err != nil {
			panic(err)
		}
	}
}
