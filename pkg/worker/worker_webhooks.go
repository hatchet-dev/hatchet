package worker

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type WebhookWorker struct {
	w *Worker
}

func (w *Worker) RegisterWebhook(t triggerConverter, url string, workflow workflowConverter) (*WebhookWorker, error) {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return nil, fmt.Errorf("could not load default service")
	}

	if err := svc.(*Service).RegisterWebhook(t, url, workflow); err != nil {
		return nil, fmt.Errorf("could not register webhook: %w", err)
	}

	actionNames := []string{}

	for _, action := range w.actions {
		actionNames = append(actionNames, action.Name())
	}

	if err := w.client.Dispatcher().RegisterWorker(context.Background(), &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    actionNames,
		MaxRuns:    w.maxRuns,
		Webhook:    true,
	}); err != nil {
		return nil, fmt.Errorf("could not register worker: %w", err)
	}

	return &WebhookWorker{w: w}, nil
}
