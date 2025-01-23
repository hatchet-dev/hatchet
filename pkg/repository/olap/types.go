package olap

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowRun struct {
	Id           uuid.UUID `json:"id"`
	TaskId       int32     `json:"task_id"`
	WorkerId     int32     `json:"worker_id"`
	TenantId     uuid.UUID `json:"tenant_id"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	RetryCount   int32     `json:"retry_count"`
	ErrorMessage string    `json:"error_message"`
}

type Event struct {
	TaskId                         int64     `json:"task_id"`
	TenantId                       uuid.UUID `json:"tenant_id"`
	Queue                          string    `json:"queue"`
	ActionId                       string    `json:"action_id"`
	ScheduleTimeout                string    `json:"schedule_timeout"`
	StepTimeout                    *string   `json:"step_timeout,omitempty"`
	Priority                       int32     `json:"priority"`
	Sticky                         string    `json:"sticky"`
	DesiredWorkerId                uuid.UUID `json:"desired_worker_id"`
	ExternalId                     uuid.UUID `json:"external_id"`
	DisplayName                    string    `json:"display_name"`
	Input                          string    `json:"input"`
	WorkerId                       uuid.UUID `json:"worker_id"`
	Status                         string    `json:"status"`
	Timestamp                      time.Time `json:"timestamp"`
	RetryCount                     int32     `json:"retry_count"`
	ErrorMsg                       *string   `json:"error_message,omitempty"`
	AdditionalStepRunEventData     string    `json:"additional__step_run_event_data"`
	AdditionalStepRunEventMessage  string    `json:"additional__step_run_event_message"`
	AdditionalStepRunEventSeverity string    `json:"additional__step_run_event_severity"`
	AdditionalStepRunEventReason   string    `json:"additional__step_run_event_reason"`
}
