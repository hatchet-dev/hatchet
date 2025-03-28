package v1_workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
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

	// ❓ Declaring a Task
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
	// ‼️

	// Example of running a task
	_ = func() error {
		// ❓ Running a Task
		result, err := simple.Run(context.Background(), SimpleInput{Message: "Hello, World!"})
		if err != nil {
			return err
		}
		fmt.Println(result.TransformedMessage)
		// ‼️
		return nil
	}

	// Example of registering a task on a worker
	_ = func() error {
		// ❓ Declaring a Worker
		w, err := hatchet.Worker(create.WorkerOpts{
			Name: "simple-worker",
			// Workflows: []workflow.WorkflowBase{
			// 	simple,
			// },
		})
		if err != nil {
			return err
		}
		err = w.StartBlocking()
		if err != nil {
			return err
		}
		// ‼️
		return nil
	}

	return simple
}
