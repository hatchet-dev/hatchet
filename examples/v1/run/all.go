package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	v1_workflows "github.com/hatchet-dev/hatchet/examples/v1/workflows"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
	"github.com/oapi-codegen/runtime/types"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}

	// Get workflow name from command line arguments
	var workflowName string
	if len(os.Args) > 1 {
		workflowName = os.Args[1]
		fmt.Println("workflow name provided:", workflowName)
	} else {
		fmt.Println("No workflow name provided. Defaulting to 'simple'")
		workflowName = "simple"
	}

	ctx := context.Background()

	// Define workflow runners map
	runnerMap := map[string]func() error{
		"simple": func() error {
			simple := v1_workflows.Simple(hatchet)
			result, err := simple.Run(ctx, v1_workflows.SimpleInput{
				Message: "Hello, World!",
			})
			if err != nil {
				return err
			}
			fmt.Println(result.TransformedMessage)
			return nil
		},
		"child": func() error {
			parent := v1_workflows.Parent(hatchet)

			result, err := parent.Run(ctx, v1_workflows.ParentInput{
				N: 50,
			})

			if err != nil {
				return err
			}
			fmt.Println("Parent result:", result.Result)
			return nil
		},
		"dag": func() error {
			dag := v1_workflows.DagWorkflow(hatchet)
			result, err := dag.Run(ctx, v1_workflows.DagInput{
				Message: "Hello, DAG!",
			})
			if err != nil {
				return err
			}
			fmt.Println(result.Step1.Step)
			fmt.Println(result.Step2.Step)
			return nil
		},
		"sleep": func() error {
			sleep := v1_workflows.DurableSleep(hatchet)
			_, err := sleep.Run(ctx, v1_workflows.DurableSleepInput{
				Message: "Hello, Sleep!",
			})
			if err != nil {
				return err
			}
			fmt.Println("Sleep workflow completed")
			return nil
		},
		"durable-event": func() error {
			durableEventWorkflow := v1_workflows.DurableEvent(hatchet)
			run, err := durableEventWorkflow.RunNoWait(ctx, v1_workflows.DurableEventInput{
				Message: "Hello, World!",
			})

			if err != nil {
				return err
			}

			_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
				ExternalIds: &[]types.UUID{uuid.MustParse(run.WorkflowRunId())},
			})

			if err != nil {
				return nil // We expect an error here
			}

			_, err = run.Result()

			if err != nil {
				fmt.Println("Received expected error:", err)
				return nil // We expect an error here
			}
			fmt.Println("Cancellation workflow completed unexpectedly")
			return nil
		},
		"timeout": func() error {
			timeout := v1_workflows.Timeout(hatchet)
			_, err := timeout.Run(ctx, v1_workflows.TimeoutInput{})
			if err != nil {
				fmt.Println("Received expected error:", err)
				return nil // We expect an error here
			}
			fmt.Println("Timeout workflow completed unexpectedly")
			return nil
		},
		"sticky": func() error {
			sticky := v1_workflows.Sticky(hatchet)
			result, err := sticky.Run(ctx, v1_workflows.StickyInput{})
			if err != nil {
				return err
			}
			fmt.Println("Value from child workflow:", result.Result)
			return nil
		},
		"sticky-dag": func() error {
			stickyDag := v1_workflows.StickyDag(hatchet)
			result, err := stickyDag.Run(ctx, v1_workflows.StickyInput{})
			if err != nil {
				return err
			}
			fmt.Println("Value from task 1:", result.StickyTask1.Result)
			fmt.Println("Value from task 2:", result.StickyTask2.Result)
			return nil
		},
		"retries": func() error {
			retries := v1_workflows.Retries(hatchet)
			_, err := retries.Run(ctx, v1_workflows.RetriesInput{})
			if err != nil {
				fmt.Println("Received expected error:", err)
				return nil // We expect an error here
			}
			fmt.Println("Retries workflow completed unexpectedly")
			return nil
		},
		"retries-count": func() error {
			retriesCount := v1_workflows.RetriesWithCount(hatchet)
			result, err := retriesCount.Run(ctx, v1_workflows.RetriesWithCountInput{})
			if err != nil {
				return err
			}
			fmt.Println("Result message:", result.Message)
			return nil
		},
		"with-backoff": func() error {
			withBackoff := v1_workflows.WithBackoff(hatchet)
			_, err := withBackoff.Run(ctx, v1_workflows.BackoffInput{})
			if err != nil {
				fmt.Println("Received expected error:", err)
				return nil // We expect an error here
			}
			fmt.Println("WithBackoff workflow completed unexpectedly")
			return nil
		},
		"non-retryable": func() error {
			nonRetryable := v1_workflows.NonRetryableError(hatchet)
			_, err := nonRetryable.Run(ctx, v1_workflows.NonRetryableInput{})
			if err != nil {
				fmt.Println("Received expected error:", err)
				return nil // We expect an error here
			}
			fmt.Println("NonRetryable workflow completed unexpectedly")
			return nil
		},
		"on-cron": func() error {
			cronTask := v1_workflows.OnCron(hatchet)
			result, err := cronTask.Run(ctx, v1_workflows.OnCronInput{
				Message: "Hello, Cron!",
			})
			if err != nil {
				return err
			}
			fmt.Println("Cron task result:", result.Job.TransformedMessage)
			return nil
		},
		"priority": func() error {

			nRuns := 10
			priorityWorkflow := v1_workflows.Priority(hatchet)

			for i := 0; i < nRuns; i++ {
				randomPrio := int32(rand.Intn(3) + 1)

				fmt.Println("Random priority:", randomPrio)

				priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{
					UserId: "1234",
				}, client.WithRunMetadata(map[string]int32{"priority": randomPrio}), client.WithPriority(randomPrio))
			}

			triggerAt := time.Now().Add(time.Second + 5)

			for i := 0; i < nRuns; i++ {
				randomPrio := int32(rand.Intn(3) + 1)

				fmt.Println("Random priority:", randomPrio)

				priorityWorkflow.Schedule(ctx, triggerAt, v1_workflows.PriorityInput{
					UserId: "1234",
				}, client.WithRunMetadata(map[string]int32{"priority": randomPrio}), client.WithPriority(randomPrio))
			}

			return nil
		},
	}

	// Lookup workflow runner from map
	runner, ok := runnerMap[workflowName]
	if !ok {
		fmt.Println("Invalid workflow name provided. Usage: go run examples/v1/run/simple.go [workflow-name]")
		fmt.Println("Available workflows:", getAvailableWorkflows(runnerMap))
		os.Exit(1)
	}

	// Run the selected workflow
	err = runner()
	if err != nil {
		panic(err)
	}
}

// Helper function to get available workflows as a formatted string
func getAvailableWorkflows(runnerMap map[string]func() error) string {
	var workflows string
	count := 0
	for name := range runnerMap {
		if count > 0 {
			workflows += ", "
		}
		workflows += fmt.Sprintf("'%s'", name)
		count++
	}
	return workflows
}
