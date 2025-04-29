package v1_workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	v1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleInput struct {
	Message string
}
type SimpleResult struct {
	TransformedMessage string
}

func Simple(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {

	// Create a simple standalone task using the task factory
	// Note the use of typed generics for both input and output

	// > Declaring a Task
	simple := factory.NewTask(
		create.StandaloneTask{
			Name: "simple-task",
		}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {
			// Transform the input message to lowercase
			return &SimpleResult{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
		hatchet,
	)

	// Example of running a task
	_ = func() error {
		// > Running a Task
		result, err := simple.Run(context.Background(), SimpleInput{Message: "Hello, World!"})
		if err != nil {
			return err
		}
		fmt.Println(result.TransformedMessage)
		return nil
	}

	// Example of registering a task on a worker
	_ = func() error {
		// > Declaring a Worker
		w, err := hatchet.Worker(v1worker.WorkerOpts{
			Name: "simple-worker",
			Workflows: []workflow.WorkflowBase{
				simple,
			},
		})
		if err != nil {
			return err
		}
		err = w.StartBlocking(context.Background())
		if err != nil {
			return err
		}
		return nil
	}

	return simple
}

func ParentTask(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {

	// > Spawning Tasks from within a Task
	simple := Simple(hatchet)

	parent := factory.NewTask(
		create.StandaloneTask{
			Name: "parent-task",
		}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {

			// Run the child task
			child, err := workflow.RunChildWorkflow(ctx, simple, SimpleInput{Message: input.Message})
			if err != nil {
				return nil, err
			}

			// Transform the input message to lowercase
			return &SimpleResult{
				TransformedMessage: child.TransformedMessage,
			}, nil
		},
		hatchet,
	)

	return parent
}
