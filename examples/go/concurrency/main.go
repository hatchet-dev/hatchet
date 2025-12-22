package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ConcurrencyInput struct {
	Message string
	Tier    string
	Account string
}

type TransformedOutput struct {
	TransformedMessage string
}

func ConcurrencyRoundRobin(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Concurrency Strategy With Key
	var maxRuns int32 = 1
	strategy := types.GroupRoundRobin

	return client.NewStandaloneTask("simple-concurrency",
		func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
}

func MultipleConcurrencyKeys(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Multiple Concurrency Keys
	strategy := types.GroupRoundRobin
	var maxRuns int32 = 20

	return client.NewStandaloneTask("multi-concurrency",
		func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(
			types.Concurrency{
				Expression:    "input.Tier",
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
			}, types.Concurrency{
				Expression:    "input.Account",
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
			},
		),
	)
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	worker, err := client.NewWorker("concurrency-worker",
		hatchet.WithWorkflows(
			ConcurrencyRoundRobin(client),
			MultipleConcurrencyKeys(client),
		),
		hatchet.WithSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting worker with concurrency controls...")
	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
