package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	evictionTTLSeconds = 5
	longSleepSeconds   = 15
	eventKey           = "durable-eviction:event"
)

type EmptyInput struct{}

type EvictionOutput struct {
	Status string `json:"status"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Eviction Policy
	evictionPolicy := &hatchet.EvictionPolicy{
		TTL:                   evictionTTLSeconds * time.Second,
		AllowCapacityEviction: true,
		Priority:              0,
	}

	// > Evictable Sleep
	evictableSleep := client.NewStandaloneDurableTask("evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			if _, err := ctx.SleepFor(longSleepSeconds * time.Second); err != nil {
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	// > Evictable Wait For Event
	evictableWaitForEvent := client.NewStandaloneDurableTask("evictable-wait-for-event",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			if _, err := ctx.WaitForEvent(eventKey, "true"); err != nil {
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	// > Non Evictable Sleep
	nonEvictablePolicy := &hatchet.EvictionPolicy{
		AllowCapacityEviction: false,
		Priority:              0,
	}

	nonEvictableSleep := client.NewStandaloneDurableTask("non-evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			if _, err := ctx.SleepFor(10 * time.Second); err != nil {
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(nonEvictablePolicy),
	)

	// > Capacity Evictable Sleep
	capacityEvictionPolicy := &hatchet.EvictionPolicy{
		AllowCapacityEviction: true,
		Priority:              0,
	}

	capacityEvictableSleep := client.NewStandaloneDurableTask("capacity-evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			if _, err := ctx.SleepFor(20 * time.Second); err != nil {
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(capacityEvictionPolicy),
	)

	// > Eviction vs Cancellation
	evictionVsCancellation := client.NewStandaloneDurableTask("eviction-vs-cancellation",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			if _, err := ctx.SleepFor(longSleepSeconds * time.Second); err != nil {
				switch {
				case errors.Is(context.Cause(ctx), hatchet.ErrTaskCancelled):
					// Run was cancelled; clean up external resources or child runs.
					log.Println("task cancelled, running cleanup")
				case errors.Is(context.Cause(ctx), hatchet.ErrTaskEvicted):
					// Evicted to free the worker slot; the run will resume later.
					log.Println("task evicted, skipping cleanup")
				}
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)


	// > Cancel Child With Trigger Context
	cancelChildOnParentCancel := client.NewStandaloneDurableTask("cancel-child-on-parent-cancel",
		func(ctx hatchet.DurableContext, input EmptyInput) (EvictionOutput, error) {
			ackCtx, cancelAck := context.WithTimeout(context.WithoutCancel(ctx.GetContext()), 10*time.Second)
			defer cancelAck()

			ref, err := evictableSleep.RunNoWait(ctx, EmptyInput{}, hatchet.WithTriggerContext(ackCtx))
			if err != nil {
				return EvictionOutput{}, err
			}

			if _, err := ref.Result(); err != nil {
				if errors.Is(context.Cause(ctx), hatchet.ErrTaskCancelled) {
					// Run was cancelled (not evicted); cancel the child with a detached context.
					cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancelCleanup()
					runID, parseErr := uuid.Parse(ref.RunId)
					if parseErr == nil {
						_, _ = client.Runs().Cancel(cleanupCtx, rest.V1CancelTaskRequest{
							ExternalIds: &[]uuid.UUID{runID},
						})
					}
				}
				return EvictionOutput{}, err
			}
			return EvictionOutput{Status: "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	worker, err := client.NewWorker("eviction-worker",
		hatchet.WithWorkflows(
			evictableSleep,
			evictableWaitForEvent,
			nonEvictableSleep,
			capacityEvictableSleep,
			evictionVsCancellation,
			cancelChildOnParentCancel,
		),
		hatchet.WithDurableSlots(10),
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
