package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet-go-quickstart/client"
	"github.com/hatchet-dev/hatchet-go-quickstart/workflows"
)

func main() {
	c, err := client.HatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	simple := workflows.FirstWorkflow(c)

	result, err := simple.Run(context.Background(), workflows.SimpleInput{
		Message: "HELLO, WORLD!",
	})
	if err != nil {
		log.Fatalf("Failed to run Hatchet task: %v", err)
	}

	var simpleResult workflows.SimpleOutput

	err = result.Into(&simpleResult)
	if err != nil {
		log.Fatalf("Failed to convert result to SimpleOutput: %v", err)
	}

	fmt.Println(simpleResult.TransformedMessage)
}
