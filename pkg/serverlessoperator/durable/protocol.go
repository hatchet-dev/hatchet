// Package durable relays one durable task invocation between the engine, reached through a
// link.DurableChannel, and a serverless endpoint, reached over an operator-dialed websocket.
// The socket is the invocation's single request: the core sends the assigned action as the
// first frame, forwards the endpoint's DurableTaskRequests to the engine and the engine's
// DurableTaskResponses back, and reads the outcome from the endpoint's final done frame.
package durable

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Close codes the relay sends. The 4xxx codes are application-defined; 1013 is the standard
// "try again later" code, used for backpressure.
const (
	// CloseEvicted follows a forwarded server_evict: the engine superseded the invocation.
	CloseEvicted = 4001
	// CloseShuttingDown means the operator is stopping; the engine re-delivers the task.
	CloseShuttingDown = 4002
	// CloseCancelled means the engine cancelled the task.
	CloseCancelled = 4003
	// CloseInvocationMismatch means a request carried another task id or invocation count.
	CloseInvocationMismatch = 4004
	// CloseForbiddenMessage means the endpoint sent a link-internal request
	// (register_worker, worker_status) or an unrecognised frame.
	CloseForbiddenMessage = 4005
	// CloseRequestInFlight means a second ack-bearing request was sent before the first
	// was acknowledged.
	CloseRequestInFlight = 4006
	// CloseTimeout means the endpoint's request timeout elapsed with no done frame.
	CloseTimeout = 4007
	// CloseUnresponsive means two consecutive pings went unanswered.
	CloseUnresponsive = 4008
	// CloseBackpressure means the endpoint fell more than sendQueueSize frames behind.
	CloseBackpressure = 1013
	// CloseNormal is sent after a done frame.
	CloseNormal = 1000
	// CloseInternalError is sent when the engine side of the relay fails.
	CloseInternalError = 1011
)

// FirstFrame is the first frame the core sends after the upgrade: the assigned action
// (protojson, action id and workflow name namespaced as registered), the endpoint's
// namespace, the invocation count and how long the endpoint may wait inline for a wait_for
// entry before evicting.
type FirstFrame struct {
	Action             json.RawMessage `json:"action"`
	Namespace          string          `json:"namespace"`
	InvocationCount    int32           `json:"invocation_count"`
	InlineWaitBudgetMs int32           `json:"inline_wait_budget_ms"`
}

// InboundFrame is any frame the endpoint sends: a request (with an endpoint-chosen sequence
// id, echoed in logs only) or the final done frame.
type InboundFrame struct {
	Request json.RawMessage `json:"request,omitempty"`
	Done    json.RawMessage `json:"done,omitempty"`
	Id      *int64          `json:"id,omitempty"`
}

// DoneFrame is the endpoint's terminal frame. Exactly one shape applies, checked in this
// order: Status "evicted" (no terminal event), Error set (FAILED, Retry decides whether the
// engine retries), otherwise Output (COMPLETED; an absent output completes with {}).
type DoneFrame struct {
	Output json.RawMessage `json:"output,omitempty"`
	Error  *string         `json:"error,omitempty"`
	Status string          `json:"status,omitempty"`
	Retry  bool            `json:"retry,omitempty"`
}

// StatusEvicted is the DoneFrame.Status value for an evicted invocation.
const StatusEvicted = "evicted"

// ResponseFrame carries one protojson DurableTaskResponse from the engine.
type ResponseFrame struct {
	Response json.RawMessage `json:"response"`
}

// ErrorFrame reports an engine-side error for the invocation, such as a non-determinism
// error on replay. The endpoint is expected to finish with done {"error", "retry": false}.
type ErrorFrame struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the payload of an ErrorFrame. Code is the lowercased DurableTaskErrorType
// without its prefix ("nondeterminism"), or "unspecified".
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorCodeNonDeterminism is the ErrorBody.Code for a replay that diverged from the log.
const ErrorCodeNonDeterminism = "nondeterminism"

var (
	marshalOpts   = protojson.MarshalOptions{}
	unmarshalOpts = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// MarshalProto encodes a protobuf message as protojson for a frame.
func MarshalProto(m proto.Message) (json.RawMessage, error) {
	raw, err := marshalOpts.Marshal(m)

	if err != nil {
		return nil, fmt.Errorf("could not encode %s: %w", m.ProtoReflect().Descriptor().FullName(), err)
	}

	return raw, nil
}

// UnmarshalProto decodes a frame's protojson payload into m, ignoring unknown fields so a
// newer endpoint SDK does not break an older operator.
func UnmarshalProto(raw json.RawMessage, m proto.Message) error {
	if err := unmarshalOpts.Unmarshal(raw, m); err != nil {
		return fmt.Errorf("could not decode %s: %w", m.ProtoReflect().Descriptor().FullName(), err)
	}

	return nil
}
