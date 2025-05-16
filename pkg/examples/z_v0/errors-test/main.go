package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/errors/sentry"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserId   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx context.Context) (result *stepOneOutput, err error) {
	return nil, fmt.Errorf("this is an error")
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	client, err := client.New()

	if err != nil {
		panic(err)
	}

	sentryAlerter, err := sentry.NewSentryAlerter(&sentry.SentryAlerterOpts{
		DSN:         os.Getenv("SENTRY_DSN"),
		Environment: os.Getenv("SENTRY_ENVIRONMENT"),
	})

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
		worker.WithErrorAlerter(sentryAlerter),
	)

	if err != nil {
		panic(err)
	}

	err = w.On(worker.Event("user:create"), &worker.WorkflowJob{
		Name:        "failing-workflow",
		Description: "This is a failing workflow.",
		Steps: []*worker.WorkflowStep{
			{
				Function: StepOne,
			},
		},
	})

	if err != nil {
		panic(err)
	}

	// err = worker.RegisterAction("echo:echo", func(ctx context.Context, input *actionInput) (result any, err error) {
	// 	return map[string]interface{}{
	// 		"message": input.Message,
	// 	}, nil
	// })

	// if err != nil {
	// 	panic(err)
	// }

	// err = worker.RegisterAction("echo:object", func(ctx context.Context, input *actionInput) (result any, err error) {
	// 	return nil, nil
	// })

	// if err != nil {
	// 	panic(err)
	// }

	ch := cmdutils.InterruptChan()

	cleanup, err := w.Start()
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

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		testEvent,
	)

	if err != nil {
		panic(err)
	}

	<-ch

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}
