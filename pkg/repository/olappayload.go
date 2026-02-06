package repository

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type OLAPPayloadToOffload struct {
	ExternalLocationKey string
	ExternalId          uuid.UUID
}

type OLAPPayloadsToOffload struct {
	Payloads []OLAPPayloadToOffload
}

func OLAPPayloadOffloadMessage(tenantId uuid.UUID, payloads []OLAPPayloadToOffload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDOffloadPayload,
		false,
		true,
		OLAPPayloadsToOffload{
			Payloads: payloads,
		},
	)
}
