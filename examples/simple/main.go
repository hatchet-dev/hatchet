package main

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserId   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type actionInput struct {
	Message string `json:"message"`
}

func main() {
	client, err := client.New(
		client.InitWorkflows(),
	)

	if err != nil {
		panic(err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
	worker, err := worker.NewWorker(
		worker.WithDispatcherClient(
			client.Dispatcher(),
		),
	)

	if err != nil {
		panic(err)
	}

	err = worker.RegisterAction("echo:echo", func(ctx context.Context, input *actionInput) (result any, err error) {
		return map[string]interface{}{
			"message": input.Message,
		}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContext(cmdutils.InterruptChan())
	defer cancel()

	go func() {
		err = worker.Start(interruptCtx)

		if err != nil {
			panic(err)
		}

		cancel()
	}()

	testEvent := userCreateEvent{
		Username: "echo-test",
		UserId:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		testEvent,
	)

	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-interruptCtx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
