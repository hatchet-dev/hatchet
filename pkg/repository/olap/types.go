package olap

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowRun struct {
	// AdditionalMetadata Additional metadata for the workflow run.
	AdditionalMetadata *string `json:"additionalMetadata,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	// DisplayName The display name of the workflow run.
	DisplayName *string `json:"displayName,omitempty"`

	// Duration The duration of the workflow run.
	Duration *int64 `json:"duration,omitempty"`

	// ErrorMessage The error message of the workflow run.
	ErrorMessage *string `json:"errorMessage,omitempty"`

	// FinishedAt The timestamp the workflow run finished.
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	// Id The ID of the workflow run.
	Id uuid.UUID `json:"id"`

	Input string `json:"input"`

	Output string `json:"output"`

	// StartedAt The timestamp the workflow run started.
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// Status The status of the workflow run.
	Status string `json:"status"`

	// TaskId The ID of the task associated with this workflow run.
	TaskId uuid.UUID `json:"taskId"`

	// TenantId The ID of the tenant.
	TenantId *uuid.UUID `json:"tenantId,omitempty"`

	// Timestamp The timestamp of the workflow run.
	Timestamp time.Time `json:"timestamp"`
}

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
)

type ReadableTaskStatus string

const (
	READABLE_TASK_STATUS_QUEUED    ReadableTaskStatus = "QUEUED"
	READABLE_TASK_STATUS_RUNNING   ReadableTaskStatus = "RUNNING"
	READABLE_TASK_STATUS_COMPLETED ReadableTaskStatus = "COMPLETED"
	READABLE_TASK_STATUS_CANCELLED ReadableTaskStatus = "CANCELLED"
	READABLE_TASK_STATUS_FAILED    ReadableTaskStatus = "FAILED"
)

type Task struct {
	Id                 uuid.UUID  `json:"id"`
	TenantId           uuid.UUID  `json:"tenant_id"`
	Queue              string     `json:"queue"`
	ActionId           string     `json:"action_id"`
	ScheduleTimeout    string     `json:"schedule_timeout"`
	StepTimeout        string     `json:"step_timeout"`
	Priority           int32      `json:"priority"`
	Sticky             Sticky     `json:"sticky"`
	DesiredWorkerId    *uuid.UUID `json:"desired_worker_id,omitempty"`
	DisplayName        string     `json:"display_name"`
	Input              string     `json:"input"`
	AdditionalMetadata string     `json:"additional_metadata"`
}

type TaskEvent struct {
	TaskId                 uuid.UUID  `json:"task_id"`
	TenantId               uuid.UUID  `json:"tenant_id"`
	EventType              EventType  `json:"event_type"`
	Timestamp              time.Time  `json:"timestamp"`
	RetryCount             uint32     `json:"retry_count"`
	ErrorMsg               string     `json:"error_message"`
	Output                 string     `json:"output"`
	WorkerId               *uuid.UUID `json:"worker_id,omitempty"`
	AdditionalEventData    string     `json:"additional__event_data"`
	AdditionalEventMessage string     `json:"additional__event_message"`
}
