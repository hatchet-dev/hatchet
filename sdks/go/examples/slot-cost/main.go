package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Slot cost
	omega := client.NewStandaloneTask("omega", func(ctx hatchet.Context, input any) (any, error) {
		log.Println("heavy work")
		return nil, nil
	}, hatchet.WithSlotCost(5))

	weenie := client.NewStandaloneTask("weenie", func(ctx hatchet.Context, input any) (any, error) {
		log.Println("light work")
		return nil, nil
	}, hatchet.WithSlotCost(1))
	// !!

	worker, err := client.NewWorker("slot-cost-worker",
		hatchet.WithWorkflows(omega, weenie),
		hatchet.WithSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting slot cost worker...")
	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
