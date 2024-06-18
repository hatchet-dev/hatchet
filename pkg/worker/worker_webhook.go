package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

type RegisterWebhookWorkerOpts struct {
	URL    string
	Secret *string
}

func (w *Worker) RegisterWebhook(ww RegisterWebhookWorkerOpts) error {
	tenantId := openapi_types.UUID{}
	if err := tenantId.Scan(w.client.TenantId()); err != nil {
		return fmt.Errorf("error getting tenant id: %w", err)
	}

	res, err := w.client.API().WebhookCreate(context.Background(), tenantId, rest.WebhookCreateJSONRequestBody{
		Url:       ww.URL,
		Workflows: &w.workflows,
		Secret:    ww.Secret,
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
	URL    string
	Secret string
}

// TODO do not expose this to the end-user client somehow
func (w *Worker) StartWebhook(ww WebhookWorkerOpts) (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	listener, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    w.initActionNames,
		MaxRuns:    w.maxRuns,
	})

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not get action listener: %w", err)
	}

	actionCh, err := listener.Actions(ctx)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not get action channel: %w", err)
	}

	go func() {
		for {
			select {
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

		w.l.Debug().Msgf("worker %s is stopping...", w.name)

		err := listener.Unregister()
		if err != nil {
			return fmt.Errorf("could not unregister worker: %w", err)
		}

		w.l.Debug().Msgf("worker %s stopped", w.name)

		return nil
	}

	return cleanup, nil
}

func (w *Worker) sendWebhook(ctx context.Context, action *client.Action, ww WebhookWorkerOpts) error {
	w.l.Debug().Msgf("action received, sending webhook at %s", time.Now())

	body, err := json.Marshal(action)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ww.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	sig, err := signature.Sign(string(body), ww.Secret)
	if err != nil {
		return err
	}
	req.Header.Set("X-Hatchet-Signature", sig)

	w.l.Debug().Msgf("sending webhook to: %s", ww.URL)

	httpClient := &http.Client{
		// use 10 minutes timeout
		Timeout: time.Second * 600,
	}

	// nolint:gosec
	resp, err := httpClient.Do(req)
	if err != nil {
		w.l.Warn().Msgf("could not send webhook to %s: %s", ww.URL, err)
		if err := w.markFailed(action, fmt.Errorf("could not send webhook: %w", err)); err != nil {
			return fmt.Errorf("could not send webhook and then could not send failed action event: %w", err)
		}
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.l.Warn().Msgf("could not send webhook to %s: code %d", ww.URL, resp.StatusCode)
		if err := w.markFailed(action, fmt.Errorf("webhook failed with status code %d", resp.StatusCode)); err != nil {
			return fmt.Errorf("webhook failed with status code %d and then could not send failed action event: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("webhook failed with status code %d", resp.StatusCode)
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
		context.TODO(),
		failureEvent,
	)

	if err != nil {
		return fmt.Errorf("could not send action event: %w", err)
	}

	return nil
}
