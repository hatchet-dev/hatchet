//go:build !e2e && !load && !rampup && !integration

package hostgrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // OperatorService's client lives in the legacy client package
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// testJWT builds an unsigned JWT with the claims the client loader reads.
func testJWT(t *testing.T, tenantId uuid.UUID) string {
	t.Helper()

	header, _ := json.Marshal(map[string]string{"alg": "none"})
	claims, err := json.Marshal(map[string]any{
		"sub":                    tenantId.String(),
		"server_url":             "https://app.example.test",
		"grpc_broadcast_address": "grpc.example.test:443",
		"exp":                    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)

	enc := base64.RawURLEncoding

	return enc.EncodeToString(header) + "." + enc.EncodeToString(claims) + ".sig"
}

// writeFile replaces path atomically, through a temporary file and a rename, the way a
// rotated secret or a mounted file is swapped. The source's poller stats the file between
// writes, so a truncate-then-write could be observed as an empty (valid, tenantless) file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	tmp := path + ".tmp"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0o600))
	require.NoError(t, os.Rename(tmp, path))
}

func TestLocalExchangeLoadsAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.yaml")

	tenantA := uuid.New()
	tenantB := uuid.New()

	writeFile(t, filepath.Join(dir, "b.token"), "  token-b\n")
	writeFile(t, path, "tenants:\n  "+tenantA.String()+": { token: token-a }\n  "+tenantB.String()+": { token_file: b.token }\n")

	ex, err := NewLocalExchange(path, WithPollInterval(10*time.Millisecond))
	require.NoError(t, err)
	defer ex.Close()

	tok, err := ex.Token(context.Background(), tenantA)
	require.NoError(t, err)
	assert.Equal(t, "token-a", tok)

	tok, err = ex.Token(context.Background(), tenantB)
	require.NoError(t, err)
	assert.Equal(t, "token-b", tok, "token_file is read relative to the yaml and trimmed")

	_, err = ex.Token(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNoToken)

	// Rewriting the file with a newer mtime is picked up by the poller.
	writeFile(t, path, "tenants:\n  "+tenantA.String()+": { token: token-a2 }\n")
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(path, future, future))

	require.Eventually(t, func() bool {
		tok, err := ex.Token(context.Background(), tenantA)
		return err == nil && tok == "token-a2"
	}, 2*time.Second, 5*time.Millisecond)

	_, err = ex.Token(context.Background(), tenantB)
	assert.ErrorIs(t, err, ErrNoToken, "a tenant removed from the file is no longer served")

	// A broken rewrite keeps the previous mapping.
	writeFile(t, path, "tenants:\n  not-a-uuid: { token: x }\n")
	later := future.Add(2 * time.Second)
	require.NoError(t, os.Chtimes(path, later, later))

	time.Sleep(50 * time.Millisecond)

	tok, err = ex.Token(context.Background(), tenantA)
	require.NoError(t, err)
	assert.Equal(t, "token-a2", tok)
}

func TestLocalExchangeRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := NewLocalExchange(filepath.Join(dir, "missing.yaml"))
	assert.Error(t, err)

	path := filepath.Join(dir, "empty-entry.yaml")
	writeFile(t, path, "tenants:\n  "+uuid.New().String()+": {}\n")

	_, err = NewLocalExchange(path)
	assert.Error(t, err, "an entry needs token or token_file")
}

// Rotating the contents of a referenced token_file, without touching the YAML, is observed by
// the source's poller.
func TestLocalExchangeObservesReferencedTokenFile(t *testing.T) {
	dir := t.TempDir()
	tenant := uuid.New()
	yamlPath := filepath.Join(dir, "tenants.yaml")
	tokenPath := filepath.Join(dir, "tenant.token")

	require.NoError(t, os.WriteFile(tokenPath, []byte("old-token"), 0o600))
	require.NoError(t, os.WriteFile(yamlPath, []byte("tenants:\n  "+tenant.String()+": { token_file: tenant.token }\n"), 0o600))

	e, err := NewLocalExchange(yamlPath, WithPollInterval(time.Hour))
	require.NoError(t, err)
	defer e.Close()

	// Filesystems may round mtimes; make the rotation land on a later second.
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, os.WriteFile(tokenPath, []byte("new-token"), 0o600))

	changed, err := e.reloadIfChanged()
	require.NoError(t, err)
	assert.True(t, changed, "referenced token file changed but the YAML did not; the rotation was not observed")

	tok, err := e.Token(context.Background(), tenant)
	require.NoError(t, err)
	assert.Equal(t, "new-token", tok)

	// Nothing changed: no reload.
	changed, err = e.reloadIfChanged()
	require.NoError(t, err)
	assert.False(t, changed)
}

