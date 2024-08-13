package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type JobRunQueuedTaskPayload struct {
	JobRunId string `json:"job_run_id" validate:"required,uuid"`
}

type JobRunQueuedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func JobRunQueuedToTask(tenantId, jobRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(JobRunQueuedTaskPayload{
		JobRunId: jobRunId,
	})

	metadata, _ := datautils.ToJSONMap(JobRunQueuedTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "job-run-queued",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

type CheckTenantQueueMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func CheckTenantQueueToTask(tenantId string) *msgqueue.Message {
	metadata, _ := datautils.ToJSONMap(CheckTenantQueueMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "check-tenant-queue",
		Payload:  nil,
		Metadata: metadata,
		Retries:  3,
	}
}

type JobRunCancelledTaskPayload struct {
	JobRunId string  `json:"job_run_id" validate:"required,uuid"`
	Reason   *string `json:"reason,omitempty"`
}

type JobRunCancelledTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func JobRunCancelledToTask(tenantId, jobRunId string, reason *string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(JobRunCancelledTaskPayload{
		JobRunId: jobRunId,
		Reason:   reason,
	})

	metadata, _ := datautils.ToJSONMap(JobRunCancelledTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "job-run-cancelled",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

type JobRunTimedOutTaskPayload struct {
	JobRunId string `json:"job_run_id" validate:"required,uuid"`
}

type JobRunTimedOutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
