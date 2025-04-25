package main

import (
	"fmt"
	"os"
	"time"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
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

	// Get workflow name from command line arguments
	var workflowName string
	if len(os.Args) > 1 {
		workflowName = os.Args[1]
		fmt.Println("workflow name provided:", workflowName)
	}

	// Define workflows map
	workflowMap := map[string][]workflow.WorkflowBase{
		"dag":           {v1_workflows.DagWorkflow(hatchet)},
		"on-failure":    {v1_workflows.OnFailure(hatchet)},
		"simple":        {v1_workflows.Simple(hatchet)},
		"sleep":         {v1_workflows.DurableSleep(hatchet)},
		"child":         {v1_workflows.Parent(hatchet), v1_workflows.Child(hatchet)},
		"cancellation":  {v1_workflows.Cancellation(hatchet)},
		"timeout":       {v1_workflows.Timeout(hatchet)},
		"sticky":        {v1_workflows.Sticky(hatchet), v1_workflows.StickyDag(hatchet), v1_workflows.Child(hatchet)},
		"retries":       {v1_workflows.Retries(hatchet), v1_workflows.RetriesWithCount(hatchet), v1_workflows.WithBackoff(hatchet)},
		"on-cron":       {v1_workflows.OnCron(hatchet)},
		"non-retryable": {v1_workflows.NonRetryableError(hatchet)},
		"priority":      {v1_workflows.Priority(hatchet)},
	}

	// Add an "all" option that registers all workflows
	allWorkflows := []workflow.WorkflowBase{}
	for _, wfs := range workflowMap {
		allWorkflows = append(allWorkflows, wfs...)
	}
	workflowMap["all"] = allWorkflows

	// Lookup workflow from map
	workflow, ok := workflowMap[workflowName]
	if !ok {
		fmt.Println("Invalid workflow name provided. Usage: go run examples/v1/worker/start.go [workflow-name]")
		fmt.Println("Available workflows:", getAvailableWorkflows(workflowMap))
		os.Exit(1)
	}

	var slots int
	if workflowName == "priority" {
		slots = 1
	} else {
		slots = 100
	}

	worker, err := hatchet.Worker(
		worker.WorkerOpts{
			Name:      fmt.Sprintf("%s-worker", workflowName),
			Workflows: workflow,
			Slots:     slots,
		},
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()

	err = worker.StartBlocking(interruptCtx)

	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(10 * time.Second)
		cancel()
	}()
}

// Helper function to get available workflows as a formatted string
func getAvailableWorkflows(workflowMap map[string][]workflow.WorkflowBase) string {
	var workflows string
	count := 0
	for name := range workflowMap {
		if count > 0 {
			workflows += ", "
		}
		workflows += fmt.Sprintf("'%s'", name)
		count++
	}
	return workflows
}
