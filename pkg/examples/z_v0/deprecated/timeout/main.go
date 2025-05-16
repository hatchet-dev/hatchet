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

type timeoutInput struct{}

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

	err = worker.RegisterAction("timeout:timeout", func(ctx context.Context, input *timeoutInput) (result any, err error) {
		// wait for context done signal
		timeStart := time.Now().UTC()
		<-ctx.Done()
		fmt.Println("context cancelled in ", time.Since(timeStart).Seconds(), " seconds")

		return map[string]interface{}{}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	cleanup, err := worker.Start()
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	event := sampleEvent{}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		event,
		nil,
		nil,
	)

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
