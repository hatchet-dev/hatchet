package main

import (
	"context"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type BatchInput struct {
	Items []string `json:"items"`
}

type ItemInput struct {
	ItemID string `json:"item_id"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 03 Process Item
	childTask := client.NewStandaloneTask("process-item", func(ctx hatchet.Context, input ItemInput) (map[string]string, error) {
		return map[string]string{"status": "done", "item_id": input.ItemID}, nil
	})
	// !!

	// > Step 01 Define Parent Task
	parentTask := client.NewStandaloneTask("spawn-children", func(ctx hatchet.Context, input BatchInput) (map[string]interface{}, error) {
		results := []interface{}{}
		for _, itemID := range input.Items {
			res, err := childTask.Run(ctx, ItemInput{ItemID: itemID})
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
		return map[string]interface{}{"processed": len(results), "results": results}, nil
	})
	// !!

	// > Step 02 Fan Out Children
	// Parent fans out: for each item, run child task. Hatchet distributes across workers.
	fanOut := func(ctx context.Context, input BatchInput, child *hatchet.StandaloneTask) ([]interface{}, error) {
		results := []interface{}{}
		for _, itemID := range input.Items {
			res, err := child.Run(ctx, ItemInput{ItemID: itemID})
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
		return results, nil
	}
	_ = fanOut
	// !!

	// > Step 04 Run Worker
	worker, err := client.NewWorker("batch-worker", hatchet.WithWorkflows(parentTask, childTask), hatchet.WithSlots(20))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		cancel()
		log.Fatalf("failed to start worker: %v", err)
	}
	// !!
}
