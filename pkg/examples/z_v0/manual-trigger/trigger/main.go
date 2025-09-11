package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 50)
	if err := run(cmdutils.InterruptChan(), events); err != nil {
		panic(err)
	}
}

func run(ch <-chan interface{}, events chan<- string) error {
	c, err := client.New()

	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	time.Sleep(1 * time.Second)

	// trigger workflow
	workflow, err := c.Admin().RunWorkflow(
		"post-user-update",
		&userCreateEvent{
			Username: "echo-test",
			UserID:   "1234",
			Data: map[string]string{
				"test": "test",
			},
		},
		client.WithRunMetadata(map[string]interface{}{
			"hello": "world",
		}),
	)

	if err != nil {
		return fmt.Errorf("error running workflow: %w", err)
	}

	fmt.Println("workflow run id:", workflow.WorkflowRunId())

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)
	defer cancel()

	err = c.Subscribe().On(interruptCtx, workflow.WorkflowRunId(), func(event client.WorkflowEvent) error {
		fmt.Println(event.EventPayload)

		return nil
	})

	return err
}
