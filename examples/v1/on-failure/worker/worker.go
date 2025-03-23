package main

import (
	wf "github.com/hatchet-dev/hatchet/examples/v1/on-failure/workflow"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}

	onFailure := wf.Workflow(&hatchet)

	worker, err := hatchet.Worker(
		worker.CreateOpts{
			Name: "on-failure-worker",
		},
		worker.WithWorkflows(onFailure),
	)

	if err != nil {
		panic(err)
	}

	err = worker.StartBlocking()

	if err != nil {
		panic(err)
	}
}
