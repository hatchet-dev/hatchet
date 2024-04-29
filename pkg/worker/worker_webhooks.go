package worker

import (
	"fmt"
)

func (w *Worker) RegisterWebhook(t triggerConverter, url string, workflow workflowConverter) error {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return fmt.Errorf("could not load default service")
	}

	return svc.(*Service).RegisterWebhook(t, url, workflow)
}
