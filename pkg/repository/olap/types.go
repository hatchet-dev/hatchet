package olap

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowRun struct {
	// AdditionalMetadata Additional metadata for the workflow run.
	AdditionalMetadata *string `json:"additionalMetadata,omitempty"`

	// DisplayName The display name of the workflow run.
	DisplayName *string `json:"displayName,omitempty"`

	// Duration The duration of the workflow run.
	Duration *int32 `json:"duration,omitempty"`

	// ErrorMessage The error message of the workflow run.
	ErrorMessage *string `json:"errorMessage,omitempty"`

	// FinishedAt The timestamp the workflow run finished.
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	// Id The ID of the workflow run.
	Id uuid.UUID `json:"id"`

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

type Task struct {
	Id                 uuid.UUID  `json:"id"`
	TenantId           uuid.UUID  `json:"tenant_id"`
	Queue              string     `json:"queue"`
	ActionId           string     `json:"action_id"`
	ScheduleTimeout    string     `json:"schedule_timeout"`
	StepTimeout        *string    `json:"step_timeout,omitempty"`
	Priority           int32      `json:"priority"`
	Sticky             *string    `json:"sticky"`
	DesiredWorkerId    *uuid.UUID `json:"desired_worker_id"`
	DisplayName        string     `json:"display_name"`
	Input              string     `json:"input"`
	AdditionalMetadata string     `json:"additional_metadata"`
}

type TaskEvent struct {
	TaskId                  uuid.UUID `json:"task_id"`
	TenantId                uuid.UUID `json:"tenant_id"`
	Status                  string    `json:"status"`
	Timestamp               time.Time `json:"timestamp"`
	RetryCount              int32     `json:"retry_count"`
	ErrorMsg                *string   `json:"error_message,omitempty"`
	Output                  *string   `json:"output,omitempty"`
	AdditionalEventData     *string   `json:"additional__event_data"`
	AdditionalEventMessage  *string   `json:"additional__event_message"`
	AdditionalEventSeverity *string   `json:"additional__event_severity"`
	AdditionalEventReason   *string   `json:"additional__event_reason"`
}
