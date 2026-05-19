package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	type StepOutput struct {
		Status string `json:"status"`
	}

	workflow := client.NewWorkflow("slot-release-workflow")

	// > SlotRelease

	_ = workflow.NewTask("step1", func(ctx hatchet.Context, _ any) (*StepOutput, error) {
		fmt.Println("RESOURCE INTENSIVE PROCESS")
		time.Sleep(10 * time.Second)

		// Release the slot after the resource-intensive process,
		// so that other steps can run on this worker.
		if releaseErr := ctx.ReleaseSlot(); releaseErr != nil {
			return nil, fmt.Errorf("failed to release slot: %w", releaseErr)
		}

		fmt.Println("NON RESOURCE INTENSIVE PROCESS")

		return &StepOutput{Status: "success"}, nil
	})

	// !!

	worker, err := client.NewWorker("slot-release-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Printf("failed to start worker: %v\n", err)
	}
}
