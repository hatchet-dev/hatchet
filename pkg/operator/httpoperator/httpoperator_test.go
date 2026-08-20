//go:build !e2e && !load && !rampup && !integration

package httpoperator

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeTaskEventWriter captures every reported step action event.
type fakeTaskEventWriter struct {
	events []*contracts.StepActionEvent
}

func (f *fakeTaskEventWriter) SendStepActionEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.events = append(f.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeTaskEventWriter) RegisterDurableTask(_ context.Context, _ uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error) {
	return nil, nil, nil
}

func (f *fakeTaskEventWriter) CancelTaskEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.events = append(f.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeTaskEventWriter) TriggerDAGStep(_ context.Context, _ uuid.UUID, _ *operator.DAGStepTriggerRequest) (*operator.DAGStepTriggerResult, error) {
	return nil, nil
}

func (f *fakeTaskEventWriter) CancelDAGChildren(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

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

// newTestHTTPOperator builds an HTTPOperator whose shared state is wired to a fake sender and
// fake event writer, without going through NewHTTPOperator (which would start a real
// healthcheck-polling goroutine and require a real safeclient sender). configJSON is the raw
// HTTPOperatorConfig JSON stored on the operator row.
func newTestHTTPOperator(t *testing.T, workerId uuid.UUID, sender requestSender, writer operator.TaskEventWriter, configJSON string) *HTTPOperator {
	t.Helper()

	l := zerolog.Nop()

	shared, err := operator.NewSharedOperator(&sqlcv1.V1Operator{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Config:   []byte(configJSON),
	}, &l, nil, writer, workerId, HTTPOperatorConfig{})
	require.NoError(t, err)

	return &HTTPOperator{
		SharedOperator: shared,
		sender:         sender,
	}
}

func TestHandleAction_CancelStepRun_ReportsCancelledWithoutDelivering(t *testing.T) {
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 200}}
	writer := &fakeTaskEventWriter{}
	workerId := uuid.New()

	h := newTestHTTPOperator(t, workerId, f, writer, `{"triggerEndpoint":"https://example.com/hook"}`)

	action := testAction()
	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	err := h.HandleAction(context.Background(), action)
	require.NoError(t, err)

	assert.Empty(t, f.gotEndpoint, "cancelling a task must not deliver the HTTP request")

	require.Len(t, writer.events, 1, "cancelling a task must report exactly one step action event")
	got := writer.events[0]
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, got.EventType)
	assert.Equal(t, action.TaskRunExternalId, got.TaskRunExternalId)
	assert.Equal(t, action.TaskId, got.TaskId)
	assert.Equal(t, workerId.String(), got.WorkerId)
}

func TestHandleAction_StartStepRun_StillDelivers(t *testing.T) {
	f := &fakeSender{result: &safeclient.DeliveryResult{StatusCode: 200}}
	writer := &fakeTaskEventWriter{}
	workerId := uuid.New()

	h := newTestHTTPOperator(t, workerId, f, writer, `{"triggerEndpoint":"https://example.com/hook"}`)

	action := testAction()

	err := h.HandleAction(context.Background(), action)
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/hook", f.gotEndpoint, "starting a task must still deliver the HTTP request")

	var sawStarted bool
	for _, e := range writer.events {
		if e.EventType == contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED {
			sawStarted = true
		}
		assert.NotEqual(t, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, e.EventType, "starting a task must not report cancelled")
	}
	assert.True(t, sawStarted, "starting a task must report started")
}
