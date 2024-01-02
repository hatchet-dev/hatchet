package worker

import (
	"fmt"
	"reflect"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type Service struct {
	Name string

	worker *Worker
}

func (s *Service) On(t triggerConverter, workflow workflowConverter) error {
	apiWorkflow := workflow.ToWorkflow(s.Name)

	wt := &types.WorkflowTriggers{}

	t.ToWorkflowTriggers(wt)

	apiWorkflow.Triggers = *wt

	// create the workflow via the API
	err := s.worker.client.Admin().PutWorkflow(&apiWorkflow)

	if err != nil {
		return err
	}

	// register all steps as actions
	for actionId, fn := range workflow.ToActionMap(s.Name) {
		err := s.worker.registerAction(actionId, fn)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) RegisterAction(fn any) error {
	fnType := reflect.TypeOf(fn)

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("method must be a function")
	}

	if fnType.Name() == "" {
		return fmt.Errorf("function cannot be anonymous")
	}

	fnId := fnType.Name()

	actionId := fmt.Sprintf("%s:%s", s.Name, fnId)

	return s.worker.registerAction(actionId, fn)
}