func TestStaticExchange(t *testing.T) {
	tenant := uuid.New()
	tok := testJWT(t, tenant)

	ex, err := NewStaticExchange(tok)
	require.NoError(t, err)
	assert.Equal(t, tenant, ex.TenantId())

	got, err := ex.Token(context.Background(), tenant)
	require.NoError(t, err)
	assert.Equal(t, tok, got)

	_, err = ex.Token(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNoToken)

	_, err = NewStaticExchange("not.a.jwt")
	assert.Error(t, err)
}

// fakeSession is the minimal OperatorSession the host needs. Durable listeners are built over
// a fakeDurableStream (durable_test.go) and stopped on Close like the real session does.
// Action deltas and flushes are recorded in order; flushErr, when set, fails the next Flush.
// actions is what Actions hands the host's deliver loop; the channels close on Close.
type fakeSession struct {
	client.OperatorSession
	stream    *fakeDurableStream
	listeners []*client.DurableTaskListener
	workerId  string
	tenantId  string
	added     [][]string
	removed   [][]string
	puts      []*v1.CreateWorkflowVersionRequest
	events    []*contracts.StepActionEvent
	flushErr  error
	flushes   int
	pauses    int
	closed    bool

	actions chan *contracts.AssignedAction
	errs    chan error
	mu      sync.Mutex
}

func newFakeSession(workerId, tenantId string) *fakeSession {
	return &fakeSession{
		workerId: workerId,
		tenantId: tenantId,
		actions:  make(chan *contracts.AssignedAction),
		errs:     make(chan error, 1),
	}
}

func (f *fakeSession) Registration() client.OperatorRegistration {
	return client.OperatorRegistration{WorkerId: f.workerId, TenantId: f.tenantId, OperatorId: uuid.NewString()}
}

func (f *fakeSession) Actions(context.Context) (<-chan *contracts.AssignedAction, <-chan error, error) {
	return f.actions, f.errs, nil
}

func (f *fakeSession) AddActions(ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.added = append(f.added, append([]string{}, ids...))
}

func (f *fakeSession) RemoveActions(ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.removed = append(f.removed, append([]string{}, ids...))
}

func (f *fakeSession) Flush(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.flushes++

	if f.flushErr != nil {
		err := f.flushErr
		f.flushErr = nil

		return err
	}

	return nil
}

func (f *fakeSession) PutWorkflow(_ context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, []string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.puts = append(f.puts, wf)

	actions := make([]string, 0, len(wf.Tasks))

	for _, task := range wf.Tasks {
		actions = append(actions, task.Action)
	}

	return &v1.CreateWorkflowVersionResponse{Id: "v"}, actions, nil
}

func (f *fakeSession) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, ev)

	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeSession) Pause(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pauses++

	return nil
}

func (f *fakeSession) NewDurableTaskListener(opts ...client.DurableTaskListenerOpt) *client.DurableTaskListener {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.stream == nil {
		f.stream = newFakeDurableStream()
	}

	stream := f.stream

	listener := client.NewDurableTaskListener(f.workerId, func(context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
		return stream, nil
	}, nil, opts...)

	f.listeners = append(f.listeners, listener)

	return listener
}

func (f *fakeSession) Close(_ ...client.CloseOpt) error {
	f.mu.Lock()

	if f.closed {
		f.mu.Unlock()
		return nil
	}

	f.closed = true
	listeners := f.listeners
	f.mu.Unlock()

	close(f.actions)
	close(f.errs)

	for _, l := range listeners {
		l.Stop()
	}

	return nil
}

func (f *fakeSession) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closed
}

func (f *fakeSession) sentEvents() []*contracts.StepActionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*contracts.StepActionEvent(nil), f.events...)
}

