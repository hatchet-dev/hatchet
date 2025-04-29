package main

import (
	"context"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
)

func event() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}
	// > Pushing an Event
	err = hatchet.Events().Push(
		context.Background(),
		"simple-event:create",
		v1_workflows.SimpleInput{
			Message: "Hello, World!",
		},
	)
	

	if err != nil {
		panic(err)
	}
}
