package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type JobRunQueuedTaskPayload struct {
	JobRunId string `json:"job_run_id" validate:"required,uuid"`
}

type JobRunQueuedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`

	JobId             string `json:"job_id" validate:"required,uuid"`
	JobName           string `json:"job_name" validate:"required,hatchetName"`
	WorkflowVersionId string `json:"workflow_version_id" validate:"required,uuid"`
}

func JobRunQueuedToTask(job *db.JobModel, jobRun *db.JobRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(JobRunQueuedTaskPayload{
		JobRunId: jobRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(JobRunQueuedTaskMetadata{
		JobName:           job.Name,
		JobId:             job.ID,
		WorkflowVersionId: job.WorkflowVersionID,
		TenantId:          job.TenantID,
	})

	return &taskqueue.Task{
		ID:       "job-run-queued",
		Payload:  payload,
		Metadata: metadata,
	}
}

type JobRunTimedOutTaskPayload struct {
	JobRunId string `json:"job_run_id" validate:"required,uuid"`
}

type JobRunTimedOutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
