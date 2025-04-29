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

type TimeoutInput struct{}
type TimeoutResult struct {
	Completed bool
}

func Timeout(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[TimeoutInput, TimeoutResult] {

	// > Execution Timeout
	// Create a task with a timeout of 3 seconds that tries to sleep for 10 seconds
	timeout := factory.NewTask(
		create.StandaloneTask{
			Name:             "timeout-task",
			ExecutionTimeout: 3 * time.Second, // Task will timeout after 3 seconds
		}, func(ctx worker.HatchetContext, input TimeoutInput) (*TimeoutResult, error) {
			// Sleep for 10 seconds
			time.Sleep(10 * time.Second)

			// Check if the context was cancelled due to timeout
			select {
			case <-ctx.Done():
				return nil, errors.New("TASK TIMED OUT")
			default:
				// Continue execution
			}

			return &TimeoutResult{
				Completed: true,
			}, nil
		},
		hatchet,
	)

	return timeout
}

func RefreshTimeout(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[TimeoutInput, TimeoutResult] {

	// > Refresh Timeout
	timeout := factory.NewTask(
		create.StandaloneTask{
			Name:             "timeout-task",
			ExecutionTimeout: 3 * time.Second, // Task will timeout after 3 seconds
		}, func(ctx worker.HatchetContext, input TimeoutInput) (*TimeoutResult, error) {

			// Refresh the timeout by 10 seconds (new timeout will be 13 seconds)
			ctx.RefreshTimeout("10s")

			// Sleep for 10 seconds
			time.Sleep(10 * time.Second)

			// Check if the context was cancelled due to timeout
			select {
			case <-ctx.Done():
				return nil, errors.New("TASK TIMED OUT")
			default:
				// Continue execution
			}

			return &TimeoutResult{
				Completed: true,
			}, nil
		},
		hatchet,
	)

	return timeout
}
