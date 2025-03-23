package v1

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

// HatchetClient is the main interface for interacting with the Hatchet task orchestrator.
// It provides access to workflow creation, worker registration, and legacy V0 client functionality.
type HatchetClient interface {
	// V0 returns the underlying V0 client for backward compatibility.
	V0() v0Client.Client

	// Workflow creates a new workflow declaration with the provided options.
	Workflow(opts workflow.CreateOpts) workflow.WorkflowDeclaration[any, any]

	// Worker creates and configures a new worker with the provided options and optional configuration functions.
	// @example
	// ```go
	// worker, err := hatchet.Worker(worker.CreateOpts{
	// 	   Name: "my-worker",
	//   },
	//   v1.WithWorkflows(simple)
	// )
	// ```
	Worker(opts worker.CreateOpts, optFns ...func(*worker.WorkerImpl)) (worker.Worker, error)

	// TODO bind run, cron, schedule, etc.
}

// v1HatchetClientImpl is the implementation of the HatchetClient interface.
type v1HatchetClientImpl struct {
	v0 *v0Client.Client
}

// NewHatchetClient creates a new V1 Hatchet client with the provided configuration.
// If no configuration is provided, default settings will be used.
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

// V0 returns the underlying V0 client for backward compatibility.
func (c *v1HatchetClientImpl) V0() v0Client.Client {
	return *c.v0
}

// Workflow creates a new workflow declaration with the provided options.
func (c *v1HatchetClientImpl) Workflow(opts workflow.CreateOpts) workflow.WorkflowDeclaration[any, any] {
	var v0 v0Client.Client
	if c.v0 != nil {
		v0 = *c.v0
	}

	return workflow.NewWorkflowDeclaration[any, any](opts, &v0)
}

// Worker creates and configures a new worker with the provided options and optional configuration functions.
func (c *v1HatchetClientImpl) Worker(opts worker.CreateOpts, optFns ...func(*worker.WorkerImpl)) (worker.Worker, error) {
	return worker.NewWorker(c.v0, opts, optFns...)
}

// WorkflowFactory creates a new workflow declaration with the specified input and output types before a client is initialized.
// This function is used to create strongly typed workflow declarations with the given client.
// NOTE: This is placed on the client due to circular dependency concerns.
func WorkflowFactory[I any, O any](opts workflow.CreateOpts, client *HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = (*client).V0()
	}

	return workflow.NewWorkflowDeclaration[I, O](opts, &v0)
}
