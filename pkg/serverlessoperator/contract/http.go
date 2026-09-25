// Package contract holds the parts of the serverless endpoint contract that are not protobuf
// messages: header names, the upgrade signature payload, wire constants and the protojson
// encoding every message travels in. The messages themselves are generated from
// api-contracts/v1/serverless.proto into internal/services/shared/proto/v1, which is also
// the source of the endpoint SDK's TypeScript types.
package contract

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// Headers carried by every request the operator sends to an endpoint. Endpoints verify
// SignatureHeader by recomputing the HMAC-SHA256 hex digest of the raw body with their
// signing secret (internal/signature.Verify). The signed body of every POST
// (v1.ServerlessHealthcheckRequest, v1.ServerlessTriggerRequest) carries the operator's
// timestamp in Unix seconds; an endpoint rejects a body whose timestamp lies more than
// RequestMaxAge from its own clock in either direction, so a captured request cannot be
// replayed later. Within that window a delivery may be repeated: endpoints key side effects
// on (endpoint id, task run external id, retry count).
const (
	SignatureHeader  = "X-Hatchet-Signature"
	EndpointIdHeader = "X-Hatchet-Endpoint-Id"
	TimestampHeader  = "X-Hatchet-Timestamp"
)

// Headers carried only by the durable websocket upgrade, which has no body to sign. The
// endpoint verifies SignatureHeader against UpgradeSigningPayload with its signing secret,
// rejects a TimestampHeader more than UpgradeMaxAge from its clock in either direction,
// consumes NonceHeader from a nonce set after the signature verified so a captured upgrade
// cannot be replayed within the window (every accepted nonce is kept until its window has
// passed; an endpoint whose nonce storage cannot admit another refuses the upgrade rather
// than forget a live nonce), and checks that the first frame's task id and invocation are
// the ones the headers were signed for.
const (
	NonceHeader      = "X-Hatchet-Nonce"
	TaskIdHeader     = "X-Hatchet-Task-Id"
	InvocationHeader = "X-Hatchet-Invocation"
)

// RequestMaxAge is how far a signed timestamp may lie from the endpoint's clock, in either
// direction, before the endpoint rejects the request. It applies to the timestamp inside
// every signed POST body and to the upgrade's TimestampHeader.
const RequestMaxAge = 5 * time.Minute

// UpgradeMaxAge is RequestMaxAge as it applies to the durable upgrade.
const UpgradeMaxAge = RequestMaxAge

// UpgradeSigningPayload is the string the durable upgrade signature covers:
// endpoint_id "." timestamp "." nonce "." task_id "." invocation, each as it appears in its
// header. The endpoint id is part of it so a signature made for one endpoint cannot be
// presented to another that shares the signing secret: the endpoint verifies the id it
// serves against the signed one, not against an unsigned header alone.
func UpgradeSigningPayload(endpointId, timestamp, nonce, taskId, invocation string) string {
	return endpointId + "." + timestamp + "." + nonce + "." + taskId + "." + invocation
}

// TriggerEnvelopeVersion is the version field of v1.ServerlessTriggerRequest.
const TriggerEnvelopeVersion = 1

// NamespaceSeparator joins an endpoint's namespace to the names it owns, the way the SDK's
// HATCHET_CLIENT_NAMESPACE does. The operator registers every workflow name, action service
// and event key an endpoint serves as <namespace><separator><name>, and the durable relay
// applies the same prefix to the workflow names and user event keys an endpoint references
// in nested requests, so the namespace is the boundary of what an endpoint can reach.
const NamespaceSeparator = "_"

// NamespacePrefix is what ApplyNamespace prepends.
func NamespacePrefix(namespace string) string {
	return namespace + NamespaceSeparator
}

// ApplyNamespace prefixes a workflow name or event key with the namespace. A name that
// already carries the prefix is left alone, so applying it twice is harmless; an empty
// namespace applies nothing.
func ApplyNamespace(namespace, name string) string {
	if namespace == "" {
		return name
	}

	prefix := NamespacePrefix(namespace)

	if strings.HasPrefix(name, prefix) {
		return name
	}

	return prefix + name
}

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
