package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

const ingestEventKey = "ingest:create"

type ingestEvent struct {
	Message string `json:"Message"`
}

type ingestOutput struct {
}

const storeEventKey = "store:create"

type storeEvent struct {
	Message string `json:"Message"`
}

type storeOutput struct {
}

const parseEventKey = "parse:create"

type parseEvent struct {
	Message string `json:"Message"`
}

type parseOutput struct {
}

func main() {
	interrupt := cmdutils.InterruptChan()

	cleanup, err := run()
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run() (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithName(os.Getenv("SERVICE")),
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(300),
	)

	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	if os.Getenv("SERVICE") == "ingest" {

		err = w.RegisterWorkflow(
			&worker.WorkflowJob{
				On:          worker.Events(ingestEventKey),
				Name:        "ingest",
				Description: "parent workflow",
				Steps: []*worker.WorkflowStep{
					worker.Fn(func(ctx worker.HatchetContext) (result *ingestOutput, err error) {
						input := &ingestEvent{}

						err = ctx.WorkflowInput(input)

						if err != nil {
							return nil, err
						}

						fmt.Println(input.Message)

						err = c.Event().Push(ctx.GetContext(), storeEventKey, &storeEvent{
							Message: input.Message,
						})

						if err != nil {
							return nil, err
						}

						err = c.Event().Push(ctx.GetContext(), parseEventKey, &parseEvent{
							Message: input.Message,
						})

						if err != nil {
							return nil, err
						}

						return &ingestOutput{}, nil
					},
					).SetName("ingest"),
				},
			},
		)

		if err != nil {
			return nil, fmt.Errorf("error registering workflow: %w", err)
		}
	}

	if os.Getenv("SERVICE") == "store" {

		err = w.RegisterWorkflow(
			&worker.WorkflowJob{
				On:          worker.Events(storeEventKey),
				Name:        "store",
				Description: "store workflow",
				Steps: []*worker.WorkflowStep{
					worker.Fn(func(ctx worker.HatchetContext) (result *storeOutput, err error) {
						input := &storeEvent{}

						err = ctx.WorkflowInput(input)

						if err != nil {
							return nil, err
						}

						fmt.Println(input.Message)

						return &storeOutput{}, nil
					},
					).SetName("store"),
				},
			},
		)

		if err != nil {
			return nil, fmt.Errorf("error registering workflow: %w", err)
		}
	}

	if os.Getenv("SERVICE") == "parse" {
		err = w.RegisterWorkflow(
			&worker.WorkflowJob{
				On:          worker.Events(parseEventKey, "parse:create:two"),
				Name:        "parse",
				Description: "parse workflow",
				Steps: []*worker.WorkflowStep{
					worker.Fn(func(ctx worker.HatchetContext) (result *parseOutput, err error) {
						input := &parseEvent{}

						err = ctx.WorkflowInput(input)

						if err != nil {
							return nil, err
						}

						fmt.Println(input.Message)

						time.Sleep(time.Second * 1)

						return &parseOutput{}, nil
					},
					).SetName("parse"),
				},
			},
		)

		if err != nil {
			return nil, fmt.Errorf("error registering workflow: %w", err)
		}
	}

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
	}

	return cleanup, nil
}
