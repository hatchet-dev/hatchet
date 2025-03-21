package main

import (
	simpleWorkflow "github.com/hatchet-dev/hatchet/examples/v1/simple/workflow"
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

	simple, err := simpleWorkflow.SimpleWorkflow(&hatchet)

	if err != nil {
		panic(err)
	}

	_, err = simple.Run(simpleWorkflow.Input{
		Message: "Hello, World!",
	})

	if err != nil {
		panic(err)
	}

	// fmt.Println(res.Reverse.TransformedMessage)
}
