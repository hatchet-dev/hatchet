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

type WorkflowRunFinishedTask struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	Status        string `json:"status" validate:"required"`
}

type WorkflowRunFinishedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func WorkflowRunFinishedToTask(tenantId, workflowRunId, status string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(WorkflowRunFinishedTask{
		WorkflowRunId: workflowRunId,
		Status:        status,
	})

	metadata, _ := datautils.ToJSONMap(WorkflowRunFinishedTaskMetadata{
		TenantId: tenantId,
	})

	return &taskqueue.Task{
		ID:       "workflow-run-finished",
		Payload:  payload,
		Metadata: metadata,
	}
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
		Payload:  payload,
		Metadata: metadata,
	}
}