// fakeOperatorClient connects sessions the engine authenticated as tenantId.
type fakeOperatorClient struct {
	connectErr error
	flushErr   error
	tenantId   string
	requests   []*client.ConnectOperatorRequest
	sessions   []*fakeSession
}

func (f *fakeOperatorClient) Connect(_ context.Context, req *client.ConnectOperatorRequest) (client.OperatorSession, error) {
	f.requests = append(f.requests, req)

	if f.connectErr != nil {
		err := f.connectErr
		f.connectErr = nil

		return nil, err
	}

	s := newFakeSession(uuid.NewString(), f.tenantId)
	s.flushErr = f.flushErr
	f.flushErr = nil
	f.sessions = append(f.sessions, s)

	return s, nil
}

type fakeClient struct {
	client.Client
	operator *fakeOperatorClient
	token    string
}

func (f *fakeClient) Operator() client.OperatorClient {
	return f.operator
}

func (f *fakeClient) Close() error { return nil }

type mapSource map[uuid.UUID]string

func (m mapSource) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	tok, ok := m[tenantId]

	if !ok {
		return "", ErrNoToken
	}

	return tok, nil
}

// recordingHandler records the actions it is handled and can refuse them.
type recordingHandler struct {
	mu      sync.Mutex
	actions []*contracts.AssignedAction
	err     error
}

func (h *recordingHandler) HandleAction(_ context.Context, action *contracts.AssignedAction) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.actions = append(h.actions, action)

	return h.err
}

func (h *recordingHandler) handled() []*contracts.AssignedAction {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]*contracts.AssignedAction(nil), h.actions...)
}

func identity(tenant uuid.UUID) operator.Identity {
	return operator.Identity{TenantId: tenant, Name: "serverless"}
}

func TestOpenCachesClientPerTenant(t *testing.T) {
	tenant := uuid.New()
	source := mapSource{tenant: "tok-1"}

	var built []*fakeClient

	host := New(source, Options{
		NewClient: func(token string) (client.Client, error) {
			c := &fakeClient{token: token, operator: &fakeOperatorClient{tenantId: tenant.String()}}
			built = append(built, c)

			return c, nil
		},
	})

	opts := operator.OpenOpts{Handler: &recordingHandler{}, Actions: []string{"ns_svc:run"}, SlotConfig: map[string]int32{"default": 1}, Labels: map[string]interface{}{"k": "v"}}

	s, err := host.Open(context.Background(), identity(tenant), opts)
	require.NoError(t, err)
	assert.Equal(t, built[0].operator.sessions[0].workerId, s.Registration().WorkerId.String())
	assert.Equal(t, tenant, s.Registration().TenantId)

	s2, err := host.Open(context.Background(), identity(tenant), opts)
	require.NoError(t, err)

	require.Len(t, built, 1, "one client per tenant")
	require.Len(t, built[0].operator.requests, 2)

	req := built[0].operator.requests[1]
	assert.Equal(t, "serverless", req.Name)
	assert.Equal(t, opts.SlotConfig, req.SlotConfig)
	assert.Equal(t, "v", req.Labels["k"])
	assert.Len(t, req.Labels, 1, "only the caller's labels are sent")

	// the initial action set is streamed and flushed before Open returns
	session := built[0].operator.sessions[1]
	assert.Equal(t, [][]string{opts.Actions}, session.added)
	assert.Equal(t, 1, session.flushes)

	// deltas and puts pass straight through to the session
	require.NoError(t, s2.AddActions(context.Background(), []string{"ns_svc:other"}))
	require.NoError(t, s2.RemoveActions(context.Background(), []string{"ns_svc:run"}))
	require.NoError(t, s2.Flush(context.Background()))
	assert.Equal(t, [][]string{opts.Actions, {"ns_svc:other"}}, session.added)
	assert.Equal(t, [][]string{{"ns_svc:run"}}, session.removed)
	assert.Equal(t, 2, session.flushes)

	wf := &v1.CreateWorkflowVersionRequest{Name: "ns_wf", Tasks: []*v1.CreateTaskOpts{{ReadableId: "t", Action: "ns_svc:put"}}}
	derived, err := s2.PutWorkflow(context.Background(), wf)
	require.NoError(t, err)
	assert.Equal(t, []string{"ns_svc:put"}, derived)
	require.Len(t, session.puts, 1)
	assert.Same(t, wf, session.puts[0])
	assert.Equal(t, 2, session.flushes, "a put does not touch the action set")

	require.NoError(t, s2.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{TaskRunExternalId: "r"}))
	require.Len(t, session.sentEvents(), 1)

	require.NoError(t, s2.Pause(context.Background()))
	assert.Equal(t, 1, session.pauses)

	ch, err := s.OpenDurable(context.Background(), uuid.New(), 0)
	require.NoError(t, err)
	require.NoError(t, ch.Close())

	require.NoError(t, s2.Close(context.Background()))
	assert.True(t, session.closed)
	require.NoError(t, s2.Close(context.Background()), "Close is idempotent")
	assert.ErrorIs(t, s2.AddActions(context.Background(), []string{"x"}), operator.ErrSessionClosed)
	assert.ErrorIs(t, s2.Pause(context.Background()), operator.ErrSessionClosed)

	_, err = s2.OpenDurable(context.Background(), uuid.New(), 0)
	assert.ErrorIs(t, err, operator.ErrSessionClosed)

	require.NoError(t, s.Close(context.Background()))

	// A rotated token rebuilds the client; a released tenant is evicted.
	source[tenant] = "tok-2"

	s3, err := host.Open(context.Background(), identity(tenant), opts)
	require.NoError(t, err)
	require.Len(t, built, 2)
	assert.Equal(t, "tok-2", built[1].token)
	require.NoError(t, s3.Close(context.Background()))

	host.ReleaseTenant(tenant)

	s4, err := host.Open(context.Background(), identity(tenant), opts)
	require.NoError(t, err)
	assert.Len(t, built, 3)
	require.NoError(t, s4.Close(context.Background()))

	_, err = host.Open(context.Background(), identity(uuid.New()), opts)
	assert.ErrorIs(t, err, ErrNoToken)
	assert.Len(t, built, 3, "no client is built without a token")
}

