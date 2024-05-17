package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
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

type StepRunNotifyCancelTaskPayload struct {
	StepRunId       string `json:"step_run_id" validate:"required,uuid"`
	CancelledReason string `json:"cancelled_reason" validate:"required"`
}

type StepRunNotifyCancelTaskMetadata struct {
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

type StepRunStreamEventTaskPayload struct {
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	CreatedAt     string `json:"created_at" validate:"required"`
	StreamEventId string `json:"stream_event_id"`
}

type StepRunStreamEventTaskMetadata struct {
	TenantId      string `json:"tenant_id" validate:"required,uuid"`
	StreamEventId string `json:"stream_event_id" validate:"required,integer"`
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
}

type StepRunTimedOutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunRetryTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`

	// optional - if not provided, the step run will be retried with the same input
	InputData string `json:"input_data,omitempty"`
}

type StepRunRetryTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunReplayTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`

	// optional - if not provided, the step run will be retried with the same input
	InputData string `json:"input_data,omitempty"`
}

type StepRunReplayTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func TenantToStepRunRequeueTask(tenant db.TenantModel) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(StepRunRequeueTaskPayload{
		TenantId: tenant.ID,
	})

	metadata, _ := datautils.ToJSONMap(StepRunRequeueTaskMetadata{
		TenantId: tenant.ID,
	})

	return &msgqueue.Message{
		ID:       "step-run-requeue-ticker",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunRetryToTask(stepRun *dbsqlc.GetStepRunForEngineRow, inputData []byte) *msgqueue.Message {
	jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId)

	payload, _ := datautils.ToJSONMap(StepRunRetryTaskPayload{
		JobRunId:  jobRunId,
		StepRunId: stepRunId,
		InputData: string(inputData),
	})

	metadata, _ := datautils.ToJSONMap(StepRunRetryTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-retry",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunReplayToTask(stepRun *dbsqlc.GetStepRunForEngineRow, inputData []byte) *msgqueue.Message {
	jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId)

	payload, _ := datautils.ToJSONMap(StepRunReplayTaskPayload{
		JobRunId:  jobRunId,
		StepRunId: stepRunId,
		InputData: string(inputData),
	})

	metadata, _ := datautils.ToJSONMap(StepRunReplayTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-replay",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunCancelToTask(stepRun *dbsqlc.GetStepRunForEngineRow, reason string) *msgqueue.Message {
	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId)

	payload, _ := datautils.ToJSONMap(StepRunNotifyCancelTaskPayload{
		StepRunId:       stepRunId,
		CancelledReason: reason,
	})

	metadata, _ := datautils.ToJSONMap(StepRunNotifyCancelTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-cancelled",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunQueuedToTask(stepRun *dbsqlc.GetStepRunForEngineRow) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(StepRunTaskPayload{
		JobRunId:  sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepRunId: sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
	})

	metadata, _ := datautils.ToJSONMap(StepRunTaskMetadata{
		StepId:            sqlchelpers.UUIDToStr(stepRun.StepId),
		ActionId:          stepRun.ActionId,
		JobName:           stepRun.JobName,
		JobId:             sqlchelpers.UUIDToStr(stepRun.JobId),
		WorkflowVersionId: sqlchelpers.UUIDToStr(stepRun.WorkflowVersionId),
		TenantId:          sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId),
	})

	return &msgqueue.Message{
		ID:       "step-run-queued",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
