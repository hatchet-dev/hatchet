package worker

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type Service struct {
	Name string

	mws *middlewares

	worker *Worker
}

func (s *Service) Use(mws ...MiddlewareFunc) {
	s.mws.add(mws...)
}

func (s *Service) On(t triggerConverter, workflow workflowConverter) error {
	namespace := s.worker.client.Namespace()

	apiWorkflow := workflow.ToWorkflow(s.Name, namespace)

	wt := &types.WorkflowTriggers{}

	t.ToWorkflowTriggers(wt, namespace)

	apiWorkflow.Triggers = *wt

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

type registerActionOpts struct {
	name string
}

type RegisterActionOpt func(*registerActionOpts)

func WithActionName(name string) RegisterActionOpt {
	return func(opts *registerActionOpts) {
		opts.name = name
	}
}

func (s *Service) RegisterAction(fn any, opts ...RegisterActionOpt) error {
	fnOpts := &registerActionOpts{}

	for _, opt := range opts {
		opt(fnOpts)
	}

	if fnOpts.name == "" {
		fnOpts.name = getFnName(fn)
	}

	return s.worker.registerAction(s.Name, fnOpts.name, fn)
}

func (s *Service) Call(verb string) *WorkflowStep {
	actionId := fmt.Sprintf("%s:%s", s.Name, verb)

	registeredAction, exists := s.worker.actions[actionId]

	if !exists {
		panic(fmt.Sprintf("action %s does not exist", actionId))
	}

	return &WorkflowStep{
		Function: registeredAction.MethodFn(),
		Name:     verb,
	}
}
