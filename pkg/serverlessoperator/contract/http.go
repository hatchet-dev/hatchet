package contract

import (
	"encoding/json"
	"time"
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

// TriggerEnvelopeVersion is the version field of the non-durable trigger envelope.
const TriggerEnvelopeVersion = 1

// HealthcheckRequest is the body of POST healthcheck_url. Namespace lets the endpoint SDK
// apply the same prefix the operator applies to everything it registers.
type HealthcheckRequest struct {
	EndpointId string `json:"endpoint_id"`
	Namespace  string `json:"namespace"`
	Timestamp  int64  `json:"timestamp"`
}

// HealthcheckResponse is what an endpoint answers. Workflows are protojson-encoded
// v1.CreateWorkflowVersionRequest messages without the namespace prefix; Actions are extra
// action ids not derived from a workflow. The legacy shape {"actions": [...]} parses as a
// response with no workflows.
type HealthcheckResponse struct {
	Durable   *HealthcheckDurable `json:"durable,omitempty"`
	Runtime   *HealthcheckRuntime `json:"runtime,omitempty"`
	Workflows []json.RawMessage   `json:"workflows,omitempty"`
	Actions   []string            `json:"actions,omitempty"`
}

type HealthcheckDurable struct {
	Supported bool `json:"supported"`
}

type HealthcheckRuntime struct {
	Name       string `json:"name,omitempty"`
	SdkVersion string `json:"sdk_version,omitempty"`
}

// TriggerEnvelope is the body of POST trigger_url for a non-durable task. Action is the
// protojson-encoded AssignedAction with the namespaced action id and workflow name.
type TriggerEnvelope struct {
	EndpointId string          `json:"endpoint_id"`
	Namespace  string          `json:"namespace"`
	Action     json.RawMessage `json:"action"`
	Timestamp  int64           `json:"timestamp"`
	Version    int             `json:"version"`
}

// TriggerErrorResponse is the optional body an endpoint returns to override the status
// mapping: Error is the reported message and Retry decides whether the engine retries.
type TriggerErrorResponse struct {
	Retry *bool  `json:"retry,omitempty"`
	Error string `json:"error"`
}
