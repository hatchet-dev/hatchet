package main

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type sampleEvent struct{}

type requeueInput struct{}

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

	err = worker.RegisterAction("requeue:requeue", func(ctx context.Context, input *requeueInput) (result any, err error) {
		return map[string]interface{}{}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContext(cmdutils.InterruptChan())
	defer cancel()

	event := sampleEvent{}

	// push an event
	err = client.Event().Push(
		context.Background(),
		"example:event",
		event,
	)

	if err != nil {
		panic(err)
	}

	go func() {
		// wait to register the worker for 10 seconds, to let the requeuer kick in
		time.Sleep(10 * time.Second)

		err = worker.Start(interruptCtx)

		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
