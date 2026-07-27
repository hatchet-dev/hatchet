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

func main() {
	// > Setup
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	ctx := context.Background()

	workflows, err := client.Workflows().List(ctx, nil)
	if err != nil {
		log.Fatalf("failed to list workflows: %v", err)
	}

	if workflows.Rows == nil || len(*workflows.Rows) == 0 {
		log.Fatal("no workflows found")
	}

	workflow := (*workflows.Rows)[0]
	workflowId := uuid.MustParse(workflow.Metadata.Id)
	// !!

	// > List runs
	workflowRuns, err := client.Runs().List(ctx, rest.V1WorkflowRunListParams{
		Since:       time.Now().Add(-24 * time.Hour),
		WorkflowIds: &[]types.UUID{workflowId},
	})
	if err != nil {
		log.Fatalf("failed to list workflow runs: %v", err)
	}
	// !!

	// > Cancel by run ids
	runIds := make([]types.UUID, len(workflowRuns.Rows))
	for i, run := range workflowRuns.Rows {
		runIds[i] = uuid.MustParse(run.Metadata.Id)
	}

	// to replay runs by their ids, use `client.Runs().Replay` with a
	// `rest.V1ReplayTaskRequest` instead
	_, err = client.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		ExternalIds: &runIds,
	})
	if err != nil {
		log.Fatalf("failed to bulk cancel by run ids: %v", err)
	}
	// !!

	// > Cancel by filters
	until := time.Now()

	// to replay runs matching filters, use `client.Runs().Replay` with a
	// `rest.V1ReplayTaskRequest` instead
	_, err = client.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		Filter: &rest.V1TaskFilter{
			Since:              time.Now().Add(-24 * time.Hour),
			Until:              &until,
			Statuses:           &[]rest.V1TaskStatus{rest.V1TaskStatusRUNNING},
			WorkflowIds:        &[]types.UUID{workflowId},
			AdditionalMetadata: &[]string{"key:value"},
		},
	})
	if err != nil {
		log.Fatalf("failed to bulk cancel by filters: %v", err)
	}
	// !!
}
