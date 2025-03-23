package main

import (
	v1_workflows "github.com/hatchet-dev/hatchet/examples/v1/workflows"
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

	simple := v1_workflows.Simple(&hatchet)

	worker, err := hatchet.Worker(
		worker.CreateOpts{
			Name: "simple-worker",
		},
		worker.WithWorkflows(simple),
	)

	if err != nil {
		panic(err)
	}

	err = worker.StartBlocking()

	if err != nil {
		panic(err)
	}
}
