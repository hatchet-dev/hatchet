package repository

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type OLAPPayloadToOffload struct {
	ExternalId          pgtype.UUID
	ExternalLocationKey string
}

type OLAPPayloadsToOffload struct {
	Payloads []OLAPPayloadToOffload
}

func OLAPPayloadOffloadMessage(tenantId string, payloads []OLAPPayloadToOffload) (*msgqueue.Message, error) {
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
