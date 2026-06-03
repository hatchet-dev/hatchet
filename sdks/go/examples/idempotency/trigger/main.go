package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type IdempotencyInput struct {
	ID string `json:"id"`
}

type IdempotencyOutput struct {
	Result string `json:"result"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	idempotentTask := client.NewStandaloneTask(
		"idempotent-task",
		func(ctx hatchet.Context, input IdempotencyInput) (*IdempotencyOutput, error) {
			return &IdempotencyOutput{
				Result: fmt.Sprintf("Hello, world from task %s", input.ID),
			}, nil
		},
		hatchet.WithWorkflowIdempotency(hatchet.IdempotencyConfig{
			Expression: "input.id",
			TTL:        time.Minute,
		}),
	)

	ctx := context.Background()

	// > trigger
	ref1, err := idempotentTask.RunNoWait(ctx, IdempotencyInput{ID: "123"})
	if err != nil {
		log.Fatalf("unexpected error on first run: %v", err)
	}

	ref2, err := idempotentTask.RunNoWait(ctx, IdempotencyInput{ID: "123"})

	var runID2 string

	if err != nil {
		if idempErr, ok := hatchet.IsIdempotencyCollisionError(err); ok {
			fmt.Printf("Run %s already exists for this idempotency key\n", idempErr.ExistingRunExternalId)
			runID2 = idempErr.ExistingRunExternalId
		} else {
			log.Fatalf("unexpected error on second run: %v", err)
		}
	} else {
		runID2 = ref2.RunId
	}
	// !!

	fmt.Printf("First run ID: %s\n", ref1.RunId)
	fmt.Printf("Second run ID: %s\n", runID2)
}
