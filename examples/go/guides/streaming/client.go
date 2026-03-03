package main

import (
	"context"
	"fmt"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Step 03 Subscribe Client
// Client triggers the task and subscribes to the stream.
func runAndSubscribe(client *hatchet.Client) {
	runRef, _ := client.RunNoWait(context.Background(), "stream-example", map[string]interface{}{})
	stream := client.Runs().SubscribeToStream(context.Background(), runRef.RunId)
	for chunk := range stream {
		fmt.Print(chunk)
	}
}

