package tasktypes

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type StepRunTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	JobRunId      string `json:"job_run_id" validate:"required,uuid"`
	StepRetries   *int32 `json:"step_retries,omitempty"`
	RetryCount    *int32 `json:"retry_count,omitempty"`
}

type StepRunTaskMetadata struct {
	TenantId          string `json:"tenant_id" validate:"required,uuid"`
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

type StepRunAssignedBulkTaskPayload struct {
	WorkerIdToStepRunIds map[string][]string `json:"worker_id_to_step_run_id" validate:"required"`
}

type StepRunAssignedBulkTaskMetadata struct {
	TenantId     string `json:"tenant_id" validate:"required,uuid"`
	DispatcherId string `json:"dispatcher_id" validate:"required,uuid"`
}

type StepRunCancelledTaskPayload struct {
	WorkflowRunId   string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId       string `json:"step_run_id" validate:"required,uuid"`
	WorkerId        string `json:"worker_id" validate:"required,uuid"`
	CancelledReason string `json:"cancelled_reason" validate:"required"`
	StepRetries     *int32 `json:"step_retries,omitempty"`
	RetryCount      *int32 `json:"retry_count,omitempty"`
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

type StepRunCancelTaskPayload struct {
	StepRunId           string `json:"step_run_id" validate:"required,uuid"`
	CancelledReason     string `json:"cancelled_reason" validate:"required"`
	StepRetries         *int32 `json:"step_retries,omitempty"`
	RetryCount          *int32 `json:"retry_count,omitempty"`
	PropagateToChildren bool   `json:"propagate_to_children"`
}

type StepRunCancelTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunStartedTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	StartedAt     string `json:"started_at" validate:"required"`
	StepRetries   *int32 `json:"step_retries,omitempty"`
	RetryCount    *int32 `json:"retry_count,omitempty"`
}

type StepRunStartedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunFinishedTaskPayload struct {
	WorkflowRunId  string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId      string `json:"step_run_id" validate:"required,uuid"`
	FinishedAt     string `json:"finished_at" validate:"required"`
	StepOutputData string `json:"step_output_data"`
	StepRetries    *int32 `json:"step_retries,omitempty"`
	RetryCount     *int32 `json:"retry_count,omitempty"`
}

type StepRunFinishedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunStreamEventTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	CreatedAt     string `json:"created_at" validate:"required"`
	StreamEventId string `json:"stream_event_id"`
	StepRetries   *int32 `json:"step_retries,omitempty"`
	RetryCount    *int32 `json:"retry_count,omitempty"`
}

type StepRunStreamEventTaskMetadata struct {
	TenantId      string `json:"tenant_id" validate:"required,uuid"`
	StreamEventId string `json:"stream_event_id" validate:"required,integer"`
}

type StepRunFailedTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	FailedAt      string `json:"failed_at" validate:"required"`
	Error         string `json:"error" validate:"required"`
	StepRetries   *int32 `json:"step_retries,omitempty"`
	RetryCount    *int32 `json:"retry_count,omitempty"`
}

type StepRunFailedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunTimedOutTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	StepRetries   *int32 `json:"step_retries,omitempty"`
	RetryCount    *int32 `json:"retry_count,omitempty"`
}

type StepRunTimedOutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunRetryTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	JobRunId      string `json:"job_run_id" validate:"required,uuid"`

	Error *string `json:"error,omitempty"`

	// optional - if not provided, the step run will be retried with the same input
	InputData string `json:"input_data,omitempty"`

	StepRetries *int32 `json:"step_retries,omitempty"`
	RetryCount  *int32 `json:"retry_count,omitempty"`
}

type StepRunRetryTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type StepRunReplayTaskPayload struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string `json:"step_run_id" validate:"required,uuid"`
	JobRunId      string `json:"job_run_id" validate:"required,uuid"`

	// optional - if not provided, the step run will be retried with the same input
	InputData   string `json:"input_data,omitempty"`
	StepRetries *int32 `json:"step_retries,omitempty"`
	RetryCount  *int32 `json:"retry_count,omitempty"`
}

type StepRunReplayTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func StepRunRetryToTask(stepRun *dbsqlc.GetStepRunForEngineRow, inputData []byte, err string) *msgqueue.Message {
	jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
	stepRunId := sqlchelpers.UUIDToStr(stepRun.SRID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.SRTenantId)
	workflowRunId := sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)

	payload, _ := datautils.ToJSONMap(StepRunRetryTaskPayload{
		WorkflowRunId: workflowRunId,
		JobRunId:      jobRunId,
		StepRunId:     stepRunId,
		Error:         &err,
		InputData:     string(inputData),
		StepRetries:   &stepRun.StepRetries,
		RetryCount:    &stepRun.SRRetryCount,
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
	stepRunId := sqlchelpers.UUIDToStr(stepRun.SRID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.SRTenantId)
	workflowRunId := sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)

	payload, _ := datautils.ToJSONMap(StepRunReplayTaskPayload{
		WorkflowRunId: workflowRunId,
		JobRunId:      jobRunId,
		StepRunId:     stepRunId,
		InputData:     string(inputData),
		StepRetries:   &stepRun.StepRetries,
		RetryCount:    &stepRun.SRRetryCount,
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

func StepRunFailedToTask(stepRun *dbsqlc.GetStepRunForEngineRow, errorReason string, failedAt *time.Time) *msgqueue.Message {
	stepRunId := sqlchelpers.UUIDToStr(stepRun.SRID)
	workflowRunId := sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)
	tenantId := sqlchelpers.UUIDToStr(stepRun.SRTenantId)

	payload, _ := datautils.ToJSONMap(StepRunFailedTaskPayload{
		WorkflowRunId: workflowRunId,
		StepRunId:     stepRunId,
		FailedAt:      failedAt.Format(time.RFC3339),
		Error:         errorReason,
		StepRetries:   &stepRun.StepRetries,
		RetryCount:    &stepRun.SRRetryCount,
	})

	metadata, _ := datautils.ToJSONMap(StepRunFailedTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-failed",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunCancelToTask(stepRun *dbsqlc.GetStepRunForEngineRow, reason string, propagateToChildren bool) *msgqueue.Message {
	stepRunId := sqlchelpers.UUIDToStr(stepRun.SRID)
	tenantId := sqlchelpers.UUIDToStr(stepRun.SRTenantId)

	payload, _ := datautils.ToJSONMap(StepRunCancelTaskPayload{
		StepRunId:           stepRunId,
		CancelledReason:     reason,
		StepRetries:         &stepRun.StepRetries,
		RetryCount:          &stepRun.SRRetryCount,
		PropagateToChildren: propagateToChildren,
	})

	metadata, _ := datautils.ToJSONMap(StepRunCancelTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-cancel",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func StepRunQueuedToTask(stepRun *dbsqlc.GetStepRunForEngineRow) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(StepRunTaskPayload{
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		StepRetries:   &stepRun.StepRetries,
		RetryCount:    &stepRun.SRRetryCount,
	})

	metadata, _ := datautils.ToJSONMap(StepRunTaskMetadata{
		StepId:            sqlchelpers.UUIDToStr(stepRun.StepId),
		ActionId:          stepRun.ActionId,
		JobName:           stepRun.JobName,
		JobId:             sqlchelpers.UUIDToStr(stepRun.JobId),
		WorkflowVersionId: sqlchelpers.UUIDToStr(stepRun.WorkflowVersionId),
		TenantId:          sqlchelpers.UUIDToStr(stepRun.SRTenantId),
	})

	return &msgqueue.Message{
		ID:       "step-run-queued",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
