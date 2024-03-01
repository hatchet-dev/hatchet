package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type printOutput struct{}

func print(ctx context.Context) (result *printOutput, err error) {
	fmt.Println("called print:print")

	return &printOutput{}, nil
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

	err = w.On(
		worker.Cron("* * * * *"),
		worker.Fn(print),
	)

	if err != nil {
		panic(err)
	}

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}
