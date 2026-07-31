package main

import (
	"log"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// Preview: batch tasks are in beta and may change in future releases.

type SimpleInput struct {
	Message string `json:"message"`
}

type SimpleOutput struct {
	TransformedMessage string `json:"transformed_message"`
}

type KeyedInput struct {
	Message string `json:"message"`
	Group   string `json:"group"`
}

type KeyedOutput struct {
	BatchKey   string `json:"batch_key"`
	BatchSize  int    `json:"batch_size"`
	UniqueKeys int    `json:"unique_keys"`
	Uppercase  string `json:"uppercase"`
}

type BroadcastSumOutput struct {
	Sum int `json:"sum"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Declaring a batch task
	batchSimple := client.NewStandaloneBatchTask("batch-simple",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (map[string]SimpleOutput, error) {
			out := make(map[string]SimpleOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = SimpleOutput{TransformedMessage: strings.ToUpper(inp.Message)}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     3,
			MaxInterval: durationPtr(200 * time.Millisecond),
		},
	)
	// !!

	// > Declaring a keyed batch task
	batchKeyed := client.NewStandaloneBatchTask("batch-keyed",
		func(ctx hatchet.Context, tasks map[string]KeyedInput) (map[string]KeyedOutput, error) {
			uniqueGroups := make(map[string]struct{})
			for _, inp := range tasks {
				uniqueGroups[inp.Group] = struct{}{}
			}

			out := make(map[string]KeyedOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = KeyedOutput{
					BatchKey:   inp.Group,
					BatchSize:  len(tasks),
					UniqueKeys: len(uniqueGroups),
					Uppercase:  strings.ToUpper(inp.Message),
				}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     2,
			MaxInterval: durationPtr(200 * time.Millisecond),
			GroupKey:    stringPtr("input.group"),
		},
	)
	// !!

	// > Declaring a broadcast batch task
	batchBroadcast := client.NewStandaloneBatchTask("batch-broadcast",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (BroadcastSumOutput, error) {
			sum := 0
			for _, inp := range tasks {
				sum += len(inp.Message)
			}
			return BroadcastSumOutput{Sum: sum}, nil
		},
		hatchet.BatchConfig{
			MaxSize:         10,
			MaxInterval:     durationPtr(2 * time.Second),
			BroadcastOutput: true,
		},
	)
	// !!

	worker, err := client.NewWorker("batch-assign-worker",
		hatchet.WithWorkflows(batchSimple, batchKeyed, batchBroadcast),
		hatchet.WithSlots(25),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func stringPtr(s string) *string {
	return &s
}
