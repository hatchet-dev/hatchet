package repository

import (
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/jackc/pgx/v5/pgtype"
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
