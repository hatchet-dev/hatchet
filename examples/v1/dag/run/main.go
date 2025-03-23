package main

import (
	wf "github.com/hatchet-dev/hatchet/examples/v1/dag/workflow"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
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

	dag := wf.Workflow(&hatchet)

	_, err = dag.Run(nil)

	if err != nil {
		panic(err)
	}

	// TODO bind result object
}
