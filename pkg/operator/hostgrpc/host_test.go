//go:build !e2e && !load && !rampup && !integration

package hostgrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // the default client factory builds this client; the loopback tests exercise it
	"github.com/hatchet-dev/hatchet/pkg/client/operatorclient"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
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

// fakeSession is the minimal operatorclient.Session the host needs. The durable task stream
// is one fakeDurableStream (durable_test.go), created up front and handed out on every open.
// Action deltas and flushes are recorded in order; flushErr, when set, fails the next Flush.
// actions is what Actions hands the host's deliver loop; the channels close on Close.
type fakeSession struct {
	operatorclient.Session
	stream   *fakeDurableStream
	workerId string
	tenantId string
	added    [][]string
	removed  [][]string
	puts     []*v1.CreateWorkflowVersionRequest
	events   []*contracts.StepActionEvent
	flushErr error
	flushes  int
	pauses   int
	closed   bool

	actions chan *contracts.AssignedAction
	errs    chan error
	mu      sync.Mutex
}

func newFakeSession(workerId, tenantId string) *fakeSession {
	return &fakeSession{
		workerId: workerId,
		tenantId: tenantId,
		stream:   newFakeDurableStream(),
		actions:  make(chan *contracts.AssignedAction),
		errs:     make(chan error, 1),
	}
}

func (f *fakeSession) Registration() operatorclient.Registration {
	f.mu.Lock()
	defer f.mu.Unlock()

	return operatorclient.Registration{WorkerId: f.workerId, TenantId: f.tenantId, OperatorId: uuid.NewString()}
}

// setWorker replaces the registered worker, as a fresh reconnect of the client does.
func (f *fakeSession) setWorker(workerId string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.workerId = workerId
}

// fail ends the session the way a terminal stream failure does: the error is reported and
// the channels close.
func (f *fakeSession) fail(err error) {
	f.mu.Lock()

	if f.closed {
		f.mu.Unlock()
		return
	}

	f.closed = true
	f.mu.Unlock()

	f.errs <- err
	close(f.errs)
	close(f.actions)
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

func (f *fakeSession) OpenDurableTaskStream(context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
	return f.stream, nil
}

func (f *fakeSession) Close(_ ...operatorclient.CloseOpt) error {
	f.mu.Lock()

	if f.closed {
		f.mu.Unlock()
		return nil
	}

	f.closed = true
	f.mu.Unlock()

	close(f.actions)
	close(f.errs)

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
	requests   []*operatorclient.ConnectRequest
	sessions   []*fakeSession
	mu         sync.Mutex
}

func (f *fakeOperatorClient) Connect(_ context.Context, req *operatorclient.ConnectRequest) (operatorclient.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

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

// session returns the i-th session connected, waiting for it to appear.
func (f *fakeOperatorClient) session(t *testing.T, i int) *fakeSession {
	t.Helper()

	require.Eventually(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()

		return len(f.sessions) > i
	}, eventually, time.Millisecond, "session %d was never connected", i)

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.sessions[i]
}

func (f *fakeOperatorClient) sessionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.sessions)
}

type fakeClient struct {
	engineClient
	operator *fakeOperatorClient
	token    string
}

func (f *fakeClient) Operator() operatorclient.Client {
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

// lockedSource is a token source safe to change while a session supervises itself; it counts
// the lookups made.
type lockedSource struct {
	mu      sync.Mutex
	tokens  map[uuid.UUID]string
	lookups int
}

func newLockedSource(tenant uuid.UUID, token string) *lockedSource {
	return &lockedSource{tokens: map[uuid.UUID]string{tenant: token}}
}

func (s *lockedSource) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lookups++

	tok, ok := s.tokens[tenantId]

	if !ok {
		return "", ErrNoToken
	}

	return tok, nil
}

func (s *lockedSource) set(tenant uuid.UUID, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token == "" {
		delete(s.tokens, tenant)
		return
	}

	s.tokens[tenant] = token
}

