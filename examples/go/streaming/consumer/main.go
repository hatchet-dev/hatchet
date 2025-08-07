package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

// > Consume
func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	ctx := context.Background()

	streamingWorkflow := shared.StreamingWorkflow(hatchet)

	workflowRun, err := streamingWorkflow.RunNoWait(ctx, shared.StreamTaskInput{})
	if err != nil {
		log.Fatalf("Failed to run workflow: %v", err)
	}

	id := workflowRun.RunId()
	stream, err := hatchet.Runs().SubscribeToStream(ctx, id)
	if err != nil {
		log.Fatalf("Failed to subscribe to stream: %v", err)
	}

	for content := range stream {
		fmt.Print(content)
	}

	fmt.Println("\nStreaming completed!")
}

