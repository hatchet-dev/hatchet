package v1

import (
	"time"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

type UserEventTaskPayload struct {
	EventId                 string `json:"event_id" validate:"required,uuid"`
	EventKey                string `json:"event_key" validate:"required"`
	EventData               []byte `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte `json:"event_additional_metadata"`
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
