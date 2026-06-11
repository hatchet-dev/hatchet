//go:build !e2e && !load && !rampup && !integration

package httpoperator

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator/safeclient"
)

// fakeSender captures the last delivery and returns a configurable result/error.
type fakeSender struct {
	gotMethod   string
	gotEndpoint string
	gotBody     []byte
	gotHeaders  http.Header
	result      *safeclient.DeliveryResult
	err         error
}

func (f *fakeSender) Deliver(_ context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error) {
	f.gotMethod = method
	f.gotEndpoint = endpoint
	f.gotBody = body
	f.gotHeaders = headers

	if f.err != nil {
		return nil, f.err
	}

	return f.result, nil
}

func testAction() *contracts.AssignedAction {
	return &contracts.AssignedAction{
		ActionType:        contracts.ActionType_START_STEP_RUN,
		TenantId:          "tenant-1",
		TaskId:            "task-1",
		TaskRunExternalId: "run-1",
		TaskName:          "my-task",
		ActionId:          "action-1",
		RetryCount:        2,
		ActionPayload:     `{"input":{"foo":"bar"}}`,
	}
}

func TestDeliverAction_SignsBody(t *testing.T) {
	secret := "super-secret"
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 200}}

	cfg := HTTPOperatorConfig{
		TriggerEndpoint: "https://example.com/hook",
		SigningSecret:   secret,
	}

	_, err := deliverAction(context.Background(), f, nil, cfg, testAction())
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/hook", f.gotEndpoint)

	// The signature header must verify against the exact bytes that were sent.
	sig := f.gotHeaders.Get(SignatureHeader)
	require.NotEmpty(t, sig)
	assert.True(t, signature.Verify(string(f.gotBody), secret, sig), "signature must verify over the raw body")

	// The body must be the protojson serialization of the action, so it round-trips back
	// into an equivalent AssignedAction.
	var got contracts.AssignedAction
	require.NoError(t, protojson.Unmarshal(f.gotBody, &got))
	assert.Equal(t, contracts.ActionType_START_STEP_RUN, got.ActionType)
	assert.Equal(t, "tenant-1", got.TenantId)
	assert.Equal(t, "run-1", got.TaskRunExternalId)
	assert.Equal(t, int32(2), got.RetryCount)
	assert.JSONEq(t, `{"input":{"foo":"bar"}}`, got.ActionPayload)
}

func TestDeliverAction_NoSecretNoHeader(t *testing.T) {
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 200}}

	cfg := HTTPOperatorConfig{TriggerEndpoint: "https://example.com/hook"}

	_, err := deliverAction(context.Background(), f, nil, cfg, testAction())
	require.NoError(t, err)
	assert.Empty(t, f.gotHeaders.Get(SignatureHeader))
}

func TestDeliverAction_Non2xxIsError(t *testing.T) {
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 500}}

	cfg := HTTPOperatorConfig{TriggerEndpoint: "https://example.com/hook"}

	_, err := deliverAction(context.Background(), f, nil, cfg, testAction())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestDeliverAction_DeliveryErrorPropagates(t *testing.T) {
	f := &fakeSender{err: safeclient.ErrBlockedDestination}

	cfg := HTTPOperatorConfig{TriggerEndpoint: "https://10.0.0.1/hook"}

	_, err := deliverAction(context.Background(), f, nil, cfg, testAction())
	require.Error(t, err)
	assert.ErrorIs(t, err, safeclient.ErrBlockedDestination)
}

func TestDeliverAction_NoEndpoint(t *testing.T) {
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 200}}

	_, err := deliverAction(context.Background(), f, nil, HTTPOperatorConfig{}, testAction())
	require.Error(t, err)
	assert.Empty(t, f.gotEndpoint, "sender must not be called without an endpoint")
}

func TestBuildPayload_RoundTrips(t *testing.T) {
	a := testAction()

	body, err := buildPayload(a)
	require.NoError(t, err)
	assert.True(t, json.Valid(body), "payload must be valid JSON")

	var got contracts.AssignedAction
	require.NoError(t, protojson.Unmarshal(body, &got))
	assert.Equal(t, a.ActionId, got.ActionId)
	assert.Equal(t, a.ActionPayload, got.ActionPayload)
}