func (s *lockedSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lookups
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

// newTestHost builds a host over source and factory; a nil factory keeps the default.
func newTestHost(t *testing.T, source TokenSource, factory ClientFactory) *Host {
	t.Helper()

	opts := []Opt{WithTokenSource(source)}

	if factory != nil {
		opts = append(opts, WithClientFactory(factory))
	}

	host, err := New(opts...)
	require.NoError(t, err)

	return host
}

func TestNewRequiresTokenSource(t *testing.T) {
	_, err := New()
	require.ErrorContains(t, err, "WithTokenSource")

	_, err = New(WithTokenSource(mapSource{}), WithLogger(nil))
	require.ErrorContains(t, err, "WithLogger")
}

func TestOpenCachesClientPerTenant(t *testing.T) {
	tenant := uuid.New()
	source := mapSource{tenant: "tok-1"}

	var built []*fakeClient

	host := newTestHost(t, source, func(token string) (engineClient, error) {
		c := &fakeClient{token: token, operator: &fakeOperatorClient{tenantId: tenant.String()}}
		built = append(built, c)

		return c, nil
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

	host := newTestHost(t, mapSource{tenant: "tok"}, func(string) (engineClient, error) {
		connects++
		return &fakeClient{operator: &fakeOperatorClient{tenantId: tenant.String()}}, nil
	})

	handler := &recordingHandler{}
	operatorId := uuid.New()
	workerId := uuid.New()

	_, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{})
	require.Error(t, err, "a handler is required")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant, OperatorId: &operatorId}, operator.OpenOpts{Handler: handler})
	assert.ErrorIs(t, err, operator.ErrNotSupported, "an existing row cannot be opened by id")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant, Name: "dag", Kind: sqlcv1.V1OperatorKindDAG}, operator.OpenOpts{Handler: handler})
	assert.ErrorIs(t, err, operator.ErrNotSupported, "only GRPC rows are registered")

	_, err = host.Open(context.Background(), operator.Identity{TenantId: tenant, Name: "managed", Leasing: sqlcv1.V1OperatorLeasingMANAGED}, operator.OpenOpts{Handler: handler})
	assert.ErrorIs(t, err, operator.ErrNotSupported, "only self-leased rows are registered over the wire")

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

	host := newTestHost(t, source, func(token string) (engineClient, error) {
		op := &fakeOperatorClient{tenantId: tenant.String()}

		if len(built) == 0 {
			op.connectErr = status.Error(codes.Unauthenticated, "expired")
		}

		c := &fakeClient{token: token, operator: op}
		built = append(built, c)

		return c, nil
	})

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, built, 2, "the cached client is dropped and rebuilt after Unauthenticated")
	require.NoError(t, s.Close(context.Background()))

	// Other errors are not retried.
	host2 := newTestHost(t, source, func(token string) (engineClient, error) {
		return &fakeClient{operator: &fakeOperatorClient{connectErr: errors.New("boom")}}, nil
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

	host := newTestHost(t, source, func(token string) (engineClient, error) {
		return &fakeClient{token: token, operator: op}, nil
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

	host := newTestHost(t, mapSource{tenant: "tok"}, func(string) (engineClient, error) {
		return &fakeClient{operator: op}, nil
	})

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

	host := newTestHost(t, source, func(string) (engineClient, error) {
		c := newClosableClient(&fakeOperatorClient{tenantId: tenant.String()})
		built = append(built, c)

		return c, nil
	})

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
	host := newTestHost(t, mapSource{}, nil)

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	for i := 0; i < 3; i++ {
		tenant := uuid.New()
		_, release, err := host.clientFor(tenant, testJWT(t, tenant))
		require.NoError(t, err)
		release()
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

func (s *tenantSession) Registration() operatorclient.Registration {
	return operatorclient.Registration{TenantId: s.tenantId, WorkerId: s.workerId, OperatorId: uuid.NewString()}
}

type tenantOperator struct{ session *tenantSession }

func (o *tenantOperator) Connect(context.Context, *operatorclient.ConnectRequest) (operatorclient.Session, error) {
	return o.session, nil
}

type tenantClient struct {
	engineClient
	operator *tenantOperator
}

func (c *tenantClient) Operator() operatorclient.Client { return c.operator }

func (c *tenantClient) Close() error { return nil }

// A session the engine authenticated as another tenant than the one the identity names is
// refused before any action, workflow or delivery touches it, and so is a token whose tenant
// claim names another tenant.
func TestOpenRefusesMismatchedTenant(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	session := &tenantSession{fakeSession: newFakeSession(uuid.NewString(), b.String()), tenantId: b.String()}
	c := &tenantClient{operator: &tenantOperator{session: session}}

	host := newTestHost(t, mapSource{a: testJWT(t, a)}, func(string) (engineClient, error) { return c, nil })

	_, err := host.Open(context.Background(), identity(a), operator.OpenOpts{Handler: &recordingHandler{}, Actions: []string{"review:run"}})
	require.Error(t, err, "the engine authenticated tenant B for a session of tenant A")
	assert.Contains(t, err.Error(), b.String())
	assert.Empty(t, session.added, "no actions were sent on the mismatched session")
	assert.True(t, session.isClosed(), "the mismatched session is closed")

	// A token whose claim names another tenant is refused before connecting.
	session2 := &tenantSession{fakeSession: newFakeSession(uuid.NewString(), a.String()), tenantId: a.String()}
	connects := 0
	host2 := newTestHost(t, mapSource{a: testJWT(t, b)}, func(string) (engineClient, error) {
		connects++
		return &tenantClient{operator: &tenantOperator{session: session2}}, nil
	})

	_, err = host2.Open(context.Background(), identity(a), operator.OpenOpts{Handler: &recordingHandler{}})
	require.Error(t, err, "the source returned tenant B's token for tenant A")
	assert.Equal(t, 0, connects, "no client is built for a token of the wrong tenant")
}

// A released tenant's client stays open while a session still holds it: the session opened
// again while an older one drains must not lose its connection to the older one's teardown.
func TestReleaseTenantWaitsForOpenSessions(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	tenant := uuid.New()
	source := mapSource{tenant: "tok-1"}

	var built []*closableClient

	host := newTestHost(t, source, func(string) (engineClient, error) {
		c := newClosableClient(&fakeOperatorClient{tenantId: tenant.String()})
		built = append(built, c)

		return c, nil
	})

	s, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)

	host.ReleaseTenant(tenant)
	assert.False(t, built[0].closed.Load(), "a released client with an open session stays open")
	assert.Empty(t, host.clients, "the released client is no longer handed out")

	// a session opened after the release gets a client of its own
	s2, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	require.Len(t, built, 2)

	require.NoError(t, s.Close(context.Background()))
	assert.True(t, built[0].closed.Load(), "the last session's Close closes the released client")
	assert.False(t, built[1].closed.Load())

	// a rotated token keeps the previous client alive until its session closes
	source[tenant] = "tok-2"

	s3, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)
	require.Len(t, built, 3)
	assert.False(t, built[1].closed.Load(), "the replaced client is held by its session")

	require.NoError(t, s2.Close(context.Background()))
	assert.True(t, built[1].closed.Load())

	require.NoError(t, s3.Close(context.Background()))
	assert.False(t, built[2].closed.Load(), "the current client stays cached for the next Open")

	host.Close()
	assert.True(t, built[2].closed.Load())
}

// A client session that ends with a terminal error is replaced: the host asks the source for
// the token again, the action set the session advertised is restored on the new client session
// and delivery resumes.
func TestSessionReopensAfterTerminalFailure(t *testing.T) {
	tenant := uuid.New()
	source := newLockedSource(tenant, "tok")
	op := &fakeOperatorClient{tenantId: tenant.String()}

	host := newTestHost(t, source, func(string) (engineClient, error) {
		return &fakeClient{operator: op}, nil
	})

	handler := &recordingHandler{}

	opened, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: handler, Actions: []string{"svc:a", "svc:b"}})
	require.NoError(t, err)

	s := opened.(*session)
	s.backoff = func(context.Context, int) error { return nil }

	require.NoError(t, s.AddActions(context.Background(), []string{"svc:c"}))
	require.NoError(t, s.RemoveActions(context.Background(), []string{"svc:b"}))

	first := op.session(t, 0)
	first.fail(status.Error(codes.Unauthenticated, "token revoked"))

	second := op.session(t, 1)
	require.Eventually(t, func() bool {
		second.mu.Lock()
		defer second.mu.Unlock()

		return second.flushes == 1
	}, eventually, time.Millisecond, "the action set was not restored on the new session")

	second.mu.Lock()
	require.Len(t, second.added, 1)
	assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, second.added[0], "the desired set, as adds and removes left it")
	second.mu.Unlock()

	assert.Equal(t, 2, source.count(), "the source was asked for the token again")
	assert.Equal(t, second.workerId, s.Registration().WorkerId.String())

	start := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: "run-1", ActionId: "svc:a"}
	second.actions <- start

	require.Eventually(t, func() bool { return len(handler.handled()) == 1 }, eventually, time.Millisecond)
	assert.Same(t, start, handler.handled()[0])

	select {
	case <-s.Done():
		t.Fatal("a recovered session is not done")
	default:
	}

	require.NoError(t, s.Close(context.Background()))
	assert.True(t, second.isClosed())

	<-s.Done()
	assert.NoError(t, s.Err())
}

