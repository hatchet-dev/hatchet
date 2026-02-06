package msgqueue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/datautils"
)

type Message struct {
	OtelCarrier       map[string]string `json:"otel_carrier"`
	ID                string            `json:"id"`
	Payloads          [][]byte          `json:"messages"`
	Retries           int               `json:"retries"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	ImmediatelyExpire bool              `json:"immediately_expire"`
	Persistent        bool              `json:"persistent"`
	Compressed        bool              `json:"compressed,omitempty"`
}

func NewTenantMessage[T any](tenantId uuid.UUID, id string, immediatelyExpire, persistent bool, payloads ...T) (*Message, error) {
	payloadByteArr := make([][]byte, len(payloads))

	for i, payload := range payloads {
		payloadBytes, err := json.Marshal(payload)

		if err != nil {
			return nil, err
		}

		payloadByteArr[i] = payloadBytes
	}

	return &Message{
		ID:                id,
		Payloads:          payloadByteArr,
		TenantID:          tenantId,
		ImmediatelyExpire: immediatelyExpire,
		Persistent:        persistent,
		Retries:           5,
	}, nil
}

func DecodeAndValidateSingleton(dv datautils.DataDecoderValidator, payloads [][]byte, target interface{}) error {
	if len(payloads) != 1 {
		return fmt.Errorf("expected exactly one payload, got %d", len(payloads))
	}

	return dv.DecodeAndValidate(payloads[0], target)
}

func (t *Message) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Message) SetOtelCarrier(otelCarrier map[string]string) {
	t.OtelCarrier = otelCarrier
}
