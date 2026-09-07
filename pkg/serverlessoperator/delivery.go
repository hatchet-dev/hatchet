package serverlessoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

const (
	defaultRequestTimeout = 60 * time.Second
	maxRequestTimeout     = 600 * time.Second
)

// outcome is what one delivery attempt reports back to the engine.
type outcome struct {
	errMsg string
	// result labels the deliveries_total metric: completed, failed, retryable, cancelled.
	result string
	output []byte
	status contracts.StepActionEventType
	retry  bool
}

func completedOutcome(output []byte) outcome {
	if len(output) == 0 {
		output = []byte("{}")
	}

	return outcome{status: contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, output: output, result: "completed"}
}

func failedOutcome(msg string, retry bool) outcome {
	result := "failed"

	if retry {
		result = "retryable"
	}

	return outcome{status: contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, errMsg: msg, retry: retry, result: result}
}

func cancelledOutcome() outcome {
	return outcome{status: contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, result: "cancelled"}
}

// buildTriggerEnvelope serializes the trigger body. The action is protojson so new
// AssignedAction fields flow through without a hand-maintained struct; action_id and
// workflow_name are delivered as registered, namespace prefix included.
func buildTriggerEnvelope(action *contracts.AssignedAction, ep *cachedEndpoint, timestamp int64) ([]byte, error) {
	raw, err := protojson.Marshal(action)

	if err != nil {
		return nil, fmt.Errorf("could not encode action: %w", err)
	}

	return json.Marshal(contract.TriggerEnvelope{
		Version:    contract.TriggerEnvelopeVersion,
		EndpointId: ep.id.String(),
		Namespace:  ep.namespace.String(),
		Timestamp:  timestamp,
		Action:     raw,
	})
}

func requestTimeout(cfg *endpointConfig) time.Duration {
	timeout := defaultRequestTimeout

	if cfg.requestTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.requestTimeoutSeconds) * time.Second
	}

	return min(timeout, maxRequestTimeout)
}

// deliverAction POSTs the signed envelope and maps the result. ctx is the delivery context
// the runner cancels on CANCEL_STEP_RUN; the endpoint's request timeout is applied on top.
func deliverAction(ctx context.Context, sender RequestSender, ep *cachedEndpoint, cfg *endpointConfig, action *contracts.AssignedAction) outcome {
	if cfg.secretErr != nil {
		return failedOutcome(cfg.secretErr.Error(), false)
	}

	now := time.Now().Unix()

	body, err := buildTriggerEnvelope(action, ep, now)

	if err != nil {
		return failedOutcome(err.Error(), false)
	}

	headers, err := signedHeaders(cfg.secret, ep.id, now, body)

	if err != nil {
		return failedOutcome(err.Error(), false)
	}

	timeout := requestTimeout(cfg)

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := sender.Deliver(rctx, http.MethodPost, cfg.triggerUrl, body, headers)

	if err != nil {
		return classifyError(ctx, err, timeout)
	}

	return classifyResponse(res)
}

// classifyError maps a transport-level failure. A cancelled delivery context means a
// CANCEL_STEP_RUN arrived; the deadline is the endpoint's request timeout. Policy blocks
// from safeclient are configuration errors, not transient, so they are not retried.
func classifyError(ctx context.Context, err error, timeout time.Duration) outcome {
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return cancelledOutcome()
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return failedOutcome(fmt.Sprintf("request timed out after %s", timeout), true)
	}

	if errors.Is(err, safeclient.ErrBlockedDestination) || errors.Is(err, safeclient.ErrBadScheme) || errors.Is(err, safeclient.ErrBadPort) {
		return failedOutcome(err.Error(), false)
	}

	if errors.Is(err, safeclient.ErrResponseTooLarge) {
		return failedOutcome("response body exceeded the 4 MiB limit", false)
	}

	return failedOutcome(fmt.Sprintf("request failed: %s", err.Error()), true)
}

// classifyResponse implements the status mapping table: 2xx JSON is the output, 4xx is
// permanent except 408, 425 and 429, 5xx is retryable, and a {"error", "retry"} body
// overrides both the message and the retry decision on any non-2xx response.
func classifyResponse(res *safeclient.DeliveryResult) outcome {
	code := res.StatusCode

	switch {
	case code == http.StatusNoContent:
		return completedOutcome(nil)
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		if len(res.BodyPrefix) > 0 && !json.Valid(res.BodyPrefix) {
			return failedOutcome(fmt.Sprintf("endpoint returned status %d with a non-JSON body", code), false)
		}

		return completedOutcome(res.BodyPrefix)
	}

	retry := code >= http.StatusInternalServerError ||
		code == http.StatusRequestTimeout ||
		code == http.StatusTooEarly ||
		code == http.StatusTooManyRequests

	msg := fmt.Sprintf("endpoint returned status %d", code)

	if text := http.StatusText(code); text != "" {
		msg = fmt.Sprintf("endpoint returned status %d %s", code, text)
	}

	if code >= http.StatusMultipleChoices && code < http.StatusBadRequest {
		msg = fmt.Sprintf("endpoint returned redirect status %d; redirects are not followed", code)
	}

	var override contract.TriggerErrorResponse

	if len(res.BodyPrefix) > 0 && json.Unmarshal(res.BodyPrefix, &override) == nil {
		if override.Error != "" {
			msg = override.Error
		}

		if override.Retry != nil {
			retry = *override.Retry
		}
	}

	return failedOutcome(msg, retry)
}