// A session whose tenant lost its token gives up: Done closes, Err says why and the client
// reference is released.
func TestSessionGivesUpWithoutToken(t *testing.T) {
	tenant := uuid.New()
	source := newLockedSource(tenant, "tok")
	op := &fakeOperatorClient{tenantId: tenant.String()}

	var built []*closableClient

	host := newTestHost(t, source, func(string) (engineClient, error) {
		c := newClosableClient(op)
		built = append(built, c)

		return c, nil
	})

	opened, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)

	s := opened.(*session)
	s.backoff = func(context.Context, int) error { return nil }

	source.set(tenant, "")
	op.session(t, 0).fail(status.Error(codes.Unauthenticated, "token revoked"))

	select {
	case <-s.Done():
	case <-time.After(eventually):
		t.Fatal("the session did not give up")
	}

	assert.ErrorIs(t, s.Err(), ErrNoToken)
	assert.Equal(t, 1, op.sessionCount(), "nothing was connected without a token")

	host.ReleaseTenant(tenant)
	assert.True(t, built[0].closed.Load(), "the session released its client reference")

	require.NoError(t, s.Close(context.Background()))
	assert.ErrorIs(t, s.Err(), ErrNoToken, "Close keeps the reason")
}

// Close during a reconnect backoff returns promptly.
func TestSessionCloseDuringBackoff(t *testing.T) {
	tenant := uuid.New()
	op := &fakeOperatorClient{tenantId: tenant.String(), connectErr: nil}

	host := newTestHost(t, newLockedSource(tenant, "tok"), func(string) (engineClient, error) {
		return &fakeClient{operator: op}, nil
	})

	opened, err := host.Open(context.Background(), identity(tenant), operator.OpenOpts{Handler: &recordingHandler{}})
	require.NoError(t, err)

	s := opened.(*session)
	inBackoff := make(chan struct{})

	s.backoff = func(ctx context.Context, _ int) error {
		close(inBackoff)
		<-ctx.Done()

		return ctx.Err()
	}

	// the first reconnect fails with a transient error, so supervision backs off
	op.mu.Lock()
	op.connectErr = errors.New("engine unreachable")
	op.mu.Unlock()

	op.session(t, 0).fail(errors.New("stream lost"))

	select {
	case <-inBackoff:
	case <-time.After(eventually):
		t.Fatal("supervision never backed off")
	}

	closed := make(chan error, 1)
	go func() { closed <- s.Close(context.Background()) }()

	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Close did not return during the backoff")
	}

	<-s.Done()
	assert.NoError(t, s.Err())
}

