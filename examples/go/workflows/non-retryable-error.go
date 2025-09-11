package v1_workflows

import (
	"errors"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type NonRetryableInput struct{}
type NonRetryableResult struct{}

// NonRetryableError returns a workflow which throws a non-retryable error
func NonRetryableError(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[NonRetryableInput, NonRetryableResult] {
	// > Non Retryable Error
	retries := factory.NewTask(
		create.StandaloneTask{
			Name:    "non-retryable-task",
			Retries: 3,
		}, func(ctx worker.HatchetContext, input NonRetryableInput) (*NonRetryableResult, error) {
			return nil, worker.NewNonRetryableError(errors.New("intentional failure"))
		},
		hatchet,
	)

	return retries
}
