package main

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/examples/go/streaming/workflows"
)

func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	ctx := context.Background()

	// Create the streaming workflow
	streamingWorkflow := workflows.StreamingWorkflow(hatchet)

	// Run the streaming workflow
	workflowRun, err := streamingWorkflow.RunNoWait(ctx, workflows.StreamTaskInput{})
	if err != nil {
		log.Fatalf("Failed to run workflow: %v", err)
	}

	id := workflowRun.RunId()
	fmt.Printf("Started streaming workflow with ID: %s\n", id)

	// Subscribe to the stream using the V1 subscribeToStream method
	stream, err := hatchet.Runs().SubscribeToStream(ctx, id)
	if err != nil {
		log.Fatalf("Failed to subscribe to stream: %v", err)
	}

	fmt.Println("Streaming content:")
	for content := range stream {
		fmt.Print(content)
	}

	fmt.Println("\nStreaming completed!")
}