package tasktypes

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type ReplayWorkflowRunTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
}

type ReplayWorkflowRunTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func WorkflowRunReplayToTask(tenantId, workflowRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(ReplayWorkflowRunTaskPayload{
		WorkflowRunId: workflowRunId,
	})

	metadata, _ := datautils.ToJSONMap(ReplayWorkflowRunTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "replay-workflow-run",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

type WorkflowRunQueuedTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
}

type WorkflowRunQueuedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type WorkflowRunFinishedTask struct {
	WorkflowRunId      string                 `json:"workflow_run_id" validate:"required,uuid"`
	Status             string                 `json:"status" validate:"required"`
	AdditionalMetadata map[string]interface{} `json:"additional_metadata"`
}

type WorkflowRunFinishedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func WorkflowRunFinishedToTask(tenantId, workflowRunId, status string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(WorkflowRunFinishedTask{
		WorkflowRunId: workflowRunId,
		Status:        status,
	})

	metadata, _ := datautils.ToJSONMap(WorkflowRunFinishedTaskMetadata{
		TenantId: tenantId,
	})

	fmt.Println(time.Now().String(), "| Sending workflow run finished message to queue | ", workflowRunId)

	return &msgqueue.Message{
		ID:       "workflow-run-finished",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func WorkflowRunQueuedToTask(tenantId, workflowRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(WorkflowRunQueuedTaskPayload{
		WorkflowRunId: workflowRunId,
	})

	metadata, _ := datautils.ToJSONMap(WorkflowRunQueuedTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "workflow-run-queued",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
