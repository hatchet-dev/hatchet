package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// fakeEngine is an in-memory Engine for handler tests.
type fakeEngine struct {
	tenantID string

	triggeredWorkflow string
	triggeredInput    map[string]any
	triggeredMeta     map[string]any
	triggerRunID      string
	triggerErr        error

	runs map[string]*rest.V1WorkflowRunDetails

	replayedRunID string
	replayErr     error

	workers []rest.Worker
	events  []rest.V1TaskEvent

	workflowNames map[string]string

	meta    *rest.APIMeta
	metaErr error
	version string
}

func (f *fakeEngine) TenantID() string { return f.tenantID }

func (f *fakeEngine) TriggerWorkflow(_ context.Context, workflow string, input map[string]any, meta map[string]any) (string, error) {
	f.triggeredWorkflow = workflow
	f.triggeredInput = input
	f.triggeredMeta = meta
	return f.triggerRunID, f.triggerErr
}

func (f *fakeEngine) GetRun(_ context.Context, runID string) (*rest.V1WorkflowRunDetails, error) {
	if details, ok := f.runs[runID]; ok {
		return details, nil
	}
	return nil, fmt.Errorf("run %s not found (status 404)", runID)
}

func (f *fakeEngine) ListTaskEvents(_ context.Context, _ string) ([]rest.V1TaskEvent, error) {
	return f.events, nil
}

func (f *fakeEngine) ListWorkers(_ context.Context) ([]rest.Worker, error) {
	return f.workers, nil
}

func (f *fakeEngine) ReplayRun(_ context.Context, runID string) error {
	f.replayedRunID = runID
	return f.replayErr
}

func (f *fakeEngine) WorkflowName(_ context.Context, workflowID string) (string, error) {
	if name, ok := f.workflowNames[workflowID]; ok {
		return name, nil
	}
	return "", fmt.Errorf("workflow %s not found", workflowID)
}

func (f *fakeEngine) Meta(_ context.Context) (*rest.APIMeta, error) {
	if f.metaErr != nil {
		return nil, f.metaErr
	}
	if f.meta == nil {
		return &rest.APIMeta{}, nil
	}
	return f.meta, nil
}

func (f *fakeEngine) Version(_ context.Context) (string, error) {
	return f.version, nil
}

// fakeFeedbackSender records feedback sends.
type fakeFeedbackSender struct {
	target     FeedbackTarget
	distinctID string
	event      FeedbackEvent
	sent       bool
	err        error
}

func (f *fakeFeedbackSender) Send(_ context.Context, target FeedbackTarget, distinctID string, event FeedbackEvent) error {
	f.target = target
	f.distinctID = distinctID
	f.event = event
	f.sent = f.err == nil
	return f.err
}

func runDetails(runID string, status rest.V1TaskStatus, output map[string]any, errMsg string) *rest.V1WorkflowRunDetails {
	details := &rest.V1WorkflowRunDetails{
		Run: rest.V1WorkflowRun{
			Metadata:    rest.APIResourceMeta{Id: runID},
			DisplayName: "test-run",
			Status:      status,
			WorkflowId:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Output:      output,
		},
	}
	if errMsg != "" {
		details.Run.ErrorMessage = &errMsg
	}
	return details
}

// newTestServer builds a Server with a fake engine, one granted profile
// ("local", the default) and one ungranted profile ("prod").
func newTestServer(t *testing.T, engine *fakeEngine, sender FeedbackSender, granted ...string) *Server {
	t.Helper()

	store := NewGrantStore(t.TempDir())
	require.NoError(t, store.Save(grantsFor(granted...)))

	return NewServer(Deps{
		Version:  "test",
		Profiles: testProfileSource("local", "local", "prod"),
		Grants:   store,
		NewEngine: func(profile *cliconfig.Profile) (Engine, error) {
			return engine, nil
		},
		DetectEmbedded: func(ctx context.Context) *EmbeddedDetection { return &EmbeddedDetection{} },
		Feedback:       sender,
	})
}

func resultText(t *testing.T, res *mcpsdk.CallToolResult) string {
	t.Helper()
	require.NotNil(t, res)
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok)
	return text.Text
}

