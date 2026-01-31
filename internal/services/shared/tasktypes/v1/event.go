package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type UserEventTaskPayload struct {
	EventExternalId         uuid.UUID `json:"event_id" validate:"required"`
	EventKey                string    `json:"event_key" validate:"required"`
	EventData               []byte    `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte    `json:"event_additional_metadata"`
	EventPriority           *int32    `json:"event_priority,omitempty"`
	EventScope              *string   `json:"event_scope,omitempty"`
	TriggeringWebhookName   *string   `json:"triggering_webhook_name,omitempty"`
}

func NewInternalEventMessage(tenantId uuid.UUID, timestamp time.Time, events ...v1.InternalTaskEvent) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDInternalEvent,
		false,
		true,
		events...,
	)
}

type StreamEventPayload struct {
	WorkflowRunId uuid.UUID `json:"workflow_run_id" validate:"required,uuid"`
	StepRunId     uuid.UUID `json:"step_run_id" validate:"required,uuid"`
	CreatedAt     time.Time `json:"created_at" validate:"required"`
	Payload       []byte    `json:"payload"`
	RetryCount    *int32    `json:"retry_count,omitempty"`
	EventIndex    *int64    `json:"event_index"`
}
