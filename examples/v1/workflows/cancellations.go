package v1_workflows

import (
	"errors"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type CancellationInput struct{}
type CancellationResult struct {
	Completed bool
}

func Cancellation(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[CancellationInput, CancellationResult] {

	// ❓ Cancelled task
	// Create a task that sleeps for 10 seconds and checks if it was cancelled
	cancellation := factory.NewTask(
		create.StandaloneTask{
			Name: "cancellation-task",
		}, func(ctx worker.HatchetContext, input CancellationInput) (*CancellationResult, error) {
			// Sleep for 10 seconds
			time.Sleep(10 * time.Second)

			// Check if the context was cancelled
			select {
			case <-ctx.Done():
				return nil, errors.New("Task was cancelled")
			default:
				// Continue execution
			}

			return &CancellationResult{
				Completed: true,
			}, nil
		},
		hatchet,
	)
	// !!

	return cancellation
}
