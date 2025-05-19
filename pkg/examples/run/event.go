package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

func event() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}
	// > Pushing an Event
	err = hatchet.Events().Push(
		context.Background(),
		"simple-event:create",
		v1_workflows.SimpleInput{
			Message: "Hello, World!",
		},
	)
	// !!

	// > Create a filter
	payload := map[string]interface{}{
		"main_character":       "Anna",
		"supporting_character": "Stiva",
		"location":             "Moscow",
	}

	_, err = hatchet.Filters().Create(
		context.Background(),
		rest.V1CreateFilterRequest{
			WorkflowId: uuid.New(),
			Expression: "input.shouldSkip == false",
			Scope:      "foobarbaz",
			Payload:    &payload,
		},
	)
	// !!

	if err != nil {
		panic(err)
	}

	// > Skip a run
	skipPayload := map[string]interface{}{
		"shouldSkip": true,
	}
	skipScope := "foobarbaz"
	err = hatchet.Events().Push(
		context.Background(),
		"simple-event:create",
		skipPayload,
		client.WithFilterScope(&skipScope),
	)
	// !!

	if err != nil {
		panic(err)
	}

	// > Trigger a run
	triggerPayload := map[string]interface{}{
		"shouldSkip": false,
	}
	triggerScope := "foobarbaz"
	err = hatchet.Events().Push(
		context.Background(),
		"simple-event:create",
		triggerPayload,
		client.WithFilterScope(&triggerScope),
	)
	// !!

	if err != nil {
		panic(err)
	}
}
