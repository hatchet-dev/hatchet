package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type StepRunTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`
}

type StepRunTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`

	StepId            string `json:"step_id" validate:"required,uuid"`
	ActionId          string `json:"action_id" validate:"required,actionId"`
	JobId             string `json:"job_id" validate:"required,uuid"`
	JobName           string `json:"job_name" validate:"required,hatchetName"`
	WorkflowVersionId string `json:"workflow_version_id" validate:"required,uuid"`
}

type StepRunAssignedTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	WorkerId  string `json:"worker_id" validate:"required,uuid"`
}

type StepRunAssignedTaskMetadata struct {
	TenantId     string `json:"tenant_id" validate:"required,uuid"`
	DispatcherId string `json:"dispatcher_id" validate:"required,uuid"`
}

type StepRunCancelledTaskPayload struct {
	StepRunId       string `json:"step_run_id" validate:"required,uuid"`
	WorkerId        string `json:"worker_id" validate:"required,uuid"`
	CancelledReason string `json:"cancelled_reason" validate:"required"`
}

type StepRunCancelledTaskMetadata struct {
	TenantId     string `json:"tenant_id" validate:"required,uuid"`
	DispatcherId string `json:"dispatcher_id" validate:"required,uuid"`
}

type StepRunRequeueTaskPayload struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunRequeueTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunStartedTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	StartedAt string `json:"started_at" validate:"required"`
}

type StepRunStartedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunFinishedTaskPayload struct {
	StepRunId      string `json:"step_run_id" validate:"required,uuid"`
	FinishedAt     string `json:"finished_at" validate:"required"`
	StepOutputData string `json:"step_output_data"`
}

type StepRunFinishedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunFailedTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	FailedAt  string `json:"failed_at" validate:"required"`
	Error     string `json:"error" validate:"required"`
}

type StepRunFailedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunTimedOutTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`
}

type StepRunTimedOutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func TenantToStepRunRequeueTask(tenant db.TenantModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(StepRunRequeueTaskPayload{
		TenantId: tenant.ID,
	})

	metadata, _ := datautils.ToJSONMap(StepRunRequeueTaskMetadata{
		TenantId: tenant.ID,
	})

	return &taskqueue.Task{
		ID:       "step-run-requeue-ticker",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}

func StepRunQueuedToTask(job *db.JobModel, stepRun *db.StepRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(StepRunTaskPayload{
		JobRunId:  stepRun.JobRunID,
		StepRunId: stepRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(StepRunTaskMetadata{
		StepId:            stepRun.StepID,
		ActionId:          stepRun.Step().ActionID,
		JobName:           job.Name,
		JobId:             job.ID,
		WorkflowVersionId: job.WorkflowVersionID,
		TenantId:          job.TenantID,
	})

	return &taskqueue.Task{
		ID:       "step-run-queued",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}
