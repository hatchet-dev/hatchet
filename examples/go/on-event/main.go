package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type EventInput struct {
	Message string
}

type LowerTaskOutput struct {
	TransformedMessage string
}

type UpperTaskOutput struct {
	TransformedMessage string
}

// > Run workflow on event
const SimpleEvent = "simple-event:create"

func Lower(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask(
		"lower", func(ctx hatchet.Context, input EventInput) (*LowerTaskOutput, error) {
			return &LowerTaskOutput{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
		hatchet.WithWorkflowEvents(SimpleEvent),
	)
}

// > Accessing the filter payload
func accessFilterPayload(ctx hatchet.Context, input EventInput) (*LowerTaskOutput, error) {
	fmt.Println(ctx.FilterPayload())
	return &LowerTaskOutput{
		TransformedMessage: strings.ToLower(input.Message),
	}, nil
}

// > Declare with filter
func LowerWithFilter(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask(
		"lower", accessFilterPayload,
		hatchet.WithWorkflowEvents(SimpleEvent),
		hatchet.WithFilters(types.DefaultFilter{
			Expression: "true",
			Scope:      "example-scope",
			Payload: map[string]interface{}{
				"main_character":       "Anna",
				"supporting_character": "Stiva",
				"location":             "Moscow"},
		}),
	)
}

func Upper(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask(
		"upper", func(ctx hatchet.Context, input EventInput) (*UpperTaskOutput, error) {
			return &UpperTaskOutput{
				TransformedMessage: strings.ToUpper(input.Message),
			}, nil
		},
		hatchet.WithWorkflowEvents(SimpleEvent),
	)
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	_ = func() error {
		// > Pushing an event
		err := client.Events().Push(
			context.Background(),
			"simple-event:create",
			EventInput{
				Message: "Hello, World!",
			},
		)
		if err != nil {
			return err
		}

		// > Create a filter
		_, err = client.Filters().Create(
			context.Background(),
			rest.V1CreateFilterRequest{
				WorkflowId: uuid.MustParse("bb866b59-5a86-451b-8023-10d451db11d3"),
				Expression: "true",
				Scope:      "example-scope",
			},
		)
		if err != nil {
			return err
		}

		// > Skip a run
		skipPayload := map[string]interface{}{
			"shouldSkip": true,
		}
		skipScope := "foobarbaz"
		err = client.Events().Push(
			context.Background(),
			"simple-event:create",
			skipPayload,
			v0Client.WithFilterScope(&skipScope),
		)
		if err != nil {
			return err
		}

		// > Trigger a run
		triggerPayload := map[string]interface{}{
			"shouldSkip": false,
		}
		triggerScope := "foobarbaz"
		err = client.Events().Push(
			context.Background(),
			"simple-event:create",
			triggerPayload,
			v0Client.WithFilterScope(&triggerScope),
		)
		if err != nil {
			return err
		}

		return nil
	}

	worker, err := client.NewWorker("on-event-worker",
		hatchet.WithWorkflows(Lower(client), Upper(client), LowerWithFilter(client)),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
