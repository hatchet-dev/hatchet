//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

// fakeEnc "decrypts" by stripping the enc: prefix; anything else fails.
type fakeEnc struct {
	encryption.EncryptionService
}

func (fakeEnc) DecryptString(ciphertext string, dataId string) (string, error) {
	if dataId != contract.SigningSecretEncryptionDataID {
		return "", fmt.Errorf("unexpected data id %q", dataId)
	}

	if !strings.HasPrefix(ciphertext, "enc:") {
		return "", errors.New("bad ciphertext")
	}

	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

// actionDelta is one AddActions or RemoveActions call recorded by fakeSession.
type actionDelta struct {
	add    []string
	remove []string
}

// fakeSession records what the core does with a session and hands assigned actions to the
// handler the core opened it with. openDurable, when set, backs OpenDurable; otherwise the
// session reports durable delivery unsupported, as operatortest.Session does. deltas records
// every add and remove in order; flushes counts Flush calls; ops records the lifecycle calls
// (pause, close) in order.
type fakeSession struct {
	handler     operator.ActionHandler
	reg         operator.Registration
	putErr      error
	deltaErr    error
	openDurable func(taskId uuid.UUID, invocation int32) (operator.DurableChannel, error)
	puts        []*v1.CreateWorkflowVersionRequest
	deltas      []actionDelta
	events      []*contracts.StepActionEvent
	ops         []string
	flushes     int
	mu          sync.Mutex
	closed      bool
}

var _ operator.Session = (*fakeSession)(nil)

func (f *fakeSession) Registration() operator.Registration { return f.reg }

// workerId is the worker id the way events carry it.
func (f *fakeSession) workerId() string { return f.reg.WorkerId.String() }

// deliver hands an action to the core's handler the way a host does and fails the test if
// the handler refuses it.
func (f *fakeSession) deliver(t *testing.T, action *contracts.AssignedAction) {
	t.Helper()

	if err := f.handler.HandleAction(context.Background(), action); err != nil {
		t.Fatalf("handler refused the action: %v", err)
	}
}

func (f *fakeSession) PutWorkflow(_ context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.putErr != nil {
		return nil, f.putErr
	}

	f.puts = append(f.puts, wf)

	return actionsForWorkflow(wf)
}

func (f *fakeSession) AddActions(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deltaErr != nil {
		return f.deltaErr
	}

	f.deltas = append(f.deltas, actionDelta{add: append([]string{}, ids...)})

	return nil
}

func (f *fakeSession) RemoveActions(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deltaErr != nil {
		return f.deltaErr
	}

	f.deltas = append(f.deltas, actionDelta{remove: append([]string{}, ids...)})

	return nil
}

func (f *fakeSession) Flush(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.flushes++

	return nil
}

func (f *fakeSession) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, ev)

	return nil
}

func (f *fakeSession) OpenDurable(_ context.Context, taskId uuid.UUID, invocation int32) (operator.DurableChannel, error) {
	f.mu.Lock()
	open := f.openDurable
	f.mu.Unlock()

	if open == nil {
		return nil, operator.ErrNotSupported
	}

	return open(taskId, invocation)
}

func (f *fakeSession) setOpenDurable(open func(taskId uuid.UUID, invocation int32) (operator.DurableChannel, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.openDurable = open
}

func (f *fakeSession) Pause(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return operator.ErrSessionClosed
	}

	f.ops = append(f.ops, "pause")

	return nil
}

func (f *fakeSession) Close(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true
	f.ops = append(f.ops, "close")

	return nil
}

func (f *fakeSession) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closed
}

// lifecycle returns the pause and close calls in order.
func (f *fakeSession) lifecycle() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string{}, f.ops...)
}

func (f *fakeSession) eventTypes() []contracts.StepActionEventType {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]contracts.StepActionEventType, 0, len(f.events))

	for _, ev := range f.events {
		out = append(out, ev.EventType)
	}

	return out
}

func (f *fakeSession) lastEvent() *contracts.StepActionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.events) == 0 {
		return nil
	}

	return f.events[len(f.events)-1]
}

