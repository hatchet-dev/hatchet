// Deprecated: This package is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead.
package factory

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

// Deprecated: NewWorkflow is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead.
func NewWorkflow[I any, O any](opts create.WorkflowCreateOpts[I], client v1.HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = client.V0()
	}

	return workflow.NewWorkflowDeclaration[I, O](opts, v0)
}