// Reports go out under the worker the client currently is: after the client registered a
// fresh worker, a report the operator stamped with the previous worker is rewritten, while a
// worker that was never the session's passes through.
func TestSessionReportsUnderCurrentWorker(t *testing.T) {
	s, fs := openTestSession(t)

	first := fs.workerId
	start := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: "run-1", ActionId: "svc:run"}
	fs.actions <- start

	require.Eventually(t, func() bool { return len(s.handler.(*recordingHandler).handled()) == 1 }, eventually, time.Millisecond)

	second := uuid.NewString()
	fs.setWorker(second)

	assert.Equal(t, second, s.Registration().WorkerId.String(), "the registration follows the client")

	require.NoError(t, s.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{WorkerId: first, TaskRunExternalId: "run-1", EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED}))
	require.NoError(t, s.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{TaskRunExternalId: "run-2", EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED}))

	foreign := uuid.NewString()
	require.NoError(t, s.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{WorkerId: foreign, TaskRunExternalId: "run-3"}))

	events := fs.sentEvents()
	require.Len(t, events, 3)
	assert.Equal(t, second, events[0].WorkerId, "the previous worker is rewritten to the current one")
	assert.Equal(t, second, events[1].WorkerId, "an empty worker is the current one")
	assert.Equal(t, foreign, events[2].WorkerId, "a foreign worker is the engine's to refuse")

	s.mu.Lock()
	_, recorded := s.attempts["run-1"]
	s.mu.Unlock()
	assert.False(t, recorded, "a terminal report drops the attempt")
}

