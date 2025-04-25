package main

import (
	"context"
	"fmt"

	hatchet_client "hatchet-go-quickstart/hatchet_client"
	workflows "hatchet-go-quickstart/workflows"
)

func main() {
	hatchet, err := hatchet_client.HatchetClient()

	if err != nil {
		panic(err)
	}

	simple := workflows.FirstTask(hatchet)

	result, err := simple.Run(context.Background(), workflows.SimpleInput{
		Message: "Hello, World!",
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(
		"Finished running task, and got the transformed message! The transformed message is:",
		result.ToLower.TransformedMessage,
	)
}
