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

	ep := &cachedEndpoint{id: uuid.New()}
	cfg := &endpointConfig{triggerUrl: "https://ep.example.test/trigger", secret: "test-signing-secret-0123456789abcdef", requestTimeoutSeconds: 5}

	sender.respond(cfg.triggerUrl, http.StatusOK, `{"result":42}`)

	action := startAction("svc:run")

	out := deliverAction(context.Background(), sender, ep, cfg, action)

	require.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, out.status)
	assert.Equal(t, `{"result":42}`, string(out.output))

	calls := sender.callsTo(cfg.triggerUrl)
	require.Len(t, calls, 1)

	call := calls[0]
	assert.Equal(t, http.MethodPost, call.method)
	assert.Equal(t, ep.id.String(), call.headers.Get(contract.EndpointIdHeader))
	assert.NotEmpty(t, call.headers.Get(contract.TimestampHeader))
	assert.True(t, contract.Verify(string(call.body), "test-signing-secret-0123456789abcdef", call.headers.Get(contract.SignatureHeader)))

	// The body is protojson: lowerCamelCase field names, the action nested as a message.
	assert.Contains(t, string(call.body), `"endpointId"`)

	envelope := &v1.ServerlessTriggerRequest{}
	require.NoError(t, contract.Unmarshal(call.body, envelope))
	assert.Equal(t, int32(contract.TriggerEnvelopeVersion), envelope.Version)
	assert.Equal(t, ep.id.String(), envelope.EndpointId)
	assert.NotZero(t, envelope.TimestampUnixSeconds)

	delivered := envelope.GetAction()
	require.NotNil(t, delivered)
	assert.Equal(t, action.ActionId, delivered.ActionId, "the action id is delivered as registered")
	assert.Equal(t, action.TaskRunExternalId, delivered.TaskRunExternalId)
}

func TestDeliverActionWithoutSecretFails(t *testing.T) {
	sender := newFakeSender()
	ep := &cachedEndpoint{id: uuid.New()}
	cfg := &endpointConfig{triggerUrl: "https://ep.example.test/trigger", secretErr: errors.New("could not decrypt signing secret")}

	out := deliverAction(context.Background(), sender, ep, cfg, startAction("svc:run"))

	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, out.status)
	assert.False(t, out.retry)
	assert.Empty(t, sender.callsTo(cfg.triggerUrl), "nothing is sent unsigned")
}