// blockingHandler blocks every start until released and records cancels at once.
type blockingHandler struct {
	recordingHandler
	release chan struct{}
}

func (h *blockingHandler) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	if err := h.recordingHandler.HandleAction(ctx, action); err != nil {
		return err
	}

	if action.ActionType == contracts.ActionType_START_STEP_RUN {
		select {
		case <-h.release:
		case <-ctx.Done():
		}
	}

	return nil
}

// A start that blocks the handler holds up neither the cancels behind it nor the reader: a
// cancel reaches the handler at once, later starts wait in a bounded queue, and a start past
// the queue is refused with a retryable failure instead of being held.
func TestCancelsBypassBlockedStarts(t *testing.T) {
	fs := newFakeSession(uuid.NewString(), uuid.NewString())
	reg, err := parseRegistration(fs.Registration())
	require.NoError(t, err)

	handler := &blockingHandler{release: make(chan struct{})}
	s := newSession(fs, reg, handler, newNopLogger())
	s.startQueueSize = 1
	require.NoError(t, s.startDelivery())

	t.Cleanup(func() {
		_ = s.Close(context.Background())
		fs.stream.end()
	})

	start := func(run string) *contracts.AssignedAction {
		return &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: run, ActionId: "svc:run"}
	}

	fs.actions <- start("run-1")
	require.Eventually(t, func() bool { return len(handler.handled()) == 1 }, eventually, time.Millisecond, "the first start was not handed on")

	fs.actions <- start("run-2")
	fs.actions <- &contracts.AssignedAction{ActionType: contracts.ActionType_CANCEL_STEP_RUN, TaskRunExternalId: "run-1"}

	require.Eventually(t, func() bool { return len(handler.handled()) == 2 }, eventually, time.Millisecond, "the cancel waited behind the blocked start")
	assert.Equal(t, contracts.ActionType_CANCEL_STEP_RUN, handler.handled()[1].ActionType)

	fs.actions <- start("run-3")

	require.Eventually(t, func() bool { return len(fs.sentEvents()) == 1 }, eventually, time.Millisecond, "the start past the queue was not refused")
	refused := fs.sentEvents()[0]
	assert.Equal(t, "run-3", refused.TaskRunExternalId)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, refused.EventType)
	assert.Contains(t, refused.EventPayload, "start queue")
	assert.Nil(t, refused.ShouldNotRetry)

	assert.Len(t, handler.handled(), 2, "the queued start waits for the handler")

	close(handler.release)

	require.Eventually(t, func() bool { return len(handler.handled()) == 3 }, eventually, time.Millisecond, "the queued start did not run once the handler was free")
	assert.Equal(t, "run-2", handler.handled()[2].TaskRunExternalId)
}

// loopbackEngine is an OperatorService over a real listener for the real client: Register
// accepts the token it is told to, hands out the first worker to the first registration and
// the next worker after that, and Listen acknowledges deltas and pauses and delivers what the
// test pushes. Reports are refused for a worker that is not the current one, as the engine
// refuses a worker it deleted.
type loopbackEngine struct {
	v1.UnimplementedOperatorServiceServer

	tenant, op, first, next string

	mu            sync.Mutex
	accept        string
	registrations int
	events        []*contracts.StepActionEvent
	streams       []chan *v1.OperatorListenResponse
	dropFirst     chan struct{}
	dropOnce      sync.Once
}

