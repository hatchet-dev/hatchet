package worker

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

func (w *Worker) RegisterWebhook(t triggerConverter, url string, workflow workflowConverter) error {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return fmt.Errorf("could not load default service")
	}

	if err := svc.(*Service).RegisterWebhook(t, url, workflow); err != nil {
		return fmt.Errorf("could not register webhook: %w", err)
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
		return fmt.Errorf("could not register worker: %w", err)
	}

	return nil
}
