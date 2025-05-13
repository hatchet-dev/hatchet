package v1

import (
	"time"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

type UserEventTaskPayload struct {
	EventExternalId         string  `json:"event_id" validate:"required,uuid"`
	EventKey                string  `json:"event_key" validate:"required"`
	EventData               []byte  `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte  `json:"event_additional_metadata"`
	EventPriority           *int32  `json:"event_priority,omitempty"`
	EventResourceHint       *string `json:"event_resource_hint,omitempty"`
}

func NewInternalEventMessage(tenantId string, timestamp time.Time, events ...v1.InternalTaskEvent) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"internal-event",
		false,
		true,
		events...,
	)
}

type StreamEventPayload struct {
	WorkflowRunId string    `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     string    `json:"step_run_id" validate:"required,uuid"`
	CreatedAt     time.Time `json:"created_at" validate:"required"`
	Payload       []byte    `json:"payload"`
	RetryCount    *int32    `json:"retry_count,omitempty"`
}
