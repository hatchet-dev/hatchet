package main

import (
	"log"
	"time"

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
	// !!

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
	// !!

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
	// !!

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
	// !!

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
	// !!

	worker, err := client.NewWorker("eviction-worker",
		hatchet.WithWorkflows(
			evictableSleep,
			evictableWaitForEvent,
			nonEvictableSleep,
			capacityEvictableSleep,
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
