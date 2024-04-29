package worker

import (
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

func (s *Service) RegisterWebhook(t triggerConverter, url string, workflow workflowConverter) error {
	namespace := s.worker.client.Namespace()

	apiWorkflow := workflow.ToWorkflow(s.Name, namespace)

	wt := &types.WorkflowTriggers{}

	t.ToWorkflowTriggers(wt, namespace)

	apiWorkflow.Triggers = *wt
	apiWorkflow.Webhook = &url

	// create the workflow via the API
	err := s.worker.client.Admin().PutWorkflow(&apiWorkflow)

	if err != nil {
		return err
	}

	return nil
}
