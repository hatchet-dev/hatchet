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
	TaskId     int32     `json:"task_id"`
	WorkerId   int32     `json:"worker_id"`
	TenantId   uuid.UUID `json:"tenant_id"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int32     `json:"retry_count"`
	ErrorMsg   *string   `json:"error_message,omitempty"`
}
