package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
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
	strategy := hatchet.GroupRoundRobin

	return client.NewStandaloneTask("simple-concurrency",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
	// !!
}

func MultipleConcurrencyKeys(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Multiple Concurrency Keys
	strategy := hatchet.GroupRoundRobin
	var maxRuns int32 = 20

	return client.NewStandaloneTask("multi-concurrency",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(
			hatchet.Concurrency{
				Expression:    "input.Tier",
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
			}, hatchet.Concurrency{
				Expression:    "input.Account",
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
			},
		),
	)
	// !!
}

func ConcurrencyCancelInProgress(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Cancel In Progress
	var maxRuns int32 = 1
	strategy := hatchet.CancelInProgress

	return client.NewStandaloneTask("cancel-in-progress",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
	// !!
}

func ConcurrencyCancelNewest(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Cancel Newest
	var maxRuns int32 = 1
	strategy := hatchet.CancelNewest

	return client.NewStandaloneTask("cancel-newest",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
	// !!
}

func ConcurrencyCancelQueuedExceptNewest(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Cancel Queued Except Newest
	var maxRuns int32 = 1
	strategy := hatchet.CancelQueuedExceptNewest

	return client.NewStandaloneTask("cancel-queued-except-newest",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
	// !!
}

func ConcurrencyCancelQueuedExceptOldest(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Cancel Queued Except Oldest
	var maxRuns int32 = 1
	strategy := hatchet.CancelQueuedExceptOldest

	return client.NewStandaloneTask("cancel-queued-except-oldest",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:    "input.GroupKey",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)
	// !!
}

func SharedConcurrencyStrategy(client *hatchet.Client) (*hatchet.StandaloneTask, *hatchet.StandaloneTask) {
	// > Shared Concurrency Strategy
	var maxRuns int32 = 1
	strategy := hatchet.GroupRoundRobin

	// A tenant-scoped strategy is shared across workflows: every task declaring the
	// same name consumes the same concurrency limit. Re-registering the name updates
	// it in place.
	sharedLimit := hatchet.Concurrency{
		Name:           "example-shared-limit",
		IsTenantScoped: true,
		Expression:     "input.Account",
		MaxRuns:        &maxRuns,
		LimitStrategy:  &strategy,
	}

	// two different tasks, in two different workflows, consuming one limit
	syncCrm := client.NewStandaloneTask("sync-crm",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			return &TransformedOutput{TransformedMessage: input.Message}, nil
		},
		hatchet.WithWorkflowConcurrency(sharedLimit),
	)

	generateReport := client.NewStandaloneTask("generate-report",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			return &TransformedOutput{TransformedMessage: input.Message}, nil
		},
		hatchet.WithWorkflowConcurrency(sharedLimit),
	)

	return syncCrm, generateReport
	// !!
}

func DynamicMaxRuns(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Dynamic Max Runs
	var maxRuns int32 = 1
	strategy := hatchet.GroupRoundRobin

	// MaxRunsExpression computes each concurrency group's limit from the task's
	// input; MaxRuns stays the static default.
	maxRunsExpression := "input.Tier == 'premium' ? 10 : 1"

	return client.NewStandaloneTask("dynamic-concurrency",
		func(ctx hatchet.Context, input ConcurrencyInput) (*TransformedOutput, error) {
			return &TransformedOutput{TransformedMessage: input.Message}, nil
		},
		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
			Expression:        "input.Account",
			MaxRuns:           &maxRuns,
			LimitStrategy:     &strategy,
			MaxRunsExpression: &maxRunsExpression,
		}),
	)
	// !!
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	syncCrm, generateReport := SharedConcurrencyStrategy(client)

	// > Slots
	worker, err := client.NewWorker("concurrency-worker",
		hatchet.WithWorkflows(
			ConcurrencyRoundRobin(client),
			MultipleConcurrencyKeys(client),
			ConcurrencyCancelInProgress(client),
			ConcurrencyCancelNewest(client),
			ConcurrencyCancelQueuedExceptNewest(client),
			ConcurrencyCancelQueuedExceptOldest(client),
			syncCrm,
			generateReport,
			DynamicMaxRuns(client),
		),
		hatchet.WithSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}
	// !!

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting worker with concurrency controls...")
	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