func (f *fakeSession) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.puts)
}

func (f *fakeSession) deltaCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.deltas)
}

// added and removed flatten the recorded deltas, in order.
func (f *fakeSession) added() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0)

	for _, d := range f.deltas {
		out = append(out, d.add...)
	}

	return out
}

func (f *fakeSession) removed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0)

	for _, d := range f.deltas {
		out = append(out, d.remove...)
	}

	return out
}

func (f *fakeSession) flushCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.flushes
}

type openCall struct {
	id   operator.Identity
	opts operator.OpenOpts
}

// fakeHost hands out fakeSessions and records Opens and tenant releases.
type fakeHost struct {
	openErr  error
	opens    []openCall
	sessions []*fakeSession
	released []uuid.UUID
	mu       sync.Mutex
}

var _ operator.Host = (*fakeHost)(nil)

func (f *fakeHost) Open(_ context.Context, id operator.Identity, opts operator.OpenOpts) (operator.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.opens = append(f.opens, openCall{id: id, opts: opts})

	if f.openErr != nil {
		return nil, f.openErr
	}

	s := &fakeSession{
		handler: opts.Handler,
		reg:     operator.Registration{TenantId: id.TenantId, OperatorId: uuid.New(), WorkerId: uuid.New()},
	}

	f.sessions = append(f.sessions, s)

	return s, nil
}

func (f *fakeHost) ReleaseTenant(tenantId uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.released = append(f.released, tenantId)
}

func (f *fakeHost) setOpenErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.openErr = err
}

func (f *fakeHost) openCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.opens)
}

// session returns the i-th session opened, or nil.
func (f *fakeHost) session(i int) *fakeSession {
	f.mu.Lock()
	defer f.mu.Unlock()

	if i >= len(f.sessions) {
		return nil
	}

	return f.sessions[i]
}

func (f *fakeHost) releasedTenants() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID{}, f.released...)
}

type senderCall struct {
	headers  http.Header
	method   string
	endpoint string
	body     []byte
}

type senderHandler func(ctx context.Context, call senderCall) (*safeclient.DeliveryResult, error)

// fakeSender routes by endpoint URL to a handler and records every call.
type fakeSender struct {
	handlers map[string]senderHandler
	calls    []senderCall
	mu       sync.Mutex
}

func newFakeSender() *fakeSender {
	return &fakeSender{handlers: map[string]senderHandler{}}
}

func (f *fakeSender) handle(endpoint string, h senderHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers[endpoint] = h
}

// respond installs a fixed response for an endpoint.
func (f *fakeSender) respond(endpoint string, status int, body string) {
	f.handle(endpoint, func(_ context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
		return &safeclient.DeliveryResult{StatusCode: status, BodyPrefix: []byte(body)}, nil
	})
}

func (f *fakeSender) Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error) {
	call := senderCall{method: method, endpoint: endpoint, body: body, headers: headers}

	f.mu.Lock()
	f.calls = append(f.calls, call)
	h, ok := f.handlers[endpoint]
	f.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no handler for %s", endpoint)
	}

	return h(ctx, call)
}

// DialContext dials directly: tests only reach loopback servers. InsecureDestinations lets
// the durable relay dial http test servers as ws.
func (f *fakeSender) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func (f *fakeSender) InsecureDestinations() bool {
	return true
}

func (f *fakeSender) callsTo(endpoint string) []senderCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]senderCall, 0)

	for _, c := range f.calls {
		if c.endpoint == endpoint {
			out = append(out, c)
		}
	}

	return out
}

// healthcheckBody builds a healthcheck response advertising only explicit actions.
func healthcheckBody(actions ...string) string {
	b, _ := contract.Marshal(&v1.ServerlessHealthcheckResponse{Actions: actions})
	return string(b)
}

