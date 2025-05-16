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

type sampleEvent struct{}

type requeueInput struct{}

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

	err = worker.RegisterAction("requeue:requeue", func(ctx context.Context, input *requeueInput) (result any, err error) {
		return map[string]interface{}{}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	event := sampleEvent{}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"example:event",
		event,
		nil,
		nil,
	)

	if err != nil {
		panic(err)
	}

	// wait to register the worker for 10 seconds, to let the requeuer kick in
	time.Sleep(10 * time.Second)
	cleanup, err := worker.Start()
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-interruptCtx.Done():
			if err := cleanup(); err != nil {
				panic(fmt.Errorf("error cleaning up: %w", err))
			}
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
