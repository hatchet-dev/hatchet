package worker

import (
	"fmt"

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

	// register all steps as actions
	for actionId, fn := range workflow.ToActionMap(s.Name) {
		parsedAction, err := types.ParseActionID(actionId)

		if err != nil {
			return err
		}

		if parsedAction.Service != s.Name {
			// check that it's concurrency, otherwise throw error
			if parsedAction.Service != "concurrency" {
				return fmt.Errorf("action %s does not belong to service %s", actionId, s.Name)
			}
		}

		err = s.worker.registerAction(parsedAction.Service, parsedAction.Verb, fn)

		if err != nil {
			return err
		}
	}

	return nil
}
