package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	idempotentTask := IdempotentTask(client)

	worker, err := client.NewWorker("idempotency-worker", hatchet.WithWorkflows(idempotentTask))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Fatalf("failed to start worker: %v", err)
		}
	}()

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
