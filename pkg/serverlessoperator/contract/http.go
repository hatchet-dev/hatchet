// Package contract holds the parts of the serverless endpoint contract that are not protobuf
// messages: header names, the upgrade signature payload, wire constants and the protojson
// encoding every message travels in. The messages themselves are generated from
// api-contracts/v1/serverless.proto into internal/services/shared/proto/v1, which is also
// the source of the endpoint SDK's TypeScript types.
package contract

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// Headers carried by every request the operator sends to an endpoint. Endpoints verify
// SignatureHeader by recomputing the HMAC-SHA256 hex digest of the raw body with their
// signing secret (internal/signature.Verify).
const (
	SignatureHeader  = "X-Hatchet-Signature"
	EndpointIdHeader = "X-Hatchet-Endpoint-Id"
	TimestampHeader  = "X-Hatchet-Timestamp"
)

// Headers carried only by the durable websocket upgrade, which has no body to sign. The
// endpoint verifies SignatureHeader against UpgradeSigningPayload with its signing secret,
// rejects timestamps older than UpgradeMaxAge, and may keep a nonce set against replay.
const (
	NonceHeader      = "X-Hatchet-Nonce"
	TaskIdHeader     = "X-Hatchet-Task-Id"
	InvocationHeader = "X-Hatchet-Invocation"
)

// UpgradeMaxAge is how old an upgrade timestamp may be before an endpoint rejects it.
const UpgradeMaxAge = 5 * time.Minute

// UpgradeSigningPayload is the string the durable upgrade signature covers:
// timestamp "." nonce "." task_id "." invocation, each as it appears in its header.
func UpgradeSigningPayload(timestamp, nonce, taskId, invocation string) string {
	return timestamp + "." + nonce + "." + taskId + "." + invocation
}

// TriggerEnvelopeVersion is the version field of v1.ServerlessTriggerRequest.
const TriggerEnvelopeVersion = 1

// DoneStatusEvicted is the v1.ServerlessDoneFrame status of an invocation that evicted
// itself.
const DoneStatusEvicted = "evicted"

// Codes of v1.ServerlessErrorFrame.
const (
	// ErrorCodeNonDeterminism reports a replay that diverged from the log.
	ErrorCodeNonDeterminism = "nondeterminism"
	// ErrorCodeUnspecified is every other engine-side error.
	ErrorCodeUnspecified = "unspecified"
)

var (
	// marshalOpts is the default protojson encoding: lowerCamelCase names, int64 as
	// strings, bytes as base64, enums as names. The endpoint SDK's generated toJSON writes
	// the same encoding.
	marshalOpts = protojson.MarshalOptions{}

	// unmarshalOpts ignores unknown fields so a newer endpoint SDK does not break an older
	// operator.
	unmarshalOpts = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// Marshal encodes a contract message as protojson.
func Marshal(m proto.Message) ([]byte, error) {
	raw, err := marshalOpts.Marshal(m)

	if err != nil {
		return nil, fmt.Errorf("could not encode %s: %w", m.ProtoReflect().Descriptor().FullName(), err)
	}

	return raw, nil
}

// Unmarshal decodes protojson into m, ignoring unknown fields.
func Unmarshal(raw []byte, m proto.Message) error {
	if err := unmarshalOpts.Unmarshal(raw, m); err != nil {
		return fmt.Errorf("could not decode %s: %w", m.ProtoReflect().Descriptor().FullName(), err)
	}

	return nil
}

// MarshalFrame encodes one durable websocket frame.
func MarshalFrame(frame *v1.ServerlessDurableFrame) ([]byte, error) {
	return Marshal(frame)
}

// UnmarshalFrame decodes one durable websocket frame.
func UnmarshalFrame(raw []byte) (*v1.ServerlessDurableFrame, error) {
	frame := &v1.ServerlessDurableFrame{}

	if err := Unmarshal(raw, frame); err != nil {
		return nil, err
	}

	return frame, nil
}
