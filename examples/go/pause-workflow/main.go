package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type OrderInput struct {
	OrderID int    `json:"orderId"`
	Item    string `json:"item"`
}

type OrderOutput struct {
	OrderID int    `json:"orderId"`
	Status  string `json:"status"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	workflow := client.NewWorkflow("pause-example")

	workflow.NewTask("process-order", func(ctx hatchet.Context, input OrderInput) (OrderOutput, error) {
		log.Printf("processing order %d (%s)", input.OrderID, input.Item)
		time.Sleep(5 * time.Second)

		return OrderOutput{
			OrderID: input.OrderID,
			Status:  fmt.Sprintf("fulfilled: %s", input.Item),
		}, nil
	})

	// slot is made 1, so only 1 run can be processed at a time. To clearly see the pause workflow in action.
	worker, err := client.NewWorker("pause-worker", hatchet.WithWorkflows(workflow), hatchet.WithSlots(1))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	cleanup, err := worker.Start()
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
	defer cleanup() //nolint:errcheck

	time.Sleep(2 * time.Second)

	// > Trigger runs
	log.Println("Triggering 5 runs...")

	inputs := make([]hatchet.RunManyOpt, 5)
	for i := range inputs {
		inputs[i] = hatchet.RunManyOpt{
			Input: OrderInput{OrderID: i + 1, Item: fmt.Sprintf("item-%d", i+1)},
		}
	}

	runRefs, err := workflow.RunMany(context.Background(), inputs)
	if err != nil {
		log.Fatalf("failed to enqueue runs: %v", err)
	}
	log.Printf("Enqueued %d runs", len(runRefs))

	time.Sleep(3 * time.Second)

	// > Pause workflow
	log.Println("Pausing workflow...")

	_, err = client.Workflows().Pause(context.Background(), "pause-example")
	if err != nil {
		log.Fatalf("failed to pause workflow: %v", err)
	}
	log.Println("Workflow paused — in-flight tasks cancelled, queued tasks held.")

	// > Check pause state
	isPaused, err := client.Workflows().IsPaused(context.Background(), "pause-example")
	if err != nil {
		log.Fatalf("failed to check pause state: %v", err)
	}
	log.Printf("isPaused = %v", isPaused)

	// > Trigger runs while paused
	log.Println("Triggering 3 runs while paused (held until resume)...")
	for i := range 3 {
		_, err := workflow.RunNoWait(context.Background(), OrderInput{
			OrderID: 100 + i + 1,
			Item:    fmt.Sprintf("paused-item-%d", i+1),
		})
		if err != nil {
			log.Printf("failed to trigger run: %v", err)
		}
	}

	log.Println("Waiting 30 seconds before resuming...")
	time.Sleep(30 * time.Second)

	// > Unpause workflow
	log.Println("Unpausing workflow...")

	_, err = client.Workflows().Unpause(context.Background(), "pause-example")
	if err != nil {
		log.Fatalf("failed to unpause workflow: %v", err)
	}
	log.Println("Workflow unpaused — all held runs will now be dispatched.")

	<-interruptCtx.Done()
}
