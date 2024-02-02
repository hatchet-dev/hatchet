package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type WorkflowRunQueuedTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
}

type WorkflowRunQueuedTaskMetadata struct {
	TenantId          string `json:"tenant_id" validate:"required,uuid"`
	WorkflowVersionId string `json:"workflow_version_id" validate:"required,uuid"`
}

func WorkflowRunQueuedToTask(workflowRun *db.WorkflowRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(WorkflowRunQueuedTaskPayload{
		WorkflowRunId: workflowRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(WorkflowRunQueuedTaskMetadata{
		WorkflowVersionId: workflowRun.WorkflowVersionID,
		TenantId:          workflowRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "workflow-run-queued",
		Queue:    taskqueue.WORKFLOW_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}
