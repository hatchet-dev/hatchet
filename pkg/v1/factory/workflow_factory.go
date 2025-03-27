package factory

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

// NewWorkflow creates a new workflow declaration with the specified input and output types before a client is initialized.
// This function is used to create strongly typed workflow declarations with the given client.
// NOTE: This is placed on the client due to circular dependency concerns.
func NewWorkflow[I any, O any](opts create.WorkflowCreateOpts[I], client v1.HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = client.V0()
	}

	return workflow.NewWorkflowDeclaration[I, O](opts, v0)
}
