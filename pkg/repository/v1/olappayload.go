package v1

import (
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/jackc/pgx/v5/pgtype"
)

type OLAPPayloadToOffload struct {
	InsertedAt          pgtype.Timestamptz
	ExternalId          pgtype.UUID
	ExternalLocationKey string
}

type OLAPPayloadsToOffload struct {
	Payloads []OLAPPayloadToOffload
}

func OLAPPayloadOffloadMessage(tenantId string, payloads []OLAPPayloadToOffload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"offload-payload",
		false,
		true,
		OLAPPayloadsToOffload{
			Payloads: payloads,
		},
	)
}
