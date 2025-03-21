package main

import (
	simpleWorkflow "github.com/hatchet-dev/hatchet/examples/v1/simple/workflow"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	v1Worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"
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

	simple, err := simpleWorkflow.SimpleWorkflow(&hatchet)

	if err != nil {
		panic(err)
	}

	worker, err := hatchet.Worker(
		v1Worker.CreateOpts{
			Name: "simple-worker",
		},
		v1Worker.WithWorkflows(simple),
	)

	if err != nil {
		panic(err)
	}

	err = worker.Start()

	if err != nil {
		panic(err)
	}
}
