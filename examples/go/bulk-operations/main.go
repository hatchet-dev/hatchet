package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

func main() {
	// > Setup

	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	ctx := context.Background()

	workflows, err := hatchet.Workflows().List(ctx, nil)
	if err != nil {
		log.Fatalf("failed to list workflows: %v", err)
	}

	if workflows == nil || workflows.Rows == nil || len(*workflows.Rows) == 0 {
		log.Fatalf("no workflows found")
	}

	selectedWorkflow := (*workflows.Rows)[0]
	selectedWorkflowUUID := uuid.MustParse(selectedWorkflow.Metadata.Id)


	// > List runs
	workflowRuns, err := hatchet.Runs().List(ctx, rest.V1WorkflowRunListParams{
		WorkflowIds: &[]types.UUID{selectedWorkflowUUID},
	})
	if err != nil || workflowRuns == nil || workflowRuns.JSON200 == nil || workflowRuns.JSON200.Rows == nil {
		log.Fatalf("failed to list workflow runs for workflow %s: %v", selectedWorkflow.Name, err)
	}

	var runIds []types.UUID

	for _, run := range workflowRuns.JSON200.Rows {
		runIds = append(runIds, uuid.MustParse(run.Metadata.Id))
	}


	// > Cancel by run ids
	_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		ExternalIds: &runIds,
	})
	if err != nil {
		log.Fatalf("failed to cancel runs by ids: %v", err)
	}


	// > Cancel by filters
	tNow := time.Now().UTC()

	_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		Filter: &rest.V1TaskFilter{
			Since:              tNow.Add(-24 * time.Hour),
			Until:              &tNow,
			Statuses:           &[]rest.V1TaskStatus{rest.V1TaskStatusRUNNING},
			WorkflowIds:        &[]types.UUID{selectedWorkflowUUID},
			AdditionalMetadata: &[]string{`{"key": "value"}`},
		},
	})
	if err != nil {
		log.Fatalf("failed to cancel runs by filters: %v", err)
	}


	fmt.Println("cancelled all runs for workflow", selectedWorkflow.Name)
}
