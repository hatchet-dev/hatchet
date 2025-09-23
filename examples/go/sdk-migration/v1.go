package main

import (
	"log"
	"strings"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type SimpleResult struct {
	TransformedMessage string `json:"result"`
}

func V1() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	workflow := client.NewStandaloneTask("simple-workflow", func(ctx hatchet.Context, input SimpleInput) (SimpleResult, error) {
		log.Println("executed step1")
		return SimpleResult{TransformedMessage: strings.ToLower(input.Message)}, nil
	}, hatchet.WithWorkflowEvents("user:create"))

	_, err = client.NewWorker(
		"worker",
		hatchet.WithWorkflows(workflow),
	)
	if err != nil {
		log.Fatal(err)
	}
}