func newLoopbackEngine(tenant uuid.UUID, token string) *loopbackEngine {
	return &loopbackEngine{tenant: tenant.String(), op: uuid.NewString(), first: uuid.NewString(), next: uuid.NewString(), accept: token, dropFirst: make(chan struct{})}
}

func (e *loopbackEngine) token(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)

	if values := md.Get("authorization"); len(values) > 0 {
		return strings.TrimPrefix(values[0], "Bearer ")
	}

	return ""
}

func (e *loopbackEngine) setAccepted(token string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.accept = token
}

func (e *loopbackEngine) Register(ctx context.Context, _ *v1.OperatorRegisterRequest) (*v1.OperatorRegisterResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.token(ctx) != e.accept {
		return nil, status.Error(codes.Unauthenticated, "token revoked")
	}

	e.registrations++

	worker := e.first

	if e.registrations > 1 {
		worker = e.next
	}

	return &v1.OperatorRegisterResponse{TenantId: e.tenant, OperatorId: e.op, WorkerId: worker, Resumed: false}, nil
}

func (e *loopbackEngine) Listen(stream v1.OperatorService_ListenServer) error {
	req, err := stream.Recv()

	if err != nil {
		return err
	}

	start := req.GetStart()

	if start == nil {
		return status.Error(codes.InvalidArgument, "start first")
	}

	deliveries := make(chan *v1.OperatorListenResponse, 16)

	e.mu.Lock()
	e.streams = append(e.streams, deliveries)
	e.mu.Unlock()

	incoming := make(chan *v1.OperatorListenRequest)
	recvErr := make(chan error, 1)

	go func() {
		for {
			req, err := stream.Recv()

			if err != nil {
				recvErr <- err
				return
			}

			select {
			case incoming <- req:
			case <-stream.Context().Done():
				return
			}
		}
	}()

	for {
		var drop chan struct{}

		if start.WorkerId == e.first {
			drop = e.dropFirst
		}

		select {
		case <-drop:
			return status.Error(codes.Unavailable, "forced reconnect")
		case err := <-recvErr:
			return err
		case resp := <-deliveries:
			if err := stream.Send(resp); err != nil {
				return err
			}
		case req := <-incoming:
			var resp *v1.OperatorListenResponse

			switch msg := req.Message.(type) {
			case *v1.OperatorListenRequest_Actions:
				resp = &v1.OperatorListenResponse{Message: &v1.OperatorListenResponse_Ack{Ack: &v1.OperatorActionsAck{Sequence: msg.Actions.Sequence}}}
			case *v1.OperatorListenRequest_Pause:
				resp = &v1.OperatorListenResponse{Message: &v1.OperatorListenResponse_PauseAck{PauseAck: &v1.OperatorPauseAck{Paused: msg.Pause.Paused}}}
			default:
				continue
			}

			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

func (e *loopbackEngine) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	current := e.first

	if e.registrations > 1 {
		current = e.next
	}

	if ev.WorkerId != current {
		return nil, status.Error(codes.PermissionDenied, "worker no longer exists")
	}

	e.events = append(e.events, ev)

	return &contracts.ActionEventResponse{WorkerId: ev.WorkerId}, nil
}

// dropFirstStream ends the first worker's Listen stream with a retryable error.
func (e *loopbackEngine) dropFirstStream() {
	e.dropOnce.Do(func() { close(e.dropFirst) })
}

func (e *loopbackEngine) registrationCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.registrations
}

// stream returns the i-th Listen stream's delivery channel, waiting for it to open.
func (e *loopbackEngine) stream(t *testing.T, i int) chan *v1.OperatorListenResponse {
	t.Helper()

	require.Eventually(t, func() bool {
		e.mu.Lock()
		defer e.mu.Unlock()

		return len(e.streams) > i
	}, 10*time.Second, 5*time.Millisecond, "stream %d never opened", i)

	e.mu.Lock()
	defer e.mu.Unlock()

	return e.streams[i]
}

// loopbackHost serves engine on a real listener and opens a session over the real client.
func loopbackHost(t *testing.T, engine *loopbackEngine, source TokenSource) (*Host, *session, *recordingHandler) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	v1.RegisterOperatorServiceServer(server, engine)

	go func() { _ = server.Serve(lis) }()

	t.Cleanup(server.Stop)

	port := lis.Addr().(*net.TCPAddr).Port
	nop := newNopLogger()

	host := newTestHost(t, source, func(token string) (engineClient, error) {
		return client.New(client.WithToken(token), client.WithHostPort("127.0.0.1", port), client.WithTLSConfig(nil), client.WithLogger(nop)) //nolint:staticcheck // see the import
	})

	t.Cleanup(host.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	handler := &recordingHandler{}
	opened, err := host.Open(ctx, operator.Identity{TenantId: uuid.MustParse(engine.tenant), Name: "loopback"}, operator.OpenOpts{Handler: handler, Actions: []string{"svc:run"}})
	require.NoError(t, err)

	s := opened.(*session)
	t.Cleanup(func() { _ = s.Close(context.Background()) })

	return host, s, handler
}

// Over a real connection: the token is rotated while a session is live, the engine refuses
// the old one on the client's next Register, and the session recovers with the token the
// source now serves, restoring its action set and resuming delivery.
func TestLoopbackSessionRecoversWithRotatedToken(t *testing.T) {
	tenant := uuid.New()
	old, rotated := testJWT(t, tenant), testJWT(t, tenant)+"r"
	engine := newLoopbackEngine(tenant, old)
	source := newLockedSource(tenant, old)

	_, s, handler := loopbackHost(t, engine, source)
	s.backoff = func(ctx context.Context, _ int) error { return retry.Sleep(ctx, 10*time.Millisecond) }

	require.Equal(t, engine.first, s.Registration().WorkerId.String())
	require.Equal(t, 1, source.count())

	// the engine stops accepting the old token; the source serves the rotated one
	source.set(tenant, rotated)
	engine.setAccepted(rotated)
	engine.dropFirstStream()

	require.Eventually(t, func() bool { return engine.registrationCount() >= 2 }, 15*time.Second, 5*time.Millisecond, "no new registration")
	require.Eventually(t, func() bool { return s.Registration().WorkerId.String() == engine.next }, 15*time.Second, 5*time.Millisecond, "the session did not move to the new worker")
	assert.GreaterOrEqual(t, source.count(), 2, "the source was asked for the token again")

	run := uuid.NewString()

	engine.stream(t, 1) <- &v1.OperatorListenResponse{Message: &v1.OperatorListenResponse_Action{Action: &contracts.AssignedAction{
		ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: run, ActionId: "svc:run",
	}}}

	require.Eventually(t, func() bool { return len(handler.handled()) == 1 }, 10*time.Second, 5*time.Millisecond, "the action on the new stream never reached the handler")
	assert.NoError(t, s.Err())

	// the report goes out under the new worker, and lets Close drain without waiting
	require.NoError(t, s.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{
		TaskRunExternalId: run, EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}))
}

// Over a real connection: the client reconnects on its own after a retryable stream failure
// and registers a fresh worker; the session reports under that worker, so a report the
// operator stamped with the previous worker is accepted.
func TestLoopbackFreshReconnectReportsUnderNewWorker(t *testing.T) {
	tenant := uuid.New()
	token := testJWT(t, tenant)
	engine := newLoopbackEngine(tenant, token)

	_, s, _ := loopbackHost(t, engine, newLockedSource(tenant, token))

	previous := s.Registration().WorkerId.String()
	require.Equal(t, engine.first, previous)

	engine.dropFirstStream()

	require.Eventually(t, func() bool { return s.Registration().WorkerId.String() == engine.next }, 15*time.Second, 5*time.Millisecond, "the client did not register a fresh worker")

	err := s.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{
		WorkerId: previous, TaskRunExternalId: uuid.NewString(), EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	})
	require.NoError(t, err, "a report under the previous worker is rewritten and accepted")

	engine.mu.Lock()
	defer engine.mu.Unlock()
	require.Len(t, engine.events, 1)
	assert.Equal(t, engine.next, engine.events[0].WorkerId)
}
