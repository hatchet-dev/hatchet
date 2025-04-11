package main

import (
	"context"
	"fmt"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/v1/workflows"
	"github.com/hatchet-dev/hatchet/pkg/client"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/joho/godotenv"
)

func priority() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// ❓ Bulk Run Tasks
	priorityWorkflow := v1_workflows.Priority(hatchet)

	priority := int32(3)
	runOpts := workflow.RunOpts{
	}

	runId, err := priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{
		UserId: "1234",
	}, runOpts)

	if err != nil {
		panic(err)
	}

	fmt.Println(runId)
	// !!
}