func TestOpenRejects(t *testing.T) {
	tenant := uuid.New()
	connects := 0

	host := New(mapSource{tenant: "tok"}, Options{NewClient: func(string) (client.Client, error) {
		connects++
		return &fakeClient{operator: &fakeOperatorClient{tenantId: tenant.String()}}, nil
	}})

	handler := &recordingHandler{}
	operatorId := uuid.New()
	workerId := uuid.New()

	_, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{})
	require.Error(t, err, "a handler is required")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant, OperatorId: &operatorId}, operator.OpenOpts{Handler: handler})
	assert.ErrorIs(t, err, operator.ErrNotSupported, "an existing row cannot be opened by id")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant, Name: "dag", Kind: sqlcv1.V1OperatorKindDAG}, operator.OpenOpts{Handler: handler})
	assert.ErrorIs(t, err, operator.ErrNotSupported, "only GRPC rows are registered")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant}, operator.OpenOpts{Handler: handler})
	require.Error(t, err, "a name is required")

	_, err = host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: handler, ResumeWorkerId: &workerId})
	assert.ErrorIs(t, err, operator.ErrNotSupported)

	_, err = host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: handler, WorkerName: "replica-1"})
	assert.ErrorIs(t, err, operator.ErrNotSupported)

	assert.Zero(t, connects, "nothing is refused after a client is built")

	s, err := host.Open(context.Background(), operator.Identity{TenantId: tenant, Name: "ok", Kind: sqlcv1.V1OperatorKindGRPC}, operator.OpenOpts{Handler: handler})
	require.NoError(t, err)
	require.NoError(t, s.Close(context.Background()))
}

