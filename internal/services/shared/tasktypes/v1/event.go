package v1

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type UserEventTaskPayload struct {
	EventExternalId         string  `json:"event_id" validate:"required,uuid"`
	EventKey                string  `json:"event_key" validate:"required"`
	EventData               []byte  `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte  `json:"event_additional_metadata"`
	EventPriority           *int32  `json:"event_priority,omitempty"`
	EventScope              *string `json:"event_scope,omitempty"`
	TriggeringWebhookName   *string `json:"triggering_webhook_name,omitempty"`

	// WasProcessedLocally indicates whether the event was written and tasks were triggered on the gRPC server
	// instead of the controller, so we can skip the triggering logic downstream
	WasProcessedLocally bool `json:"was_processed_locally"`
}

func NewInternalEventMessage(tenantId string, timestamp time.Time, events ...v1.InternalTaskEvent) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDInternalEvent,
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
	EventIndex    *int64    `json:"event_index"`
}
