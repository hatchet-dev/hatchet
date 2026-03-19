package v1

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

func NewCronUpdateMessage(tenantId uuid.UUID, id string) *msgqueue.Message {
	msg := &msgqueue.Message{
		ID:                id,
		Payloads:          nil,
		TenantID:          tenantId,
		ImmediatelyExpire: true,
		Persistent:        false,
		Retries:           5,
	}
	return msg
}
