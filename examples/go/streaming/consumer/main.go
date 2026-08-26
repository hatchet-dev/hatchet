package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Consume
func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	ctx := context.Background()

	streamingWorkflow := shared.StreamingWorkflow(client)

	workflowRun, err := streamingWorkflow.RunNoWait(ctx, shared.StreamTaskInput{})
	if err != nil {
		log.Fatalf("Failed to run workflow: %v", err)
	}

	id := workflowRun.RunId
	stream := client.Runs().SubscribeToStream(ctx, id)

	for content := range stream {
		fmt.Print(content)
	}

	fmt.Println("\nStreaming completed!")
}
