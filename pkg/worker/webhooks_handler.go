package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

func (w *Worker) WebhookHandler(secret string) http.HandlerFunc {
	return func(writer http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		log.Printf("got webhook request!")

		expected := r.Header.Get("X-Hatchet-Signature")
		actual, err := signature.Sign(string(data), secret)
		if err != nil {
			panic(fmt.Errorf("could not sign data: %w", err))
		}

		log.Printf("actual: %s", actual)
		log.Printf("expected: %s", expected)

		if expected != actual {
			panic(fmt.Errorf("invalid webhook signature"))
		}

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
			ActionPayload:    []byte(event.ActionPayload),
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
		resp, err := w.webhookProcess(ctx)
		if err != nil {
			// TODO handle error gracefully and send a failed event
			panic(err)
		}

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

func (w *Worker) webhookProcess(ctx HatchetContext) (interface{}, error) {
	log.Printf("processing webhook")

	var do Action
	for _, action := range w.actions {
		split := strings.Split(action.Name(), ":") // service:action
		log.Printf("action: %s", split[1])
		if split[1] == ctx.StepName() {
			do = action
			break
		}
	}

	res := do.Run(ctx)

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid response from action, expected 2 values")
	}

	if res[1] != nil {
		err, ok := res[1].(error)
		if !ok {
			return nil, fmt.Errorf("invalid response from action, expected error")
		}

		if err != nil {
			return nil, err
		}
	}

	return res[0], nil
}
