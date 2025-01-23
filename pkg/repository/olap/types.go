package olap

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowRun struct {
	Id           uuid.UUID `json:"id"`
	TaskId       int32     `json:"task_id"`
	TenantId     uuid.UUID `json:"tenant_id"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	RetryCount   int32     `json:"retry_count"`
	ErrorMessage string    `json:"error_message"`
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
