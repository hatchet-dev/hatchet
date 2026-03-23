package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	type AffinityOutput struct {
		Worker string `json:"worker"`
	}

	// > AffinityWorkflow

	affinityWorkflow := client.NewWorkflow("affinity-workflow")

	_ = affinityWorkflow.NewTask("step", func(ctx hatchet.Context, _ any) (*AffinityOutput, error) {
		// > AffinityTask
		if ctx.Worker().GetLabels()["model"] != "fancy-ai-model-v2" {
			_ = ctx.Worker().UpsertLabels(map[string]interface{}{"model": "unset"})
			// DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
			_ = ctx.Worker().UpsertLabels(map[string]interface{}{"model": "fancy-ai-model-v2"})
		}

		return &AffinityOutput{Worker: ctx.Worker().ID()}, nil
	})

	_ = func() error {
		// > AffinityRun
		result, runErr := affinityWorkflow.RunNoWait(context.Background(), nil,
			hatchet.WithDesiredWorkerLabels(map[string]*hatchet.DesiredWorkerLabel{
				"model": {
					Value:  "fancy-ai-model-v2",
					Weight: 10,
				},
				"memory": {
					Value:      256,
					Required:   true,
					Comparator: types.ComparatorPtr(types.WorkerLabelComparator_LESS_THAN),
				},
			}),
		)
		if runErr != nil {
			return fmt.Errorf("failed to run workflow: %w", runErr)
		}

		fmt.Println(result.RunId)

		return nil
	}

	// > AffinityWorker
	worker, err := client.NewWorker("affinity-worker",
		hatchet.WithWorkflows(affinityWorkflow),
		hatchet.WithSlots(10),
		hatchet.WithLabels(map[string]any{
			"model":  "fancy-ai-model-v2",
			"memory": 512,
		}),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Printf("failed to start worker: %v\n", err)
	}
}