func TestParseHealthcheckResponse(t *testing.T) {
	legacy, err := parseHealthcheckResponse([]byte(`{"actions":["Svc:One","svc:two",""]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Empty(t, legacy.workflows)
	assert.Equal(t, []string{"svc:one", "svc:two"}, legacy.actions)

	full, err := parseHealthcheckResponse([]byte(`{
		"workflows": [{"name": "echo", "tasks": [{"readableId": "t", "action": "svc:echo"}], "unknownField": 1}],
		"actions": ["svc:extra"],
		"durable": {"supported": true},
		"runtime": {"name": "cloudflare-workers", "sdkVersion": "0.1.0"}
	}`), catalogLimits{})
	require.NoError(t, err)
	require.Len(t, full.workflows, 1)
	assert.Equal(t, "echo", full.workflows[0].Name)
	assert.Equal(t, "svc:echo", full.workflows[0].Tasks[0].Action)
	assert.Equal(t, []string{"svc:extra"}, full.actions, "a listed set is the served set, declared workflows or not")
	assert.True(t, full.durable)
	assert.Equal(t, "cloudflare-workers", full.runtime.GetName())
	assert.Equal(t, "0.1.0", full.runtime.GetSdkVersion())
	assert.Nil(t, legacy.runtime)
	assert.NotEqual(t, legacy.hash, full.hash)

	// Formatting differences do not change the hash; content does.
	same, err := parseHealthcheckResponse([]byte(`{"durable":{"supported":true},"actions":["svc:extra"],"workflows":[{"tasks":[{"action":"svc:echo","readableId":"t"}],"name":"echo"}]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, full.hash, same.hash)

	_, err = parseHealthcheckResponse([]byte(`{"workflows":[{"name":"bad","tasks":[{"action":"noverb"}]}]}`), catalogLimits{})
	assert.Error(t, err)

	_, err = parseHealthcheckResponse([]byte(`not json`), catalogLimits{})
	assert.Error(t, err)
}

// The served set takes both forms the contract allows: listed actions are the exact served
// set, so an endpoint serving some tasks of a declared workflow lists those and the others are
// not registered for it; no listed actions means every action of every declared workflow.
func TestParseHealthcheckServedActionsForms(t *testing.T) {
	pair := `{"name":"pair","tasks":[{"readableId":"first","action":"svc:first"},{"readableId":"second","action":"Svc:Second"}]}`

	subset, err := parseHealthcheckResponse([]byte(`{"workflows":[`+pair+`],"actions":["svc:first"]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:first"}, subset.actions)
	require.Len(t, subset.workflows, 1)
	assert.Equal(t, "svc:second", subset.workflows[0].Tasks[1].Action, "the workflow is registered whole")

	whole, err := parseHealthcheckResponse([]byte(`{"workflows":[`+pair+`],"actions":[]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:first", "svc:second"}, whole.actions)

	blank, err := parseHealthcheckResponse([]byte(`{"workflows":[`+pair+`],"actions":[" "]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, whole.actions, blank.actions, "blank entries do not make a listed set")

	assert.NotEqual(t, subset.hash, whole.hash)

	_, err = parseHealthcheckResponse([]byte(`{"workflows":[`+pair+`],"actions":["svc:first"],"tasks":[{"action":"svc:second","streams":true}]}`), catalogLimits{})
	require.Error(t, err, "a stream option for an action outside the served set is a catalog error")
}

// An endpoint without durable support is not registered for its durable tasks, whichever form
// named them: the served set drops every action a declared task marks durable, the on-failure
// task included, the workflow stays whole, and the support flag alone changes the hash.
func TestParseHealthcheckDropsDurableTasksWithoutSupport(t *testing.T) {
	workflows := `[{"name":"mixed","tasks":[{"readableId":"plain","action":"svc:plain"},{"readableId":"durable","action":"svc:durable","isDurable":true}],"onFailureTask":{"readableId":"cleanup","action":"svc:cleanup","isDurable":true}}]`

	unsupported, err := parseHealthcheckResponse([]byte(`{"workflows":`+workflows+`,"durable":{"supported":false}}`), catalogLimits{})
	require.NoError(t, err)
	assert.False(t, unsupported.durable)
	assert.Equal(t, []string{"svc:plain"}, unsupported.actions)
	require.Len(t, unsupported.workflows, 1)
	assert.Len(t, unsupported.workflows[0].Tasks, 2, "the workflow is registered whole")
	assert.NotNil(t, unsupported.workflows[0].OnFailureTask)

	absent, err := parseHealthcheckResponse([]byte(`{"workflows":`+workflows+`}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, unsupported.actions, absent.actions, "no durable block means no support")

	supported, err := parseHealthcheckResponse([]byte(`{"workflows":`+workflows+`,"durable":{"supported":true}}`), catalogLimits{})
	require.NoError(t, err)
	assert.True(t, supported.durable)
	assert.Equal(t, []string{"svc:cleanup", "svc:durable", "svc:plain"}, supported.actions)
	assert.NotEqual(t, unsupported.hash, supported.hash)

	listed, err := parseHealthcheckResponse([]byte(`{"workflows":`+workflows+`,"actions":["svc:durable","svc:plain"],"durable":{"supported":false}}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:plain"}, listed.actions, "a listed durable action is dropped the same way")

	_, err = parseHealthcheckResponse([]byte(`{"workflows":`+workflows+`,"durable":{"supported":false},"tasks":[{"action":"svc:durable","streams":true}]}`), catalogLimits{})
	require.Error(t, err, "a stream option for a dropped durable action names an action the endpoint does not serve")

	plainOnly, err := parseHealthcheckResponse([]byte(`{"workflows":[{"name":"plain","tasks":[{"readableId":"t","action":"svc:plain"}]}],"durable":{"supported":false}}`), catalogLimits{})
	require.NoError(t, err)
	plainOnlySupported, err := parseHealthcheckResponse([]byte(`{"workflows":[{"name":"plain","tasks":[{"readableId":"t","action":"svc:plain"}]}],"durable":{"supported":true}}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, plainOnly.actions, plainOnlySupported.actions)
	assert.NotEqual(t, plainOnly.hash, plainOnlySupported.hash, "the support flag alone is a catalog change")
}

// The catalog hash frames every element by length, so two catalogs that differ only in how
// their ids would split on a delimiter never hash alike: the pair here has one action set and
// two different stream-action sets whose newline-joined forms are the same string. The parser
// refuses such ids; the framing keeps the digest unambiguous on its own.
func TestCatalogHashFramingIsUnambiguous(t *testing.T) {
	actions := []string{"svc:a", "svc:a\nsvc:b", "svc:b"}

	twoFlags := hashCatalog(nil, actions, []string{"svc:a", "svc:b"}, false)
	oneFlag := hashCatalog(nil, actions, []string{"svc:a\nsvc:b"}, false)
	assert.NotEqual(t, twoFlags, oneFlag, "the stream-option sets differ, so the hashes must")

	assert.NotEqual(t, hashCatalog(nil, []string{"svc:a", "svc:b"}, nil, false), hashCatalog(nil, []string{"svc:a\nsvc:b"}, nil, false))
	assert.NotEqual(t, hashCatalog(nil, []string{"svc:a"}, []string{"svc:b"}, false), hashCatalog(nil, []string{"svc:a", "svc:b"}, nil, false), "a list boundary is not movable")
	assert.NotEqual(t, hashCatalog([][]byte{{1, 0}}, nil, nil, false), hashCatalog([][]byte{{1}, {}}, nil, nil, false), "a NUL inside a workflow encoding is not a boundary")
	assert.NotEqual(t, hashCatalog([][]byte{{1}}, nil, nil, false), hashCatalog(nil, []string{"\x01"}, nil, false), "sections are tagged")
	assert.Equal(t, hashCatalog(nil, actions, []string{"svc:b"}, false), hashCatalog(nil, actions, []string{"svc:b"}, false), "the digest is deterministic")
}

// Task options flag the actions invoked over the websocket: they are normalized, sorted, part
// of the hash, and must name actions the catalog serves.
func TestParseHealthcheckTaskOptions(t *testing.T) {
	plain, err := parseHealthcheckResponse([]byte(`{"actions":["svc:a","svc:b"]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Empty(t, plain.streamActions)

	flagged, err := parseHealthcheckResponse([]byte(`{"actions":["svc:a","svc:b"],"tasks":[{"action":"Svc:B","streams":true},{"action":"svc:a","streams":false}]}`), catalogLimits{})
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:b"}, flagged.streamActions)
	assert.NotEqual(t, plain.hash, flagged.hash, "the flag is part of the catalog hash")

	_, err = parseHealthcheckResponse([]byte(`{"actions":["svc:a"],"tasks":[{"action":"svc:missing","streams":true}]}`), catalogLimits{})
	assert.ErrorContains(t, err, "does not serve")
}
