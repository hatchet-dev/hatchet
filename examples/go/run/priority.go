package main

import (
	"context"
	"fmt"
	"time"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	"github.com/hatchet-dev/hatchet/pkg/client"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
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

	priorityWorkflow := v1_workflows.Priority(hatchet)

	// > Running a Task with Priority
	priority := int32(3)

	runId, err := priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{
		UserId: "1234",
	}, client.WithPriority(priority))

	if err != nil {
		panic(err)
	}

	fmt.Println(runId)

	// > Schedule and cron
	schedulePriority := int32(3)
	runAt := time.Now().Add(time.Minute)

	scheduledRunId, _ := priorityWorkflow.Schedule(ctx, runAt, v1_workflows.PriorityInput{
		UserId: "1234",
	}, client.WithPriority(schedulePriority))

	cronId, _ := priorityWorkflow.Cron(ctx, "my-cron", "* * * * *", v1_workflows.PriorityInput{
		UserId: "1234",
	}, client.WithPriority(schedulePriority))

	fmt.Println(scheduledRunId)
	fmt.Println(cronId)

}