func TestHandleTriggerRunWaitsForCompletion(t *testing.T) {
	runID := uuid.NewString()
	engine := &fakeEngine{
		tenantID:     "tenant-local",
		triggerRunID: runID,
		runs: map[string]*rest.V1WorkflowRunDetails{
			runID: runDetails(runID, rest.V1TaskStatusCOMPLETED, map[string]any{"result": "ok"}, ""),
		},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleTriggerRun(context.Background(), nil, triggerRunArgs{
		Workflow: "simple",
		Input:    map[string]any{"message": "hi"},
	})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, "profile: local (tenant: tenant-local)", "every result names the profile/tenant it ran against")
	assert.Contains(t, text, runID)
	assert.Contains(t, text, "COMPLETED")
	assert.Contains(t, text, `"result": "ok"`)
	assert.Equal(t, "simple", engine.triggeredWorkflow)
	assert.Equal(t, map[string]any{"message": "hi"}, engine.triggeredInput)
}

func TestHandleTriggerRunNoWait(t *testing.T) {
	engine := &fakeEngine{tenantID: "tenant-local", triggerRunID: "run-1"}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	wait := false
	res, _, err := server.handleTriggerRun(context.Background(), nil, triggerRunArgs{
		Workflow: "simple",
		Wait:     &wait,
	})
	require.NoError(t, err)
	assert.Contains(t, resultText(t, res), "TRIGGERED")
}

func TestHandleTriggerRunDeniedProfile(t *testing.T) {
	engine := &fakeEngine{tenantID: "tenant-local"}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	_, _, err := server.handleTriggerRun(context.Background(), nil, triggerRunArgs{
		Profile:  "prod",
		Workflow: "simple",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `profile "prod" is not authorized for MCP use`)
	assert.Contains(t, err.Error(), "hatchet mcp auth")
	assert.Empty(t, engine.triggeredWorkflow, "no engine call may happen for a denied profile")
}

func TestHandleGetRun(t *testing.T) {
	runID := uuid.NewString()
	engine := &fakeEngine{
		tenantID: "tenant-local",
		runs: map[string]*rest.V1WorkflowRunDetails{
			runID: runDetails(runID, rest.V1TaskStatusFAILED, nil, "boom"),
		},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleGetRun(context.Background(), nil, getRunArgs{RunID: runID})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, "FAILED")
	assert.Contains(t, text, "boom")
}

func TestHandleListRunEventsFallsBackToTaskEvents(t *testing.T) {
	taskName := "step1"
	engine := &fakeEngine{
		tenantID: "tenant-local",
		events: []rest.V1TaskEvent{
			{EventType: rest.V1TaskEventTypeSTARTED, Message: "started", Timestamp: time.Now(), TaskDisplayName: &taskName},
		},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleListRunEvents(context.Background(), nil, listRunEventsArgs{RunID: uuid.NewString()})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, "STARTED")
	assert.Contains(t, text, "step1")
}

func TestHandleListWorkers(t *testing.T) {
	status := rest.ACTIVE
	engine := &fakeEngine{
		tenantID: "tenant-local",
		workers: []rest.Worker{
			{Name: "worker-1", Status: &status, Metadata: rest.APIResourceMeta{Id: "w1"}},
		},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleListWorkers(context.Background(), nil, listWorkersArgs{})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, "worker-1")
	assert.Contains(t, text, "ACTIVE")
	assert.Contains(t, text, `"count": 1`)
}

func TestHandleReplayRunInPlace(t *testing.T) {
	runID := uuid.NewString()
	engine := &fakeEngine{
		tenantID: "tenant-local",
		runs: map[string]*rest.V1WorkflowRunDetails{
			runID: runDetails(runID, rest.V1TaskStatusFAILED, nil, "boom"),
		},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	wait := false
	res, _, err := server.handleReplayRun(context.Background(), nil, replayRunArgs{RunID: runID, Wait: &wait})
	require.NoError(t, err)
	assert.Equal(t, runID, engine.replayedRunID)
	assert.Contains(t, resultText(t, res), "REPLAYED")
}

func TestHandleReplayRunWithNewInputTriggersFreshRun(t *testing.T) {
	oldRunID := uuid.NewString()
	newRunID := uuid.NewString()
	engine := &fakeEngine{
		tenantID:     "tenant-local",
		triggerRunID: newRunID,
		runs: map[string]*rest.V1WorkflowRunDetails{
			oldRunID: runDetails(oldRunID, rest.V1TaskStatusFAILED, nil, "boom"),
			newRunID: runDetails(newRunID, rest.V1TaskStatusCOMPLETED, map[string]any{"fixed": true}, ""),
		},
		workflowNames: map[string]string{"11111111-1111-1111-1111-111111111111": "simple"},
	}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleReplayRun(context.Background(), nil, replayRunArgs{
		RunID:    oldRunID,
		NewInput: map[string]any{"message": "retry"},
	})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, newRunID)
	assert.Contains(t, text, "COMPLETED")
	assert.Contains(t, text, oldRunID, "result references the run being replayed")
	assert.Equal(t, "simple", engine.triggeredWorkflow)
	assert.Equal(t, map[string]any{"message": "retry"}, engine.triggeredInput)
	assert.Empty(t, engine.replayedRunID, "new_input must not use the in-place replay API")
}

func TestHandleEngineStatusHidesUngrantedProfiles(t *testing.T) {
	engine := &fakeEngine{tenantID: "tenant-local", version: "v1.2.3"}
	server := newTestServer(t, engine, &fakeFeedbackSender{}, "local")

	res, _, err := server.handleEngineStatus(context.Background(), nil, engineStatusArgs{})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, `"local"`)
	assert.NotContains(t, text, "prod", "ungranted profile names must not leak into agent context")
	assert.Contains(t, text, `"reachable": true`)
	assert.Contains(t, text, "v1.2.3")
	assert.Contains(t, text, `"embeddedDetected": false`)
}

