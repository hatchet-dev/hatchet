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

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx context.Context, input *userCreateEvent) (result *stepOneOutput, err error) {
	// could get from context
	// testVal := ctx.Value("testkey").(string)
	// svcVal := ctx.Value("svckey").(string)

	return &stepOneOutput{
		Message: "Username is: " + input.Username,
	}, nil
}

func StepTwo(ctx context.Context, input *stepOneOutput) (result *stepOneOutput, err error) {
	return &stepOneOutput{
		Message: "Above message is: " + input.Message,
	}, nil
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

	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
	)

	if err != nil {
		panic(err)
	}

	testSvc := w.NewService("test")

	testSvc.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		ctx.SetContext(context.WithValue(ctx.GetContext(), "testkey", "testvalue"))
		return next(ctx)
	})

	err = testSvc.RegisterAction(StepOne, worker.WithActionName("step-one"))

	if err != nil {
		panic(err)
	}

	err = testSvc.RegisterAction(StepTwo, worker.WithActionName("step-two"))

	if err != nil {
		panic(err)
	}

	err = testSvc.On(
		worker.Events("user:create", "user:update"),
		&worker.WorkflowJob{
			Name:        "post-user-update",
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				// example of calling a registered action from the worker (includes service name)
				w.Call("test:step-one"),
				// example of calling a registered action from a service
				testSvc.Call("step-two"),
			},
		},
	)

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

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
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
		nil,
		nil,
	)

	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-interrupt:
			if err := cleanup(); err != nil {
				panic(fmt.Errorf("error cleaning up: %w", err))
			}
		default:
			time.Sleep(time.Second)
		}
	}
}
