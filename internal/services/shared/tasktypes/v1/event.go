package v1

import (
	"time"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
)

type UserEventTaskPayload struct {
	EventId                 string `json:"event_id" validate:"required,uuid"`
	EventKey                string `json:"event_key" validate:"required"`
	EventData               []byte `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte `json:"event_additional_metadata"`
}

type InternalEventTaskPayload struct {
	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`
	EventKey       string    `json:"event_key" validate:"required"`
	EventData      []byte    `json:"event_data" validate:"required"`
}

func NewInternalEventMessage(tenantId string, timestamp time.Time, events ...InternalEventTaskPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"internal-event",
		false,
		true,
		events...,
	)
}
