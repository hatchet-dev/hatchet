//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"encoding/json"
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
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
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

// actionDelta is one AddActions or RemoveActions call recorded by fakeRegistration.
type actionDelta struct {
	add    []string
	remove []string
}

// fakeRegistration records what the core does with a registration and lets tests push
// assigned actions. openDurable, when set, backs OpenDurable; otherwise the link reports
// durable delivery unsupported. deltas records every add and remove in order; flushes counts
// Flush calls.
type fakeRegistration struct {
	actions     chan *contracts.AssignedAction
	errs        chan error
	putErr      error
	deltaErr    error
	openDurable func(taskId string, invocation int32) (link.DurableChannel, error)
	workerId    string
	puts        []*v1.CreateWorkflowVersionRequest
	deltas      []actionDelta
	events      []*contracts.StepActionEvent
	tenantId    uuid.UUID
	shard       int
	flushes     int
	mu          sync.Mutex
	closed      bool
}

func (f *fakeRegistration) WorkerId() string { return f.workerId }

func (f *fakeRegistration) Actions(_ context.Context) (<-chan *contracts.AssignedAction, <-chan error, error) {
	return f.actions, f.errs, nil
}

func (f *fakeRegistration) PutWorkflow(_ context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.putErr != nil {
		return nil, f.putErr
	}

	f.puts = append(f.puts, wf)

	return actionsForWorkflow(wf)
}

func (f *fakeRegistration) AddActions(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deltaErr != nil {
		return f.deltaErr
	}

	f.deltas = append(f.deltas, actionDelta{add: append([]string{}, ids...)})

	return nil
}

func (f *fakeRegistration) RemoveActions(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deltaErr != nil {
		return f.deltaErr
	}

	f.deltas = append(f.deltas, actionDelta{remove: append([]string{}, ids...)})

	return nil
}

func (f *fakeRegistration) Flush(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.flushes++

	return nil
}

func (f *fakeRegistration) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, ev)

	return nil
}

func (f *fakeRegistration) OpenDurable(_ context.Context, taskId string, invocation int32) (link.DurableChannel, error) {
	f.mu.Lock()
	open := f.openDurable
	f.mu.Unlock()

	if open == nil {
		return nil, link.ErrDurableNotSupported
	}

	return open(taskId, invocation)
}

func (f *fakeRegistration) setOpenDurable(open func(taskId string, invocation int32) (link.DurableChannel, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.openDurable = open
}

func (f *fakeRegistration) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true

	return nil
}

func (f *fakeRegistration) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closed
}

func (f *fakeRegistration) eventTypes() []contracts.StepActionEventType {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]contracts.StepActionEventType, 0, len(f.events))

	for _, ev := range f.events {
		out = append(out, ev.EventType)
	}

	return out
}

func (f *fakeRegistration) lastEvent() *contracts.StepActionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.events) == 0 {
		return nil
	}

	return f.events[len(f.events)-1]
}

func (f *fakeRegistration) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.puts)
}

func (f *fakeRegistration) deltaCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.deltas)
}

// added and removed flatten the recorded deltas, in order.
func (f *fakeRegistration) added() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0)

	for _, d := range f.deltas {
		out = append(out, d.add...)
	}

	return out
}

func (f *fakeRegistration) removed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0)

	for _, d := range f.deltas {
		out = append(out, d.remove...)
	}

	return out
}

func (f *fakeRegistration) flushCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.flushes
}

type openCall struct {
	opts     link.OpenOpts
	tenantId uuid.UUID
	shard    int
}

// fakeLink hands out fakeRegistrations and records Opens and tenant releases.
type fakeLink struct {
	openErr  error
	opens    []openCall
	regs     []*fakeRegistration
	released []uuid.UUID
	mu       sync.Mutex
}

func (f *fakeLink) Open(_ context.Context, tenantId uuid.UUID, shard int, opts link.OpenOpts) (link.Registration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.opens = append(f.opens, openCall{tenantId: tenantId, shard: shard, opts: opts})

	if f.openErr != nil {
		return nil, f.openErr
	}

	reg := &fakeRegistration{
		tenantId: tenantId,
		shard:    shard,
		workerId: fmt.Sprintf("worker-%d", len(f.regs)),
		actions:  make(chan *contracts.AssignedAction, 16),
		errs:     make(chan error, 1),
	}

	f.regs = append(f.regs, reg)

	return reg, nil
}

func (f *fakeLink) ReleaseTenant(tenantId uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.released = append(f.released, tenantId)
}

func (f *fakeLink) setOpenErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.openErr = err
}

func (f *fakeLink) openCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.opens)
}

func (f *fakeLink) reg(i int) *fakeRegistration {
	f.mu.Lock()
	defer f.mu.Unlock()

	if i >= len(f.regs) {
		return nil
	}

	return f.regs[i]
}

func (f *fakeLink) releasedTenants() []uuid.UUID {
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

// healthcheckBody builds a legacy-shape healthcheck response advertising extra actions.
func healthcheckBody(actions ...string) string {
	b, _ := json.Marshal(contract.HealthcheckResponse{Actions: actions})
	return string(b)
}

// healthcheckWithWorkflows builds a response carrying protojson workflows.
func healthcheckWithWorkflows(t *testing.T, workflows ...*v1.CreateWorkflowVersionRequest) string {
	t.Helper()

	raws := make([]json.RawMessage, 0, len(workflows))

	for _, wf := range workflows {
		raw, err := protojsonMarshal(wf)

		if err != nil {
			t.Fatal(err)
		}

		raws = append(raws, raw)
	}

	b, err := json.Marshal(contract.HealthcheckResponse{Workflows: raws})

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
	link   *fakeLink
	sender *fakeSender
	r      *runner
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	l := zerolog.Nop()

	repo := memrepo.New()
	lnk := &fakeLink{}
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
		Link:       lnk,
		Encryption: fakeEnc{},
		Sender:     sender,
		Logger:     &l,
		ProcessId:  uuid.New(),
	}, cfg, newMetrics("test"))

	t.Cleanup(r.Shutdown)

	return &testEnv{repo: repo, link: lnk, sender: sender, r: r}
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