func TestHandleEngineStatusNothingGranted(t *testing.T) {
	engine := &fakeEngine{tenantID: "tenant-local"}
	server := newTestServer(t, engine, &fakeFeedbackSender{}) // no grants

	res, _, err := server.handleEngineStatus(context.Background(), nil, engineStatusArgs{})
	require.NoError(t, err, "engine_status degrades gracefully instead of erroring")

	text := resultText(t, res)
	assert.Contains(t, text, `"selected": null`)
	assert.Contains(t, text, "hatchet mcp auth")
	assert.NotContains(t, text, "prod")
	assert.NotContains(t, text, `"local"`)
}

// TestStaleEmbeddedRegistrationIsRedacted is the regression test for the
// stale-registration note: a rejected registration's stored URL (which can
// carry internal hosts, paths, or credentials) must never reach the agent,
// neither via engine_status nor via resolution errors.
func TestStaleEmbeddedRegistrationIsRedacted(t *testing.T) {
	staleURL := "http://review-user:synthetic-secret@127.0.0.1:1/private-path"
	server := newTestServer(t, &fakeEngine{tenantID: "t"}, &fakeFeedbackSender{})
	server.deps.Profiles = storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded(staleURL),
	})
	server.deps.DetectEmbedded = func(ctx context.Context) *EmbeddedDetection {
		return detectEmbedded(ctx, server.deps.Profiles)
	}

	res, _, err := server.handleEngineStatus(context.Background(), nil, engineStatusArgs{})
	require.NoError(t, err)

	text := resultText(t, res)
	assert.Contains(t, text, "stale embedded registration")
	assert.NotContains(t, text, "synthetic-secret")
	assert.NotContains(t, text, "127.0.0.1:1")
	assert.NotContains(t, text, "private-path")

	_, _, err = server.handleGetRun(context.Background(), nil, getRunArgs{Profile: EmbeddedProfileName, RunID: "unused"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stale embedded registration")
	assert.NotContains(t, err.Error(), "synthetic-secret")
	assert.NotContains(t, err.Error(), "127.0.0.1:1")
	assert.NotContains(t, err.Error(), "private-path")
}

// clearPosthogKey empties the baked-in capture key for the test's duration so
// the engine-fallback and no-key paths can be exercised deterministically.
func clearPosthogKey(t *testing.T) {
	t.Helper()

	orig := PosthogAPIKey
	PosthogAPIKey = ""
	t.Cleanup(func() { PosthogAPIKey = orig })
}

func TestHandleSubmitFeedbackUsesBakedKey(t *testing.T) {
	require.NotEmpty(t, PosthogAPIKey, "a public capture key should be baked into the build")

	engineKey := "phc_from_engine"
	engineHost := "http://engine-chosen.example.invalid"
	engine := &fakeEngine{
		tenantID: "tenant-local",
		meta:     &rest.APIMeta{Posthog: &rest.APIMetaPosthog{ApiKey: &engineKey, ApiHost: &engineHost}},
	}
	sender := &fakeFeedbackSender{}
	server := newTestServer(t, engine, sender, "local")

	_, _, err := server.handleSubmitFeedback(context.Background(), nil, submitFeedbackArgs{
		Category: "bug",
		Summary:  "x",
		Detail:   "y",
	})
	require.NoError(t, err)

	assert.True(t, sender.sent)
	assert.Equal(t, PosthogAPIKey, sender.target.APIKey, "the baked-in key is always used, never the engine-served key")
	assert.Equal(t, PosthogEndpoint, sender.target.Endpoint, "the pinned endpoint is always used, never the engine-served host")
}

func TestHandleSubmitFeedback(t *testing.T) {
	engine := &fakeEngine{tenantID: "tenant-local"}
	sender := &fakeFeedbackSender{}
	server := newTestServer(t, engine, sender, "local")

	res, _, err := server.handleSubmitFeedback(context.Background(), nil, submitFeedbackArgs{
		Category: "docs-gap",
		Summary:  "missing replay docs",
		Detail:   "could not find how to replay with new input",
		Context:  "verifying a workflow fix",
	})
	require.NoError(t, err)

	assert.True(t, sender.sent)
	assert.Equal(t, PosthogAPIKey, sender.target.APIKey)
	assert.Equal(t, PosthogEndpoint, sender.target.Endpoint)
	assert.Equal(t, "docs-gap", sender.event.Category)
	assert.Equal(t, "missing replay docs", sender.event.Summary)
	assert.Equal(t, "self-hosted", sender.event.DeploymentType)
	assert.Equal(t, "test", sender.event.CLIVersion)
	assert.Contains(t, resultText(t, res), `"sent": true`)
}

// TestHandleSubmitFeedbackEmptyKeyNeverUsesEngineTarget is the regression
// test for feedback egress pinning: with no build-time capture key, an
// engine's metadata must not be able to route the feedback event to an
// engine-chosen key or host. The tool reports "not sent" instead.
func TestHandleSubmitFeedbackEmptyKeyNeverUsesEngineTarget(t *testing.T) {
	clearPosthogKey(t)

	requests := make(chan string, 4)
	engineChosen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer engineChosen.Close()

	engineKey := "phc_from_engine"
	engineHost := engineChosen.URL
	engine := &fakeEngine{
		tenantID: "tenant-local",
		meta:     &rest.APIMeta{Posthog: &rest.APIMetaPosthog{ApiKey: &engineKey, ApiHost: &engineHost}},
	}
	server := newTestServer(t, engine, NewHTTPFeedbackSender(), "local")

	_, _, err := server.handleSubmitFeedback(context.Background(), nil, submitFeedbackArgs{
		Category: "bug",
		Summary:  "x",
		Detail:   "y",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no analytics key")
	assert.Empty(t, requests, "the engine-selected destination must never receive the feedback event")
}

func TestHandleSubmitFeedbackInvalidCategory(t *testing.T) {
	sender := &fakeFeedbackSender{}
	server := newTestServer(t, &fakeEngine{tenantID: "t"}, sender, "local")

	_, _, err := server.handleSubmitFeedback(context.Background(), nil, submitFeedbackArgs{
		Category: "rant",
		Summary:  "x",
		Detail:   "y",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid category")
	assert.False(t, sender.sent)
}

func TestHandleSubmitFeedbackNoKeyAvailable(t *testing.T) {
	clearPosthogKey(t)

	sender := &fakeFeedbackSender{}
	server := newTestServer(t, &fakeEngine{tenantID: "t"}, sender, "local")

	_, _, err := server.handleSubmitFeedback(context.Background(), nil, submitFeedbackArgs{
		Category: "bug",
		Summary:  "x",
		Detail:   "y",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no analytics key")
	assert.False(t, sender.sent)
}

func TestClassifyDeployment(t *testing.T) {
	cloudProfile := &resolvedProfile{Name: "cloud", Profile: &cliconfig.Profile{ApiServerURL: "https://tenant.onhatchet.run"}}
	assert.Equal(t, "cloud", classifyDeployment(cloudProfile, nil))

	selfHosted := &resolvedProfile{Name: "sh", Profile: &cliconfig.Profile{ApiServerURL: "https://hatchet.internal.example.com"}}
	assert.Equal(t, "self-hosted", classifyDeployment(selfHosted, nil))

	embeddedTrue := true
	assert.Equal(t, "embedded", classifyDeployment(selfHosted, &rest.APIMeta{Embedded: &embeddedTrue}))

	embeddedRP := &resolvedProfile{Name: EmbeddedProfileName, Profile: &cliconfig.Profile{ApiServerURL: "http://localhost:28243"}, Embedded: true}
	assert.Equal(t, "embedded", classifyDeployment(embeddedRP, nil))
}
