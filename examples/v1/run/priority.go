package main

import (
	"context"
	"fmt"
	"time"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/v1/workflows"
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

	priorityWorkflow := v1_workflows.Priority(hatchet)

	// ❓ Running a Task with Priority
	priority := int32(3)
	runOpts := workflow.RunOpts{
		Priority: &priority,
	}

	runId, err := priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{
		UserId: "1234",
	}, runOpts)
	// !!

	if err != nil {
		panic(err)
	}

	fmt.Println(runId)

	// ❓ Schedule and cron
	schedulePriority := int32(3)
	scheduleOpts := workflow.RunOpts{
		Priority: &schedulePriority,
	}
	runAt := time.Now().Add(time.Minute)

	scheduledRunId, _ := priorityWorkflow.Schedule(ctx, runAt, v1_workflows.PriorityInput{
		UserId: "1234",
	}, scheduleOpts)

	cronId, _ := priorityWorkflow.Cron(ctx, "my-cron", "* * * * *", v1_workflows.PriorityInput{
		UserId: "1234",
	}, scheduleOpts)
	// !!

	fmt.Println(scheduledRunId)
	fmt.Println(cronId)

	// !!
}
