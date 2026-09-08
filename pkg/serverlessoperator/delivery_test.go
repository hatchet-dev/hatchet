//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

func TestClassifyResponse(t *testing.T) {
	completed := contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED
	failed := contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED

	tests := []struct {
		name   string
		body   string
		output string
		errMsg string
		code   int
		status contracts.StepActionEventType
		retry  bool
	}{
		{name: "200 json", code: 200, body: `{"ok":true}`, status: completed, output: `{"ok":true}`},
		{name: "200 empty", code: 200, body: ``, status: completed, output: `{}`},
		{name: "201 json", code: 201, body: `[1]`, status: completed, output: `[1]`},
		{name: "204", code: 204, body: `ignored`, status: completed, output: `{}`},
		{name: "200 non-json", code: 200, body: `<html>`, status: failed, retry: false, errMsg: "endpoint returned status 200 with a non-JSON body"},
		{name: "302 redirect", code: 302, body: ``, status: failed, retry: false, errMsg: "endpoint returned redirect status 302; redirects are not followed"},
		{name: "400", code: 400, body: ``, status: failed, retry: false, errMsg: "endpoint returned status 400 Bad Request"},
		{name: "404 with error", code: 404, body: `{"error":"no such task"}`, status: failed, retry: false, errMsg: "no such task"},
		{name: "408", code: 408, body: ``, status: failed, retry: true, errMsg: "endpoint returned status 408 Request Timeout"},
		{name: "425", code: 425, body: ``, status: failed, retry: true, errMsg: "endpoint returned status 425 Too Early"},
		{name: "429", code: 429, body: ``, status: failed, retry: true, errMsg: "endpoint returned status 429 Too Many Requests"},
		{name: "500", code: 500, body: `not json`, status: failed, retry: true, errMsg: "endpoint returned status 500 Internal Server Error"},
		{name: "503 with error", code: 503, body: `{"error":"warming up"}`, status: failed, retry: true, errMsg: "warming up"},
		{name: "500 retry override false", code: 500, body: `{"error":"bad input","retry":false}`, status: failed, retry: false, errMsg: "bad input"},
		{name: "422 retry override true", code: 422, body: `{"error":"rate limited upstream","retry":true}`, status: failed, retry: true, errMsg: "rate limited upstream"},
		{name: "400 retry override without error", code: 400, body: `{"retry":true}`, status: failed, retry: true, errMsg: "endpoint returned status 400 Bad Request"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := classifyResponse(&safeclient.DeliveryResult{StatusCode: tc.code, BodyPrefix: []byte(tc.body)})

			assert.Equal(t, tc.status, out.status)
			assert.Equal(t, tc.retry, out.retry)

			if tc.status == completed {
				assert.Equal(t, tc.output, string(out.output))
			} else {
				assert.Equal(t, tc.errMsg, out.errMsg)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	out := classifyError(cancelled, context.Canceled, time.Second)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, out.status)

	out = classifyError(context.Background(), context.DeadlineExceeded, 7*time.Second)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, out.status)
	assert.True(t, out.retry)
	assert.Equal(t, "request timed out after 7s", out.errMsg)

	for _, err := range []error{safeclient.ErrBlockedDestination, safeclient.ErrBadScheme, safeclient.ErrBadPort, safeclient.ErrResponseTooLarge} {
		out = classifyError(context.Background(), err, time.Second)
		assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, out.status)
		assert.False(t, out.retry, err.Error())
	}

	out = classifyError(context.Background(), errors.New("connection reset"), time.Second)
	assert.True(t, out.retry)
	assert.Equal(t, "request failed: connection reset", out.errMsg)
}

