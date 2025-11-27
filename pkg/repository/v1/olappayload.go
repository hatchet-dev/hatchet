package v1

import (
	"github.com/google/uuid"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
)

type OLAPPayloadToOffload struct {
	ExternalId          uuid.UUID
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
