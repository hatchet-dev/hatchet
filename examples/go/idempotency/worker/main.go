package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type IdempotencyInput struct {
	ID string `json:"id"`
}

type IdempotencyOutput struct {
	Result string `json:"result"`
}

// > idempotency
func IdempotentTask(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask(
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
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	worker, err := client.NewWorker("idempotency-worker",
		hatchet.WithWorkflows(IdempotentTask(client)),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
