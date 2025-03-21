package v1

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

type HatchetClient interface {
	V0() v0Client.Client

	Workflow(opts workflow.CreateOpts) workflow.WorkflowDeclaration[any, any]
	Worker(opts worker.CreateOpts, optFns ...func(*worker.WorkerImpl)) (worker.Worker, error)
}

type v1HatchetClientImpl struct {
	v0 *v0Client.Client
}

func NewHatchetClient(config ...Config) (HatchetClient, error) {
	cf := &v0Config.ClientConfigFile{}

	if len(config) > 0 {
		opts := config[0]
		cf = mapConfigToCF(opts)
	}

	client, err := v0Client.NewFromConfigFile(cf)
	if err != nil {
		return nil, err
	}

	return &v1HatchetClientImpl{
		v0: &client,
	}, nil
}

func (c *v1HatchetClientImpl) V0() v0Client.Client {
	return *c.v0
}

func (c *v1HatchetClientImpl) Workflow(opts workflow.CreateOpts) workflow.WorkflowDeclaration[any, any] {
	return workflow.NewWorkflowDeclaration[any, any](opts, c.v0)
}

func (c *v1HatchetClientImpl) Worker(opts worker.CreateOpts, optFns ...func(*worker.WorkerImpl)) (worker.Worker, error) {
	return worker.NewWorker(c.v0, opts, optFns...)
}

// NOTE: i don't love this on the client but there is a circular
// dependency if it is in the workflow package
func WorkflowFactory[I any, O any](opts workflow.CreateOpts, client *HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = (*client).V0()
	}

	return workflow.NewWorkflowDeclaration[I, O](opts, &v0)
}
