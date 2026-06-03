package main

import (
	"fmt"
	"time"

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

// !!
