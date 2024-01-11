package worker

import (
	"github.com/hatchet-dev/hatchet/pkg/client"
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
	apiWorkflow := workflow.ToWorkflow(s.Name)

	wt := &types.WorkflowTriggers{}

	t.ToWorkflowTriggers(wt)

	apiWorkflow.Triggers = *wt

	// create the workflow via the API
	err := s.worker.client.Admin().PutWorkflow(&apiWorkflow, client.WithAutoVersion())

	if err != nil {
		return err
	}

	// register all steps as actions
	for actionId, fn := range workflow.ToActionMap(s.Name) {
		err := s.worker.registerAction(s.Name, actionId, fn)

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