func TestOpenRetriesOnceOnUnauthenticated(t *testing.T) {
	tenant := uuid.New()
	source := mapSource{tenant: "tok"}

	var built []*fakeClient

	host := New(source, Options{
		NewClient: func(token string) (client.Client, error) {
			op := &fakeOperatorClient{tenantId: tenant.String()}

			if len(built) == 0 {
				op.connectErr = status.Error(codes.Unauthenticated, "expired")
			}

			c := &fakeClient{token: token, operator: op}
			built = append(built, c)

			return c, nil
		},
	})

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, built, 2, "the cached client is dropped and rebuilt after Unauthenticated")
	require.NoError(t, s.Close(context.Background()))

	// Other errors are not retried.
	host2 := New(source, Options{
		NewClient: func(token string) (client.Client, error) {
			return &fakeClient{operator: &fakeOperatorClient{connectErr: errors.New("boom")}}, nil
		},
	})

	_, err = host2.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	assert.Error(t, err)
}

// A session whose initial actions the engine refused is closed rather than handed to the
// caller with an empty action set; an empty initial set is not flushed at all.
func TestOpenFlushesInitialActions(t *testing.T) {
	tenant := uuid.New()
	source := mapSource{tenant: "tok"}

	op := &fakeOperatorClient{tenantId: tenant.String(), flushErr: errors.New("invalid action")}

	host := New(source, Options{
		NewClient: func(token string) (client.Client, error) {
			return &fakeClient{token: token, operator: op}, nil
		},
	})

	_, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}, Actions: []string{"bad"}})
	require.ErrorContains(t, err, "invalid action")
	require.Len(t, op.sessions, 1)
	assert.True(t, op.sessions[0].isClosed(), "the session is closed when the initial flush fails")

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	assert.Empty(t, op.sessions[1].added)
	assert.Equal(t, 0, op.sessions[1].flushes)
	require.NoError(t, s.Close(context.Background()))
}

// Assigned actions reach the handler from the session's deliver loop; a refused start is
// reported as a retryable failure, since there is no requeue over gRPC.
func TestOpenDeliversActionsToHandler(t *testing.T) {
	tenant := uuid.New()
	op := &fakeOperatorClient{tenantId: tenant.String()}

	host := New(mapSource{tenant: "tok"}, Options{NewClient: func(string) (client.Client, error) {
		return &fakeClient{operator: op}, nil
	}})

	handler := &recordingHandler{}

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: handler})
	require.NoError(t, err)

	session := op.sessions[0]

	start := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: "run-1", ActionId: "svc:run", RetryCount: 1}
	session.actions <- start

	require.Eventually(t, func() bool { return len(handler.handled()) == 1 }, time.Second, time.Millisecond)
	assert.Same(t, start, handler.handled()[0])
	assert.Empty(t, session.sentEvents(), "an accepted action is the handler's to report")

	handler.mu.Lock()
	handler.err = errors.New("no capacity")
	handler.mu.Unlock()

	refused := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: "run-2", ActionId: "svc:run", RetryCount: 2}
	session.actions <- refused

	require.Eventually(t, func() bool { return len(session.sentEvents()) == 1 }, time.Second, time.Millisecond)

	ev := session.sentEvents()[0]
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev.EventType)
	assert.Equal(t, "run-2", ev.TaskRunExternalId)
	assert.Equal(t, "no capacity", ev.EventPayload)
	assert.Nil(t, ev.ShouldNotRetry)
	require.NotNil(t, ev.RetryCount)
	assert.Equal(t, int32(2), *ev.RetryCount)

	// a refused cancel has nothing to report
	session.actions <- &contracts.AssignedAction{ActionType: contracts.ActionType_CANCEL_STEP_RUN, TaskRunExternalId: "run-1"}
	require.Eventually(t, func() bool { return len(handler.handled()) == 3 }, time.Second, time.Millisecond)
	assert.Len(t, session.sentEvents(), 1)

	require.NoError(t, s.Close(context.Background()))
	assert.True(t, session.isClosed())
}

// closableClient is a fake engine client that owns a goroutine, as a gRPC connection does,
// and stops it on Close.
type closableClient struct {
	fakeClient
	closed atomic.Bool
	stop   chan struct{}
}

func newClosableClient(operator *fakeOperatorClient) *closableClient {
	c := &closableClient{fakeClient: fakeClient{operator: operator}, stop: make(chan struct{})}

	go func() { <-c.stop }()

	return c
}

func (c *closableClient) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		close(c.stop)
	}

	return nil
}

