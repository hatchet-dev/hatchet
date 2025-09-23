package main

import (
	"context"
	"fmt"
	"log"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StubInput struct {
	Message string `json:"message"`
}

type StubOutput struct {
	Ok bool `json:"ok"`
}

func StubWorkflow(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("stub-workflow", func(ctx hatchet.Context, input StubInput) (StubOutput, error) {
		return StubOutput{
			Ok: true,
		}, nil
	})
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	task := StubWorkflow(client)

	// we are simply running the task here, but it can be implemented in another service / worker
	// and in another language with the same name and input-output types
	result, err := task.Run(context.Background(), StubInput{Message: "Hello, World!"})
	if err != nil {
		log.Fatalf("failed to run task: %v", err)
	}

	fmt.Println(result)
}
