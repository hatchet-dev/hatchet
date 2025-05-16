package main

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
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

type actionOut struct {
	Message string `json:"message"`
}

func echo(ctx context.Context, input *actionInput) (result *actionOut, err error) {
	return &actionOut{
		Message: input.Message,
	}, nil
}

func object(ctx context.Context, input *userCreateEvent) error {
	return nil
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	client, err := client.New(
		client.InitWorkflows(),
	)

	if err != nil {
		panic(err)
	}

	worker, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
	)

	if err != nil {
		panic(err)
	}

	echoSvc := worker.NewService("echo")

	err = echoSvc.RegisterAction(echo)

	if err != nil {
		panic(err)
	}

	err = echoSvc.RegisterAction(object)

	if err != nil {
		panic(err)
	}

	ch := cmdutils.InterruptChan()

	cleanup, err := worker.Start()
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	testEvent := userCreateEvent{
		Username: "echo-test",
		UserId:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}

	time.Sleep(1 * time.Second)

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		testEvent,
		nil,
		nil,
	)

	if err != nil {
		panic(err)
	}

	<-ch

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up worker: %w", err))
	}
}
