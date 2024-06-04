package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

func (w *Worker) StartWebhook(id string) (func() error, error) {
	// TODO
	prisma := db.NewClient()
	if err := prisma.Connect(); err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	defer func(prisma *db.PrismaClient) {
		_ = prisma.Disconnect()
	}(prisma)

	ww, err := prisma.WebhookWorker.FindUnique(db.WebhookWorker.ID.Equals(id)).With(
		db.WebhookWorker.WebhookWorkerWorkflows.Fetch().With(
			db.WebhookWorkerWorkflow.Workflow.Fetch().With(
				db.Workflow.Versions.Fetch().OrderBy(
					db.WorkflowVersion.Order.Order(db.SortOrderDesc),
				).Take(1).With(
					db.WorkflowVersion.Jobs.Fetch().With(
						db.Job.Steps.Fetch().With(
							db.Step.Action.Fetch(),
						),
					),
				),
			),
		),
	).Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not find webhook worker: %w", err)
	}

	var actionNames []string
	for _, wwf := range ww.WebhookWorkerWorkflows() {
		for _, version := range wwf.Workflow().Versions() {
			for _, job := range version.Jobs() {
				for _, step := range job.Steps() {
					actionNames = append(actionNames, step.Action().ActionID)
				}
			}
		}
	}
	if len(actionNames) == 0 {
		return nil, fmt.Errorf("no actions found for webhook worker %s", id)
	}

	w.l.Debug().Msgf("starting webhook worker for actions: %s", actionNames)

	ctx, cancel := context.WithCancel(context.Background())
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

func (w *Worker) sendWebhook(ctx context.Context, action *client.Action, ww *db.WebhookWorkerModel) error {
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
