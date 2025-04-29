package main

import (
	"context"
	"fmt"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
)

func cron() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}
	// > Create
	simple := v1_workflows.Simple(hatchet)

	ctx := context.Background()

	result, err := simple.Cron(
		ctx,
		"daily-run",
		"0 0 * * *",
		v1_workflows.SimpleInput{
			Message: "Hello, World!",
		},
	)

	if err != nil {
		panic(err)
	}

	// it may be useful to save the cron id for later
	fmt.Println(result.Metadata.Id)
	

	// > Delete
	hatchet.Crons().Delete(ctx, result.Metadata.Id)
	

	// > List
	crons, err := hatchet.Crons().List(ctx, rest.CronWorkflowListParams{
		AdditionalMetadata: &[]string{"user:daily-run"},
	})
	
	if err != nil {
		panic(err)
	}
	fmt.Println(crons)
}
