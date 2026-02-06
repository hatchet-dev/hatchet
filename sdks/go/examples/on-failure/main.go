package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type FailureInput struct {
	Message     string `json:"message"`
	FailureType string `json:"failure_type"`
	ShouldFail  bool   `json:"should_fail"`
}

type TaskOutput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type FailureHandlerOutput struct {
	ErrorDetails   string `json:"error_details"`
	OriginalInput  string `json:"original_input"`
	FailureHandled bool   `json:"failure_handled"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create workflow with failure handling
	failureWorkflow := client.NewWorkflow("failure-handling-demo",
		hatchet.WithWorkflowDescription("Demonstrates workflow failure handling patterns"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task that may fail based on input
	_ = failureWorkflow.NewTask("potentially-failing-task", func(ctx hatchet.Context, input FailureInput) (TaskOutput, error) {
		log.Printf("Processing task with message: %s", input.Message)

		if input.ShouldFail {
			switch input.FailureType {
			case "panic":
				log.Println("Task will panic!")
				panic("intentional panic for demonstration")
			case "timeout":
				log.Println("Task will timeout!")
				time.Sleep(10 * time.Second) // This will timeout
			case "error":
				log.Println("Task will return error!")
				return TaskOutput{}, errors.New("intentional error for demonstration")
			default:
				log.Println("Task will return generic error!")
				return TaskOutput{}, errors.New("generic failure")
			}
		}

		log.Println("Task completed successfully")
		return TaskOutput{
			Status:  "success",
			Message: "Task completed: " + input.Message,
		}, nil
	}, hatchet.WithExecutionTimeout(5*time.Second))

	// Add failure handler to the workflow
	failureWorkflow.OnFailure(func(ctx hatchet.Context, input FailureInput) (FailureHandlerOutput, error) {
		log.Printf("Failure handler called for input: %s", input.Message)

		// Access step run errors to understand what failed
		stepErrors := ctx.StepRunErrors()
		var errorDetails string
		for stepName, errorMsg := range stepErrors {
			log.Printf("Step '%s' failed with error: %s", stepName, errorMsg)
			errorDetails += stepName + ": " + errorMsg + "; "
		}

		// Log failure details
		log.Printf("Handling failure for workflow. Error details: %s", errorDetails)

		// Perform cleanup or notification logic here
		log.Println("Performing failure cleanup...")

		return FailureHandlerOutput{
			FailureHandled: true,
			ErrorDetails:   errorDetails,
			OriginalInput:  input.Message,
		}, nil
	})

	// Create workflow with multi-step failure
	multiStepWorkflow := client.NewWorkflow("multi-step-failure-demo",
		hatchet.WithWorkflowDescription("Demonstrates failure handling in multi-step workflows"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// First task (succeeds)
	step1 := multiStepWorkflow.NewTask("first-step", func(ctx hatchet.Context, input FailureInput) (TaskOutput, error) {
		log.Printf("First step processing: %s", input.Message)
		return TaskOutput{
			Status:  "success",
			Message: "First step completed",
		}, nil
	})

	// Second task (may fail, depends on first)
	_ = multiStepWorkflow.NewTask("second-step", func(ctx hatchet.Context, input FailureInput) (TaskOutput, error) {
		// Get output from previous step
		var step1Output TaskOutput
		if err := ctx.StepOutput("first-step", &step1Output); err != nil {
			log.Printf("Failed to get first step output: %v", err)
			return TaskOutput{}, err
		}

		log.Printf("Second step processing after first step: %s", step1Output.Message)

		if input.ShouldFail {
			log.Println("Second step will fail!")
			return TaskOutput{}, errors.New("second step intentional failure")
		}

		return TaskOutput{
			Status:  "success",
			Message: "Second step completed after: " + step1Output.Message,
		}, nil
	}, hatchet.WithParents(step1))

	// Add failure handler for multi-step workflow
	// > On Failure Task
	multiStepWorkflow.OnFailure(func(ctx hatchet.Context, input FailureInput) (FailureHandlerOutput, error) {
		log.Printf("Multi-step failure handler called for input: %s", input.Message)

		stepErrors := ctx.StepRunErrors()
		var errorDetails string
		for stepName, errorMsg := range stepErrors {
			log.Printf("Multi-step: Step '%s' failed with error: %s", stepName, errorMsg)
			errorDetails += stepName + ": " + errorMsg + "; "
		}

		// Access successful step outputs for cleanup
		var step1Output TaskOutput
		if err := ctx.StepOutput("first-step", &step1Output); err == nil {
			log.Printf("First step completed successfully with: %s", step1Output.Message)
		}

		return FailureHandlerOutput{
			FailureHandled: true,
			ErrorDetails:   "Multi-step workflow failed: " + errorDetails,
			OriginalInput:  input.Message,
		}, nil
	})
	// !!

	// Create a worker with both workflows
	worker, err := client.NewWorker("failure-handling-worker",
		hatchet.WithWorkflows(failureWorkflow, multiStepWorkflow),
		hatchet.WithSlots(3),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run workflow instances to demonstrate failure handling
	go func() {
		time.Sleep(2 * time.Second)

		// Demo 1: Successful workflow
		log.Println("\n=== Demo 1: Successful Workflow ===")
		_, err := client.Run(context.Background(), "failure-handling-demo", FailureInput{
			Message:    "This workflow will succeed",
			ShouldFail: false,
		})
		if err != nil {
			log.Printf("Error in successful workflow: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Demo 2: Workflow with error
		log.Println("\n=== Demo 2: Workflow with Error ===")
		_, err = client.Run(context.Background(), "failure-handling-demo", FailureInput{
			Message:     "This workflow will fail with error",
			ShouldFail:  true,
			FailureType: "error",
		})
		if err != nil {
			log.Printf("Expected error in failing workflow: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Demo 3: Multi-step workflow failure
		log.Println("\n=== Demo 3: Multi-step Workflow Failure ===")
		_, err = client.Run(context.Background(), "multi-step-failure-demo", FailureInput{
			Message:    "This multi-step workflow will fail in second step",
			ShouldFail: true,
		})
		if err != nil {
			log.Printf("Expected error in multi-step workflow: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Demo 4: Multi-step workflow success
		log.Println("\n=== Demo 4: Multi-step Workflow Success ===")
		_, err = client.Run(context.Background(), "multi-step-failure-demo", FailureInput{
			Message:    "This multi-step workflow will succeed",
			ShouldFail: false,
		})
		if err != nil {
			log.Printf("Error in successful multi-step workflow: %v", err)
		}
	}()

	log.Println("Starting worker for failure handling demos...")
	log.Println("Features demonstrated:")
	log.Println("  - Workflow failure handlers (OnFailure)")
	log.Println("  - Error details access in failure handlers")
	log.Println("  - Multi-step workflow failure handling")
	log.Println("  - Successful step output access during failure")
	log.Println("  - Different failure types (error, timeout, panic)")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
