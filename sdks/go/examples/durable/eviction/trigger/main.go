package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type EmptyInput struct{}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	evictableSleep := client.NewStandaloneDurableTask("evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			return map[string]any{}, nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref, err := evictableSleep.RunNoWait(ctx, EmptyInput{})
	if err != nil {
		log.Fatalf("failed to trigger evictable-sleep: %v", err)
	}

	fmt.Printf("Triggered evictable_sleep: workflow_run_id=%s\n", ref.RunId)
}
