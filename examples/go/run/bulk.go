package main

import (
	"context"
	"fmt"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
)

func bulk() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// > Bulk Run Tasks
	simple := v1_workflows.Simple(hatchet)
	bulkRunIds, err := simple.RunBulkNoWait(ctx, []v1_workflows.SimpleInput{
		{
			Message: "Hello, World!",
		},
		{
			Message: "Hello, Moon!",
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(bulkRunIds)
}
