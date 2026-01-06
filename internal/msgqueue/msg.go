package msgqueue

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/datautils"
)

type Message struct {
	ID                string            `json:"id"`
	Payloads          []json.RawMessage `json:"messages"`
	TenantID          string            `json:"tenant_id"`
	ImmediatelyExpire bool              `json:"immediately_expire"`
	Persistent        bool              `json:"persistent"`
	OtelCarrier       map[string]string `json:"otel_carrier"`
	Retries           int               `json:"retries"`
	Compressed        bool              `json:"compressed,omitempty"`
}

func NewTenantMessage[T any](tenantId, id string, immediatelyExpire, persistent bool, payloads ...T) (*Message, error) {
	payloadByteArr := make([]json.RawMessage, len(payloads))

	for i, payload := range payloads {
		payloadBytes, err := json.Marshal(payload)

		if err != nil {
			return nil, err
		}

		payloadByteArr[i] = json.RawMessage(payloadBytes)
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