// Evicting a tenant's cached client closes it: on release, on token rotation, on an
// authentication failure and on the host's Close. The goroutine the client owns must go with
// it.
func TestEvictedClientsAreClosed(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	tenant := uuid.New()
	source := mapSource{tenant: "tok-1"}

	var built []*closableClient

	host := New(source, Options{NewClient: func(string) (client.Client, error) {
		c := newClosableClient(&fakeOperatorClient{tenantId: tenant.String()})
		built = append(built, c)

		return c, nil
	}})

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	require.NoError(t, s.Close(context.Background()))
	require.Len(t, built, 1)

	// Token rotation replaces the client and closes the previous one.
	source[tenant] = "tok-2"
	s, err = host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	require.NoError(t, s.Close(context.Background()))
	require.Len(t, built, 2)
	assert.True(t, built[0].closed.Load(), "the replaced client is closed")
	assert.False(t, built[1].closed.Load())

	// Releasing the tenant closes the current client.
	host.ReleaseTenant(tenant)
	assert.True(t, built[1].closed.Load(), "the released client is closed")
	assert.Empty(t, host.clients)

	// Closing the host closes whatever is cached.
	s, err = host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	require.NoError(t, s.Close(context.Background()))
	require.Len(t, built, 3)

	host.Close()
	assert.True(t, built[2].closed.Load())
	assert.Empty(t, host.clients)
}

// The default factory builds real SDK clients over gRPC connections; releasing a tenant must
// leave none of their goroutines behind.
func TestReleasedRealClientsLeakNoGoroutines(t *testing.T) {
	host := New(nil, Options{})

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	for i := 0; i < 3; i++ {
		tenant := uuid.New()
		_, err := host.clientFor(tenant, testJWT(t, tenant))
		require.NoError(t, err)
		host.ReleaseTenant(tenant)
	}

	assert.Empty(t, host.clients)
}

// tenantSession reports the tenant the engine authenticated, which may differ from the one
// the host asked the source for.
type tenantSession struct {
	*fakeSession
	tenantId string
}

func (s *tenantSession) Registration() client.OperatorRegistration {
	return client.OperatorRegistration{TenantId: s.tenantId, WorkerId: s.workerId, OperatorId: uuid.NewString()}
}

type tenantOperator struct{ session *tenantSession }

func (o *tenantOperator) Connect(context.Context, *client.ConnectOperatorRequest) (client.OperatorSession, error) {
	return o.session, nil
}

type tenantClient struct {
	client.Client
	operator *tenantOperator
}

func (c *tenantClient) Operator() client.OperatorClient { return c.operator }

func (c *tenantClient) Close() error { return nil }

// A session the engine authenticated as another tenant than the one the identity names is
// refused before any action, workflow or delivery touches it, and so is a token whose tenant
// claim names another tenant.
func TestOpenRefusesMismatchedTenant(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	session := &tenantSession{fakeSession: newFakeSession(uuid.NewString(), b.String()), tenantId: b.String()}
	c := &tenantClient{operator: &tenantOperator{session: session}}

	host := New(mapSource{a: testJWT(t, a)}, Options{NewClient: func(string) (client.Client, error) { return c, nil }})

	_, err := host.Open(context.Background(), identity(a), operator.OpenOpts{Handler: &recordingHandler{}, Actions: []string{"review:run"}})
	require.Error(t, err, "the engine authenticated tenant B for a session of tenant A")
	assert.Contains(t, err.Error(), b.String())
	assert.Empty(t, session.added, "no actions were sent on the mismatched session")
	assert.True(t, session.isClosed(), "the mismatched session is closed")

	// A token whose claim names another tenant is refused before connecting.
	session2 := &tenantSession{fakeSession: newFakeSession(uuid.NewString(), a.String()), tenantId: a.String()}
	connects := 0
	host2 := New(mapSource{a: testJWT(t, b)}, Options{NewClient: func(string) (client.Client, error) {
		connects++
		return &tenantClient{operator: &tenantOperator{session: session2}}, nil
	}})

	_, err = host2.Open(context.Background(), identity(a), operator.OpenOpts{Handler: &recordingHandler{}})
	require.Error(t, err, "the source returned tenant B's token for tenant A")
	assert.Equal(t, 0, connects, "no client is built for a token of the wrong tenant")
}
