package worker

import (
	"context"
	"fmt"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/internal/whrequest"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

type RegisterWebhookWorkerOpts struct {
	Name   string
	URL    string
	Secret *string
}

type ActionPayload struct {
	*client.Action

	ActionPayload string `json:"actionPayload"`
}

func (w *Worker) RegisterWebhook(ww RegisterWebhookWorkerOpts) error {
	tenantId := openapi_types.UUID{}
	if err := tenantId.Scan(w.client.TenantId()); err != nil {
		return fmt.Errorf("error getting tenant id: %w", err)
	}

	res, err := w.client.API().WebhookCreate(context.Background(), tenantId, rest.WebhookCreateJSONRequestBody{
		Url:    ww.URL,
		Secret: ww.Secret,
		Name:   ww.Name,
	})
	if err != nil {
		return fmt.Errorf("error creating webhook worker: %w", err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("error creating webhook, failed with status code %d", res.StatusCode)
	}

	return nil
}

type WebhookWorkerOpts struct {
	URL       string
	Secret    string
	WebhookId string
}

// TODO do not expose this to the end-user client somehow
func (w *Worker) StartWebhook(ww WebhookWorkerOpts) (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	listener, _, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    w.initActionNames,
		MaxRuns:    w.maxRuns,
		WebhookId:  &ww.WebhookId,
	})

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not get action listener: %w", err)
	}

	actionCh, errCh, err := listener.Actions(ctx)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not get action channel: %w", err)
	}

	go func() {
		for {
			select {
			case err := <-errCh:
				// NOTE: this matches the behavior of the old worker, until we change the signature of the webhook workers
				panic(err)
			case action := <-actionCh:
				go func(action *client.Action) {
					err := w.sendWebhook(context.Background(), action, ww)

					if err != nil {
						w.l.Error().Err(err).Msgf("could not execute action: %s", action.ActionId)
					}

					w.l.Debug().Msgf("action %s completed", action.ActionId)
				}(action)
			case <-ctx.Done():
				w.l.Debug().Msgf("worker %s received context done, stopping", w.name)
				return
			}
		}
	}()

	cleanup := func() error {
		cancel()

		w.l.Debug().Msgf("worker %s stopped", w.name)

		return nil
	}

	return cleanup, nil
}

func (w *Worker) sendWebhook(ctx context.Context, action *client.Action, ww WebhookWorkerOpts) error {
	w.l.Debug().Msgf("action received from step run %s, sending webhook at %s", action.StepRunId, time.Now())

	actionWithPayload := ActionPayload{
		Action:        action,
		ActionPayload: string(action.ActionPayload),
	}

	_, statusCode, err := whrequest.Send(ctx, ww.URL, ww.Secret, actionWithPayload)

	if statusCode != nil && *statusCode != 200 {
		w.l.Debug().Msgf("step run %s webhook sent with status code %d", action.StepRunId, *statusCode)
	}

	if err != nil {
		w.l.Warn().Msgf("step run %s could not send webhook to %s: %s", action.StepRunId, ww.URL, err)
		if err := w.markFailed(action, fmt.Errorf("could not send webhook: %w", err)); err != nil {
			return fmt.Errorf("could not send webhook and then could not send failed action event: %w", err)
		}
		return err
	}

	return nil
}

func (w *Worker) markFailed(action *client.Action, err error) error {
	failureEvent := w.getActionEvent(action, client.ActionEventTypeFailed)

	w.alerter.SendAlert(context.Background(), err, map[string]interface{}{
		"actionId":      action.ActionId,
		"workerId":      action.WorkerId,
		"workflowRunId": action.WorkflowRunId,
		"jobName":       action.JobName,
		"actionType":    action.ActionType,
	})

	failureEvent.EventPayload = err.Error()

	_, err = w.client.Dispatcher().SendStepActionEvent(
		context.Background(),
		failureEvent,
	)

	if err != nil {
		return fmt.Errorf("could not send action event: %w", err)
	}

	return nil
}
