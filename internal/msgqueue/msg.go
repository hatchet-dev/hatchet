package msgqueue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/datautils"
)

// MaxMessageSize is the maximum size of a single serialized Message body;
// publishers recursively split larger multi-payload messages into chunks.
// This is Hatchet's own contract, shared by every backend rather than read
// from a broker: 16MiB matches RabbitMQ's server-side default
// max_message_size (its default since 3.12), and the NATS backend refuses
// to start unless the server's max_payload is at least this large. If the
// contract ever changes, this constant moves every backend together.
const MaxMessageSize = 16 * 1024 * 1024

type Message struct {
	// ID is the ID of the task.
	ID string `json:"id"`

	// Payloads is the list of payloads.
	Payloads [][]byte `json:"messages"`

	// TenantID is the tenant ID.
	TenantID uuid.UUID `json:"tenant_id"`

	// Whether the message should immediately expire if it reaches the queue without an active consumer.
	ImmediatelyExpire bool `json:"immediately_expire"`

	// Whether the message should be persisted to disk
	Persistent bool `json:"persistent"`

	// OtelCarrier is the OpenTelemetry carrier for the task.
	OtelCarrier map[string]string `json:"otel_carrier"`

	// Retries is the number of retries for the task.
	//
	// Deprecated: retries are set globally at the moment.
	Retries int `json:"retries"`

	// Compressed indicates whether the payloads are gzip compressed
	Compressed bool `json:"compressed,omitempty"`

	// PublishedAt is stamped by the instrumented pub/sub at publish time and is
	// used to observe transit latency on delivery. Zero for durable-queue
	// messages and messages from engines that predate the field.
	PublishedAt time.Time `json:"published_at,omitzero"`
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

// Clone returns a shallow copy of the message. Payloads and OtelCarrier are
// shared with the original: callers must replace them, never mutate in place.
func (t *Message) Clone() *Message {
	c := *t
	return &c
}

func (t *Message) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Message) SetOtelCarrier(otelCarrier map[string]string) {
	t.OtelCarrier = otelCarrier
}
