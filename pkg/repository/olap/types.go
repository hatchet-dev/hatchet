package olap

import (
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

// type WorkflowRun struct {
// 	// AdditionalMetadata Additional metadata for the workflow run.
// 	AdditionalMetadata *string `json:"additionalMetadata,omitempty"`

// 	CreatedAt time.Time `json:"createdAt"`

// 	// DisplayName The display name of the workflow run.
// 	DisplayName *string `json:"displayName,omitempty"`

// 	// Duration The duration of the workflow run.
// 	Duration *int64 `json:"duration,omitempty"`

// 	// ErrorMessage The error message of the workflow run.
// 	ErrorMessage *string `json:"errorMessage,omitempty"`

// 	// FinishedAt The timestamp the workflow run finished.
// 	FinishedAt *time.Time `json:"finishedAt,omitempty"`

// 	// Id The ID of the workflow run.
// 	Id uuid.UUID `json:"id"`

// 	Input string `json:"input"`

// 	Output string `json:"output"`

// 	// StartedAt The timestamp the workflow run started.
// 	StartedAt *time.Time `json:"startedAt,omitempty"`

// 	// Status The status of the workflow run.
// 	Status string `json:"status"`

// 	// TaskId The ID of the task associated with this workflow run.
// 	TaskId uuid.UUID `json:"taskId"`

// 	// TenantId The ID of the tenant.
// 	TenantId *uuid.UUID `json:"tenantId,omitempty"`

// 	// Timestamp The timestamp of the workflow run.
// 	Timestamp time.Time `json:"timestamp"`

// 	WorkflowId uuid.UUID `json:"workflowId"`
// }

type TaskRunEvent struct {
	// Id The ID of the workflow run.
	Id uuid.UUID `json:"id"`

	// TaskId The ID of the task associated with this workflow run.
	TaskId uuid.UUID `json:"taskId"`

	// Timestamp The timestamp of the workflow run.
	Timestamp time.Time `json:"timestamp"`

	Message string `json:"message"`

	Data string `json:"data"`

	EventType string `json:"eventType"`

	ErrorMsg string `json:"errorMsg"`

	WorkerId *uuid.UUID `json:"workerId,omitempty"`

	TaskDisplayName string `json:"taskDisplayName"`

	TaskInput string `json:"taskInput"`

	AdditionalMetadata string `json:"additionalMetadata"`
}

type Sticky string

const (
	STICKY_HARD Sticky = "HARD"
	STICKY_SOFT Sticky = "SOFT"
	STICKY_NONE Sticky = "NONE"
)

type EventType string

const (
	EVENT_TYPE_REQUEUED_NO_WORKER   EventType = "REQUEUED_NO_WORKER"
	EVENT_TYPE_REQUEUED_RATE_LIMIT  EventType = "REQUEUED_RATE_LIMIT"
	EVENT_TYPE_SCHEDULING_TIMED_OUT EventType = "SCHEDULING_TIMED_OUT"
	EVENT_TYPE_ASSIGNED             EventType = "ASSIGNED"
	EVENT_TYPE_STARTED              EventType = "STARTED"
	EVENT_TYPE_FINISHED             EventType = "FINISHED"
	EVENT_TYPE_FAILED               EventType = "FAILED"
	EVENT_TYPE_RETRYING             EventType = "RETRYING"
	EVENT_TYPE_CANCELLED            EventType = "CANCELLED"
	EVENT_TYPE_TIMED_OUT            EventType = "TIMED_OUT"
	EVENT_TYPE_REASSIGNED           EventType = "REASSIGNED"
	EVENT_TYPE_SLOT_RELEASED        EventType = "SLOT_RELEASED"
	EVENT_TYPE_TIMEOUT_REFRESHED    EventType = "TIMEOUT_REFRESHED"
	EVENT_TYPE_RETRIED_BY_USER      EventType = "RETRIED_BY_USER"
	EVENT_TYPE_SENT_TO_WORKER       EventType = "SENT_TO_WORKER"
	EVENT_TYPE_RATE_LIMIT_ERROR     EventType = "RATE_LIMIT_ERROR"
	EVENT_TYPE_ACKNOWLEDGED         EventType = "ACKNOWLEDGED"
	EVENT_TYPE_CREATED              EventType = "CREATED"
	EVENT_TYPE_QUEUED               EventType = "QUEUED"
)

type ReadableTaskStatus string

const (
	READABLE_TASK_STATUS_QUEUED    ReadableTaskStatus = "QUEUED"
	READABLE_TASK_STATUS_RUNNING   ReadableTaskStatus = "RUNNING"
	READABLE_TASK_STATUS_COMPLETED ReadableTaskStatus = "COMPLETED"
	READABLE_TASK_STATUS_CANCELLED ReadableTaskStatus = "CANCELLED"
	READABLE_TASK_STATUS_FAILED    ReadableTaskStatus = "FAILED"
)

func (s ReadableTaskStatus) EnumValue() int {
	switch s {
	case READABLE_TASK_STATUS_QUEUED:
		return 1
	case READABLE_TASK_STATUS_RUNNING:
		return 2
	case READABLE_TASK_STATUS_COMPLETED:
		return 3
	case READABLE_TASK_STATUS_CANCELLED:
		return 4
	case READABLE_TASK_STATUS_FAILED:
		return 5
	default:
		return -1
	}
}

type TaskRunMetric struct {
	Status string `json:"status"`
	Count  uint64 `json:"count"`
}

type TaskRunDataRow struct {
	Parent   *timescalev2.ListWorkflowRunsRow
	Children []*timescalev2.ListDAGChildrenRow
}
