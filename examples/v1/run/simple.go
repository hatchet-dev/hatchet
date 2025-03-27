package main

import (
	"context"
	"fmt"
	"os"
	"time"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/v1/workflows"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
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

	// Define workflow runners map
	runnerMap := map[string]func() error{
		"simple": func() error {
			simple := v1_workflows.Simple(hatchet)
			result, err := simple.Run(v1_workflows.SimpleInput{
				Message: "Hello, World!",
			})
			if err != nil {
				return err
			}
			fmt.Println(result.TransformedMessage)
			return nil
		},
		"dag": func() error {
			dag := v1_workflows.DagWorkflow(hatchet)
			result, err := dag.Run(v1_workflows.DagInput{
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
			_, err := sleep.Run(v1_workflows.DurableSleepInput{
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
			workflow, err := durableEventWorkflow.RunNoWait(v1_workflows.DurableEventInput{
				Message: "Hello, World!",
			})

			if err != nil {
				return err
			}

			time.Sleep(10 * time.Second)

			hatchet.V0().Event().Push(context.Background(), "user:update", v1_workflows.EventData{
				Message: "User updated!",
			})

			_, err = workflow.Result()

			if err != nil {
				return err
			}
			fmt.Println("Durable event workflow completed")
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
