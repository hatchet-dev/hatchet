package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

func (w *Worker) StartWebhook() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// TODO get this from the database
	actionNames := []string{
		"default:step-one",
		"default:step-two",
	}

	log.Printf("starting webhook worker for actions: %s", actionNames)

	listener, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    actionNames,
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
					err := w.sendWebhook(context.Background(), action)

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

func (w *Worker) sendWebhook(ctx context.Context, action *client.Action) error {
	// TODO
	webhookUrl := "http://localhost:8741/webhook"
	webhookSecret := "secret"

	body, err := json.Marshal(action)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}

	log.Printf("using secret: %s", webhookSecret)

	sig, err := signature.Sign(string(body), webhookSecret)
	if err != nil {
		return err
	}
	req.Header.Set("X-Hatchet-Signature", sig)

	log.Printf("sending webhook to: %s", webhookUrl)

	// nolint:gosec
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO!!! handle error
		return fmt.Errorf("webhook failed with status code %d", resp.StatusCode)
	}

	return nil
}
