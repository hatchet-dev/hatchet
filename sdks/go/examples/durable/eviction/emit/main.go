package main

import (
	"context"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const eventKey = "durable-eviction:event"

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Events().Push(ctx, eventKey, map[string]any{}); err != nil {
		log.Fatalf("failed to push %s: %v", eventKey, err)
	}

	log.Printf("pushed event %s", eventKey)
}
