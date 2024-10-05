package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	_, err = run()
	if err != nil {
		panic(err)
	}

}

func run() (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	err = testSvc.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events("user:create:bulk"),
			Name:        "bulk",
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &userCreateEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-one")

					return &stepOneOutput{
						Message: "Username is: " + input.Username,
					}, nil
				},
				),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	var events []client.EventWithAdditionalMetadata

	// 20000 times to test the bulk push

	for i := 0; i < 20000; i++ {
		testEvent := userCreateEvent{
			Username: "echo-test",
			UserID:   "1234 " + fmt.Sprint(i),
			Data: map[string]string{
				"test": "test " + fmt.Sprint(i),
			},
		}
		events = append(events, client.EventWithAdditionalMetadata{
			Event:              testEvent,
			AdditionalMetadata: map[string]string{"hello": "world " + fmt.Sprint(i)},
			Key:                "user:create:bulk",
		})
	}

	log.Printf("pushing event user:create:bulk")

	err = c.Event().BulkPush(
		context.Background(),
		events,
	)
	if err != nil {
		panic(fmt.Errorf("error pushing event: %w", err))
	}

	return nil, nil

}
