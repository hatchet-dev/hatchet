package main

import (
	"context"
	"log"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type Input struct {
	Message string `json:"message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Run with metadata
	_, err = client.Run(
		context.Background(),
		"my-workflow",
		Input{Message: "hello"},
		hatchet.WithRunMetadata(
			map[string]string{"version": "1.0.0"},
		),
	)
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	// > Push an event with metadata
	err = client.Events().Push(
		context.Background(),
		"user:create",
		Input{Message: "hello"},
		v0Client.WithEventMetadata(
			map[string]string{"version": "1.0.0"},
		),
	)
	if err != nil {
		log.Fatalf("failed to push event: %v", err)
	}
}
