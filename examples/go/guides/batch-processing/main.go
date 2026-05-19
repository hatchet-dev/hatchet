package main

import (
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

	// > Step 01 Define Parent Task
	parentTask := client.NewStandaloneDurableTask("spawn-children", func(ctx hatchet.DurableContext, input BatchInput) (map[string]interface{}, error) {
		inputs := make([]hatchet.RunManyOpt, len(input.Items))
		for i, itemID := range input.Items {
			inputs[i] = hatchet.RunManyOpt{Input: ItemInput{ItemID: itemID}}
		}
		runRefs, err := childTask.RunMany(ctx, inputs)
		if err != nil {
			return nil, err
		}
		results := make([]interface{}, len(runRefs))
		for i, ref := range runRefs {
			result, err := ref.Result()
			if err != nil {
				return nil, err
			}
			var parsed map[string]interface{}
			if err := result.TaskOutput("process-item").Into(&parsed); err != nil {
				return nil, err
			}
			results[i] = parsed
		}
		return map[string]interface{}{"processed": len(results), "results": results}, nil
	})

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
}
