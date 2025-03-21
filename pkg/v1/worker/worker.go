package worker

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

type CreateOpts struct {
	Name string
}

type Worker interface {
	// TODO bind runs and things
	Start() error
	RegisterWorkflows(workflows ...workflow.WorkflowBase) error
}

type WorkerImpl struct {
	v0 *v0Client.Client

	name      string
	workflows []workflow.WorkflowBase
}

func WithWorkflows(workflows ...workflow.WorkflowBase) func(*WorkerImpl) {
	return func(w *WorkerImpl) {
		w.workflows = workflows
	}
}

func NewWorker(client *client.Client, opts CreateOpts, optFns ...func(*WorkerImpl)) (Worker, error) {
	w := &WorkerImpl{
		v0:   client,
		name: opts.Name,
	}

	for _, optFn := range optFns {
		optFn(w)
	}

	return w, nil
}

func (w *WorkerImpl) RegisterWorkflows(workflows ...workflow.WorkflowBase) error {
	w.workflows = append(w.workflows, workflows...)
	return nil
}

func (w *WorkerImpl) Start() error {
	for _, workflow := range w.workflows {
		fmt.Println(workflow.Name())
	}
	return nil
}