// healthcheckWithWorkflows builds a response carrying workflows.
func healthcheckWithWorkflows(t *testing.T, workflows ...*v1.CreateWorkflowVersionRequest) string {
	t.Helper()

	b, err := contract.Marshal(&v1.ServerlessHealthcheckResponse{Workflows: workflows})

	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

type endpointSpec struct {
	tenantId uuid.UUID
	shard    int32
	name     string
	actions  []string
	enabled  bool
	healthy  pgtype.Bool
}

// newEndpointRow builds an endpoint row with a fresh id and namespace. URLs derive from the
// name so tests can install sender handlers by them.
func newEndpointRow(spec endpointSpec) *sqlcv1.V1ServerlessEndpoint {
	return &sqlcv1.V1ServerlessEndpoint{
		ID:                    uuid.New(),
		TenantID:              spec.tenantId,
		Name:                  spec.name,
		Namespace:             uuid.New(),
		Kind:                  sqlcv1.V1ServerlessEndpointKindGENERICHTTP,
		HealthcheckUrl:        "https://" + spec.name + ".example.test/health",
		TriggerUrl:            "https://" + spec.name + ".example.test/trigger",
		SigningSecretEnc:      "enc:secret-" + spec.name,
		RequestTimeoutSeconds: 5,
		PollIntervalSeconds:   3600,
		Labels:                []byte("{}"),
		Enabled:               spec.enabled,
		Shard:                 spec.shard,
		Healthy:               spec.healthy,
		RegisteredActions:     spec.actions,
	}
}

func prefixed(ns uuid.UUID, action string) string {
	return namespacePrefix(ns) + action
}

type testEnv struct {
	repo   *memrepo.Repo
	host   *fakeHost
	sender *fakeSender
	r      *runner
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	l := zerolog.Nop()

	repo := memrepo.New()
	host := &fakeHost{}
	sender := newFakeSender()

	cfg := Config{
		LinkName:               "test",
		DefaultSlots:           10,
		DurableSlots:           5,
		DrainTimeout:           time.Second,
		RoutingRefreshInterval: time.Hour,
		HealthcheckTimeout:     2 * time.Second,
		HealthcheckConcurrency: 4,
	}.withDefaults()

	r := newRunner(Deps{
		Repo:       repo,
		Host:       host,
		Encryption: fakeEnc{},
		Sender:     sender,
		Logger:     &l,
		ProcessId:  uuid.New(),
	}, cfg, newMetrics("test"))

	t.Cleanup(r.Shutdown)

	return &testEnv{repo: repo, host: host, sender: sender, r: r}
}

// addEndpoint stores the row and installs a healthcheck handler that advertises the row's
// registered actions, unprefixed, so the first poll changes nothing.
func (e *testEnv) addEndpoint(row *sqlcv1.V1ServerlessEndpoint) {
	e.repo.AddEndpoint(row)

	actions := make([]string, 0, len(row.RegisteredActions))

	for _, a := range row.RegisteredActions {
		actions = append(actions, strings.TrimPrefix(a, namespacePrefix(row.Namespace)))
	}

	e.sender.respond(row.HealthcheckUrl, http.StatusOK, healthcheckBody(actions...))
}

func (e *testEnv) unit(row *sqlcv1.V1ServerlessEndpoint) memrepo.Unit {
	return memrepo.Unit{TenantId: row.TenantID, Shard: row.Shard}
}

// poller returns the running poller for an endpoint, or nil.
func (e *testEnv) poller(row *sqlcv1.V1ServerlessEndpoint) *endpointPoller {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	ts, ok := e.r.tenants[row.TenantID]

	if !ok {
		return nil
	}

	return ts.pollers[row.ID]
}

func (e *testEnv) tenant(id uuid.UUID) *tenantState {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	return e.r.tenants[id]
}

func startAction(ns uuid.UUID, action string) *contracts.AssignedAction {
	return &contracts.AssignedAction{
		ActionType:        contracts.ActionType_START_STEP_RUN,
		TenantId:          uuid.New().String(),
		JobId:             "job",
		JobRunId:          "job-run",
		TaskId:            "task",
		TaskRunExternalId: uuid.New().String(),
		ActionId:          prefixed(ns, action),
		JobName:           prefixed(ns, "wf"),
		ActionPayload:     `{"input":1}`,
		RetryCount:        1,
	}
}
