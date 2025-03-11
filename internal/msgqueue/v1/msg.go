package v1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/datautils"
)

type Message struct {
	// ID is the ID of the task.
	ID string `json:"id"`

	// Payloads is the list of payloads.
	Payloads [][]byte `json:"messages"`

	// TenantID is the tenant ID.
	TenantID string `json:"tenant_id"`

	// Whether the message should immediately expire if it reaches the queue without an active consumer.
	ImmediatelyExpire bool `json:"immediately_expire"`

	// Whether the message should be persisted to disk
	Persistent bool `json:"persistent"`

	// OtelCarrier is the OpenTelemetry carrier for the task.
	OtelCarrier map[string]string `json:"otel_carrier"`

	// Retries is the number of retries for the task.
	Retries int `json:"retries"`
}

func NewTenantMessage[T any](tenantId, id string, immediatelyExpire, persistent bool, payloads ...T) (*Message, error) {
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
