package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func cancelExample() { //nolint:unused
	// > Setup
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	ctx := context.Background()

	workflows, err := client.Workflows().List(ctx, &rest.WorkflowListParams{})
	if err != nil {
		log.Fatalf("failed to list workflows: %v", err)
	}

	workflow := (*workflows.Rows)[0]

	// > List runs
	workflowId := uuid.MustParse(workflow.Metadata.Id)

	runs, err := client.Runs().List(ctx, rest.V1WorkflowRunListParams{
		WorkflowIds: &[]types.UUID{workflowId},
		Since:       time.Now().Add(-24 * time.Hour),
	})
	if err != nil {
		log.Fatalf("failed to list runs: %v", err)
	}

	// > Cancel by run ids
	runIds := make([]types.UUID, len(runs.Rows))
	for i, run := range runs.Rows {
		runIds[i] = run.TaskExternalId
	}

	_, err = client.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		ExternalIds: &runIds,
	})
	if err != nil {
		log.Fatalf("failed to cancel runs: %v", err)
	}

	// > Cancel by filters
	now := time.Now()

	_, err = client.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		Filter: &rest.V1TaskFilter{
			Since:              time.Now().Add(-24 * time.Hour),
			Until:              &now,
			Statuses:           &[]rest.V1TaskStatus{rest.V1TaskStatusRUNNING},
			WorkflowIds:        &[]types.UUID{workflowId},
			AdditionalMetadata: &[]string{"key:value"},
		},
	})
	if err != nil {
		log.Fatalf("failed to cancel runs by filters: %v", err)
	}
}
