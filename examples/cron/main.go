package main

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type printInput struct{}

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

	err = worker.RegisterAction("print:print", func(ctx context.Context, input *printInput) (result any, err error) {
		fmt.Println("called print:print")

		return map[string]interface{}{}, nil
	})

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContext(cmdutils.InterruptChan())
	defer cancel()

	err = worker.Start(interruptCtx)

	if err != nil {
		panic(err)
	}
}
