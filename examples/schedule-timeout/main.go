package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type sampleEvent struct{}

type timeoutInput struct{}

func main() {
	client, err := client.New(
		client.InitWorkflows(),
	)

	if err != nil {
		panic(err)
	}

	event := sampleEvent{}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		event,
	)

	if err != nil {
		panic(err)
	}

	time.Sleep(35 * time.Second)

	fmt.Println("step should have timed out")
}
