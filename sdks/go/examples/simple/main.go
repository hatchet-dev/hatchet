package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v0Worker "github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Declaring a Task
	type SimpleInput struct {
		Message string `json:"message"`
	}

	type SimpleOutput struct {
		Result string `json:"result"`
	}

	task := client.NewStandaloneTask("process-message", func(ctx hatchet.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{
			Result: "Processed: " + input.Message,
		}, nil
	})
	// !!

	worker, err := client.NewWorker("simple-worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	_ = func() error {
		// > Running a Task
		result, err := task.Run(context.Background(), SimpleInput{Message: "Hello, World!"})
		if err != nil {
			return err
		}
		// !!

		var resultOutput SimpleOutput
		err = result.Into(&result)
		if err != nil {
			return err
		}

		fmt.Println(resultOutput.Result)

		return nil
	}

	_ = func() error {
		// > Running a task without waiting
		runRef, err := task.RunNoWait(context.Background(), SimpleInput{Message: "Hello, World!"})
		if err != nil {
			return err
		}

		fmt.Println(runRef.RunId)
		// !!

		// > Subscribing to results
		result, err := runRef.Result()
		if err != nil {
			return err
		}

		var resultOutput SimpleOutput
		err = result.TaskOutput("process-message").Into(&resultOutput)
		if err != nil {
			return err
		}

		fmt.Println(resultOutput.Result)
		// !!

		workflow := client.NewWorkflow("parent-workflow")

		// > Spawning tasks from within a task
		parent := workflow.NewTask("parent-task", func(ctx hatchet.Context, input SimpleInput) (*SimpleOutput, error) {
			// Run the child task
			_, err := ctx.SpawnWorkflow(task.GetName(), SimpleInput{Message: input.Message}, &v0Worker.SpawnWorkflowOpts{})
			if err != nil {
				return nil, err
			}

			return &SimpleOutput{
				Result: "Processed: " + input.Message,
			}, nil
		})
		// !!

		_ = parent

		// > Running Multiple Tasks
		var results []string
		var resultsMutex sync.Mutex
		var errs []error
		var errsMutex sync.Mutex

		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			defer wg.Done()
			result, err := task.Run(context.Background(), SimpleInput{
				Message: "Hello, World!",
			})

			if err != nil {
				errsMutex.Lock()
				errs = append(errs, err)
				errsMutex.Unlock()
				return
			}

			resultsMutex.Lock()

			var resultOutput SimpleOutput
			err = result.Into(&resultOutput)
			if err != nil {
				return
			}
			results = append(results, resultOutput.Result)
			resultsMutex.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := task.Run(context.Background(), SimpleInput{
				Message: "Hello, Moon!",
			})

			if err != nil {
				errsMutex.Lock()
				errs = append(errs, err)
				errsMutex.Unlock()
				return
			}

			resultsMutex.Lock()

			var resultOutput SimpleOutput
			err = result.Into(&resultOutput)
			if err != nil {
				return
			}
			results = append(results, resultOutput.Result)
			resultsMutex.Unlock()
		}()

		wg.Wait()
		// !!

		return nil
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