func TestDeliverActionSignsEnvelope(t *testing.T) {
	sender := newFakeSender()

	ep := &cachedEndpoint{id: uuid.New(), namespace: uuid.New()}
	cfg := &endpointConfig{triggerUrl: "https://ep.example.test/trigger", secret: "s3cret", requestTimeoutSeconds: 5}

	sender.respond(cfg.triggerUrl, http.StatusOK, `{"result":42}`)

	action := startAction(ep.namespace, "svc:run")

	out := deliverAction(context.Background(), sender, ep, cfg, action)

	require.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, out.status)
	assert.Equal(t, `{"result":42}`, string(out.output))

	calls := sender.callsTo(cfg.triggerUrl)
	require.Len(t, calls, 1)

	call := calls[0]
	assert.Equal(t, http.MethodPost, call.method)
	assert.Equal(t, ep.id.String(), call.headers.Get(contract.EndpointIdHeader))
	assert.NotEmpty(t, call.headers.Get(contract.TimestampHeader))
	assert.True(t, signature.Verify(string(call.body), "s3cret", call.headers.Get(contract.SignatureHeader)))

	// The body is protojson: lowerCamelCase field names, the action nested as a message.
	assert.Contains(t, string(call.body), `"endpointId"`)

	envelope := &v1.ServerlessTriggerRequest{}
	require.NoError(t, contract.Unmarshal(call.body, envelope))
	assert.Equal(t, int32(contract.TriggerEnvelopeVersion), envelope.Version)
	assert.Equal(t, ep.id.String(), envelope.EndpointId)
	assert.Equal(t, ep.namespace.String(), envelope.Namespace)
	assert.NotZero(t, envelope.Timestamp)

	delivered := envelope.GetAction()
	require.NotNil(t, delivered)
	assert.Equal(t, action.ActionId, delivered.ActionId, "the action id is delivered with its namespace prefix")
	assert.Equal(t, action.TaskRunExternalId, delivered.TaskRunExternalId)
}

func TestDeliverActionWithoutSecretFails(t *testing.T) {
	sender := newFakeSender()
	ep := &cachedEndpoint{id: uuid.New(), namespace: uuid.New()}
	cfg := &endpointConfig{triggerUrl: "https://ep.example.test/trigger", secretErr: errors.New("could not decrypt signing secret")}

	out := deliverAction(context.Background(), sender, ep, cfg, startAction(ep.namespace, "svc:run"))

	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, out.status)
	assert.False(t, out.retry)
	assert.Empty(t, sender.callsTo(cfg.triggerUrl), "nothing is sent unsigned")
}

func TestParseHealthcheckResponse(t *testing.T) {
	ns := uuid.New()

	legacy, err := parseHealthcheckResponse([]byte(`{"actions":["Svc:One","svc:two",""]}`), ns, catalogLimits{})
	require.NoError(t, err)
	assert.Empty(t, legacy.workflows)
	assert.Equal(t, []string{prefixed(ns, "svc:one"), prefixed(ns, "svc:two")}, legacy.actions)

	full, err := parseHealthcheckResponse([]byte(`{
		"workflows": [{"name": "echo", "tasks": [{"readableId": "t", "action": "svc:echo"}], "unknownField": 1}],
		"actions": ["svc:extra"],
		"durable": {"supported": true},
		"runtime": {"name": "cloudflare-workers", "sdkVersion": "0.1.0"}
	}`), ns, catalogLimits{})
	require.NoError(t, err)
	require.Len(t, full.workflows, 1)
	assert.Equal(t, prefixed(ns, "echo"), full.workflows[0].Name)
	assert.Equal(t, prefixed(ns, "svc:echo"), full.workflows[0].Tasks[0].Action)
	assert.Equal(t, []string{prefixed(ns, "svc:echo"), prefixed(ns, "svc:extra")}, full.actions)
	assert.True(t, full.durable)
	assert.Equal(t, "cloudflare-workers", full.runtime.GetName())
	assert.Equal(t, "0.1.0", full.runtime.GetSdkVersion())
	assert.Nil(t, legacy.runtime)
	assert.NotEqual(t, legacy.hash, full.hash)

	// Formatting differences do not change the hash; content does.
	same, err := parseHealthcheckResponse([]byte(`{"actions":["svc:extra"],"workflows":[{"tasks":[{"action":"svc:echo","readableId":"t"}],"name":"echo"}]}`), ns, catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, full.hash, same.hash)

	_, err = parseHealthcheckResponse([]byte(`{"workflows":[{"name":"bad","tasks":[{"action":"noverb"}]}]}`), ns, catalogLimits{})
	assert.Error(t, err)

	_, err = parseHealthcheckResponse([]byte(`not json`), ns, catalogLimits{})
	assert.Error(t, err)
}
