package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type EmptyInput struct{}

type AffinityResult struct {
	WorkerID string `json:"worker_id"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	affinityTask := client.NewStandaloneTask("affinity-example-task",
		func(ctx hatchet.Context, input EmptyInput) (*AffinityResult, error) {
			return &AffinityResult{WorkerID: ctx.Worker().ID()}, nil
		},
	)

	labels := []string{"foo", "bar"}

	// Start two workers with different labels
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	for _, label := range labels {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			w, err := client.NewWorker("runtime-affinity-worker",
				hatchet.WithWorkflows(affinityTask),
				hatchet.WithLabels(map[string]any{"affinity": l}),
			)
			if err != nil {
				log.Fatalf("failed to create worker: %v", err)
			}
			if err := w.StartBlocking(ctx); err != nil {
				log.Printf("worker %s stopped: %v", l, err)
			}
		}(label)
	}

	// Wait for workers to register
	time.Sleep(5 * time.Second)

	// List workers and build label-to-id map
	workerList, err := client.Workers().List(ctx)
	if err != nil {
		log.Fatalf("failed to list workers: %v", err)
	}

	workerLabelToID := make(map[string]string)
	for _, w := range *workerList.Rows {
		if w.Name == "runtime-affinity-worker" && w.Status != nil && *w.Status == rest.ACTIVE {
			for _, label := range *w.Labels {
				if label.Key == "affinity" && label.Value != nil {
					for _, l := range labels {
						if *label.Value == l {
							workerLabelToID[l] = w.Metadata.Id
						}
					}
				}
			}
		}
	}

	if len(workerLabelToID) != 2 {
		log.Fatalf("expected 2 workers with affinity labels, got %d", len(workerLabelToID))
	}

	// Run 20 tasks with random affinity labels
	for i := 0; i < 20; i++ {
		targetLabel := labels[rand.Intn(len(labels))]
		required := true
		result, err := affinityTask.Run(ctx, EmptyInput{},
			hatchet.WithDesiredWorkerLabels(map[string]*hatchet.DesiredWorkerLabel{
				"affinity": {
					Value:    targetLabel,
					Required: required,
				},
			}),
		)
		if err != nil {
			log.Fatalf("failed to run task: %v", err)
		}

		var output AffinityResult
		if err := result.Into(&output); err != nil {
			log.Fatalf("failed to parse result: %v", err)
		}

		expectedWorkerID := workerLabelToID[targetLabel]
		if output.WorkerID != expectedWorkerID {
			log.Fatalf("expected worker %s for label %s, got %s", expectedWorkerID, targetLabel, output.WorkerID)
		}

		fmt.Printf("Run %d: label=%s routed to correct worker %s\n", i+1, targetLabel, output.WorkerID)
	}

	fmt.Println("All 20 runs routed correctly!")
	cancel()
	wg.Wait()
}
