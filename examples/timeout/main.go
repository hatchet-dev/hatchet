package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type sampleEvent struct{}

type timeoutInput struct{}

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

	err = worker.RegisterAction("timeout:timeout", func(ctx context.Context, input *timeoutInput) (result any, err error) {
		// wait for context done signal
		timeStart := time.Now()
		<-ctx.Done()
		fmt.Println("context cancelled in ", time.Since(timeStart).Seconds(), " seconds")

		return map[string]interface{}{}, nil
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
	}()

	event := sampleEvent{}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"user:create",
		event,
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
