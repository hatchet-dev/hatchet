package enginelink

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// fakeDispatcher records the seam calls and, for durable sessions, plays the engine: it acks
// the register worker request and answers memos with memo acks until the session context ends.
type fakeDispatcher struct {
	mu sync.Mutex

	sessions map[uuid.UUID]operator.Operator
	released []uuid.UUID
	notified []uuid.UUID
	events   []*contracts.StepActionEvent
	eventCtx []context.Context

	// durableFirstResponse replaces the register worker ack when set.
	durableFirstResponse *v1.DurableTaskResponse
	durableRegistered    []uuid.UUID
	durableRequests      []*v1.DurableTaskRequest
	durableCtx           []context.Context
}

func newFakeDispatcher() *fakeDispatcher {
	return &fakeDispatcher{sessions: map[uuid.UUID]operator.Operator{}}
}

func (f *fakeDispatcher) AddOperatorSession(workerId uuid.UUID, op operator.Operator) func() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessions[workerId] = op

	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		delete(f.sessions, workerId)
		f.released = append(f.released, workerId)
	}
}

func (f *fakeDispatcher) NotifyNewWorker(_ context.Context, _ *sqlcv1.Tenant, workerId uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.notified = append(f.notified, workerId)
}

func (f *fakeDispatcher) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, ev)
	f.eventCtx = append(f.eventCtx, ctx)

	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeDispatcher) RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1.DurableTaskRequest, <-chan *v1.DurableTaskResponse, error) {
	f.mu.Lock()
	f.durableRegistered = append(f.durableRegistered, externalId)
	f.durableCtx = append(f.durableCtx, ctx)
	first := f.durableFirstResponse
	f.mu.Unlock()

	reqCh := make(chan *v1.DurableTaskRequest)
	respCh := make(chan *v1.DurableTaskResponse)

	send := func(resp *v1.DurableTaskResponse) bool {
		select {
		case respCh <- resp:
			return true
		case <-ctx.Done():
			return false
		}
	}

	go func() {
		defer close(respCh)

		for {
			select {
			case <-ctx.Done():
				return
			case req := <-reqCh:
				f.mu.Lock()
				f.durableRequests = append(f.durableRequests, req)
				f.mu.Unlock()

				var resp *v1.DurableTaskResponse

				switch m := req.GetMessage().(type) {
				case *v1.DurableTaskRequest_RegisterWorker:
					resp = first

					if resp == nil {
						resp = &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_RegisterWorker{
							RegisterWorker: &v1.DurableTaskResponseRegisterWorker{},
						}}
					}
				case *v1.DurableTaskRequest_Memo:
					resp = &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
						MemoAck: &v1.DurableTaskEventMemoAckResponse{
							Ref: &v1.DurableEventLogEntryRef{
								DurableTaskExternalId: m.Memo.DurableTaskExternalId,
								InvocationCount:       m.Memo.InvocationCount,
							},
						},
					}}
				default:
					continue
				}

				if !send(resp) {
					return
				}
			}
		}
	}()

	return reqCh, respCh, nil
}

func (f *fakeDispatcher) session(workerId uuid.UUID) operator.Operator {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.sessions[workerId]
}

func (f *fakeDispatcher) notifyCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.notified)
}

func (f *fakeDispatcher) requests() []*v1.DurableTaskRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*v1.DurableTaskRequest(nil), f.durableRequests...)
}

type putCall struct {
	ctx context.Context
	wf  *v1.CreateWorkflowVersionRequest
}

type fakeAdmin struct {
	v1.UnimplementedAdminServiceServer

	mu   sync.Mutex
	puts []putCall
	err  error
}

func (f *fakeAdmin) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	f.puts = append(f.puts, putCall{ctx: ctx, wf: wf})

	return &v1.CreateWorkflowVersionResponse{Id: uuid.NewString()}, nil
}

func (f *fakeAdmin) Cleanup() error { return nil }

type fakeTenants struct {
	tenant *sqlcv1.Tenant
}

func (f *fakeTenants) GetTenantByID(_ context.Context, tenantId uuid.UUID) (*sqlcv1.Tenant, error) {
	if f.tenant == nil || f.tenant.ID != tenantId {
		return nil, pgx.ErrNoRows
	}

	return f.tenant, nil
}

type createCall struct {
	dispatcherId uuid.UUID
	op           *sqlcv1.V1Operator
	opts         repository.CreateOperatorConnectionWorkerOpts
}

type fakeOperators struct {
	mu       sync.Mutex
	upserts  []string
	creates  []createCall
	actions  [][]string
	operator *sqlcv1.V1Operator
}

func (f *fakeOperators) UpsertServerlessOperator(_ context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.upserts = append(f.upserts, name)

	if f.operator == nil {
		f.operator = &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenantId, Name: name, Kind: sqlcv1.V1OperatorKindSERVERLESS}
	}

	return f.operator, nil
}

func (f *fakeOperators) CreateOperatorConnectionWorker(_ context.Context, dispatcherId uuid.UUID, op *sqlcv1.V1Operator, opts repository.CreateOperatorConnectionWorkerOpts) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.creates = append(f.creates, createCall{dispatcherId: dispatcherId, op: op, opts: opts})

	return &sqlcv1.Worker{ID: uuid.New(), TenantId: op.TenantID, Name: opts.Name}, nil
}

func (f *fakeOperators) UpdateOperatorWorkerActions(_ context.Context, _ uuid.UUID, _ uuid.UUID, actions []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.actions = append(f.actions, actions)

	return nil
}

type activeWrite struct {
	workerId uuid.UUID
	ts       time.Time
	active   bool
}

type fakeWorkers struct {
	mu         sync.Mutex
	labels     map[uuid.UUID][]repository.UpsertWorkerLabelOpts
	active     []activeWrite
	heartbeats int
	deactivate error
}

func newFakeWorkers() *fakeWorkers {
	return &fakeWorkers{labels: map[uuid.UUID][]repository.UpsertWorkerLabelOpts{}}
}

func (f *fakeWorkers) UpsertWorkerLabels(_ context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.labels[workerId] = opts

	return nil, nil
}

func (f *fakeWorkers) UpdateWorkerActiveStatus(_ context.Context, _ uuid.UUID, workerId uuid.UUID, isActive bool, ts time.Time) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !isActive && f.deactivate != nil {
		return nil, f.deactivate
	}

	f.active = append(f.active, activeWrite{workerId: workerId, ts: ts, active: isActive})

	return &sqlcv1.Worker{ID: workerId}, nil
}

func (f *fakeWorkers) UpdateWorkerHeartbeat(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.heartbeats++

	return nil
}

func (f *fakeWorkers) heartbeatCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.heartbeats
}

func (f *fakeWorkers) activeWrites() []activeWrite {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]activeWrite(nil), f.active...)
}

type harness struct {
	link       *Link
	dispatcher *fakeDispatcher
	admin      *fakeAdmin
	operators  *fakeOperators
	workers    *fakeWorkers
	tenant     *sqlcv1.Tenant
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	nop := zerolog.Nop()

	h := &harness{
		dispatcher: newFakeDispatcher(),
		admin:      &fakeAdmin{},
		operators:  &fakeOperators{},
		workers:    newFakeWorkers(),
		tenant:     &sqlcv1.Tenant{ID: uuid.New()},
	}

	h.link = &Link{
		dispatcher:        h.dispatcher,
		admin:             h.admin,
		tenants:           &fakeTenants{tenant: h.tenant},
		operators:         h.operators,
		workers:           h.workers,
		v:                 validator.NewDefaultValidator(),
		l:                 &nop,
		dispatcherId:      uuid.New(),
		name:              DefaultOperatorName,
		heartbeatInterval: 5 * time.Millisecond,
	}

	return h
}

func workflow(name string, actions ...string) *v1.CreateWorkflowVersionRequest {
	wf := &v1.CreateWorkflowVersionRequest{Name: name}

	for i, action := range actions {
		wf.Tasks = append(wf.Tasks, &v1.CreateTaskOpts{ReadableId: fmt.Sprintf("t%d", i), Action: action})
	}

	return wf
}

func (h *harness) open(t *testing.T, opts link.OpenOpts) (link.Registration, *registration) {
	t.Helper()

	reg, err := h.link.Open(context.Background(), h.tenant.ID, 2, opts)
	require.NoError(t, err)

	t.Cleanup(func() { _ = reg.Close() })

	return reg, reg.(*registration)
}

func tenantOf(t *testing.T, ctx context.Context) *sqlcv1.Tenant {
	t.Helper()

	tenant, ok := ctx.Value(tenantContextKey).(*sqlcv1.Tenant)
	require.True(t, ok, "tenant must be on the context")

	return tenant
}

func TestOpenRegistersWorkerAndSession(t *testing.T) {
	h := newHarness(t)

	wf1 := workflow("ns_a", "ns_svc:Run", "ns_svc:Finish")
	wf1.OnFailureTask = &v1.CreateTaskOpts{ReadableId: "fail", Action: "ns_svc:OnFailure"}
	wf2 := workflow("ns_b", "ns_svc:Run")

	opts := link.OpenOpts{
		Workflows:  []*v1.CreateWorkflowVersionRequest{wf1, wf2},
		Actions:    []string{"ns_other:Extra", "ns_svc:run"},
		SlotConfig: map[string]int32{repository.SlotTypeDefault: 3, repository.SlotTypeDurable: 2},
		Labels:     map[string]interface{}{"hatchet-serverless-process": "proc-1", "weight": 7},
	}

	reg, r := h.open(t, opts)

	// workflows are put with the tenant on the context
	require.Len(t, h.admin.puts, 2)
	assert.Same(t, wf1, h.admin.puts[0].wf)
	assert.Equal(t, h.tenant.ID, tenantOf(t, h.admin.puts[0].ctx).ID)

	// one SERVERLESS operator row by name
	assert.Equal(t, []string{DefaultOperatorName}, h.operators.upserts)

	// the worker is pinned to the dispatcher, named per unit, and carries the derived union
	require.Len(t, h.operators.creates, 1)
	create := h.operators.creates[0]
	assert.Equal(t, h.link.dispatcherId, create.dispatcherId)
	assert.Equal(t, sqlcv1.V1OperatorKindSERVERLESS, create.op.Kind)
	assert.Equal(t, fmt.Sprintf("serverless-%s-2", h.link.dispatcherId), create.opts.Name)
	assert.Equal(t, []string{"ns_svc:run", "ns_svc:finish", "ns_svc:onfailure", "ns_other:Extra"}, create.opts.Actions)
	assert.Equal(t, opts.SlotConfig, create.opts.SlotConfig)

	workerId := r.workerId
	assert.Equal(t, workerId.String(), reg.WorkerId())

	// labels: the core's plus the shard
	labels := h.workers.labels[workerId]
	byKey := map[string]repository.UpsertWorkerLabelOpts{}

	for _, l := range labels {
		byKey[l.Key] = l
	}

	require.Len(t, byKey, 3)
	assert.Equal(t, "proc-1", *byKey["hatchet-serverless-process"].StrValue)
	assert.Equal(t, int32(7), *byKey["weight"].IntValue)
	assert.Equal(t, int32(2), *byKey[shardLabel].IntValue)

	// activated with a millisecond-truncated timestamp
	writes := h.workers.activeWrites()
	require.Len(t, writes, 1)
	assert.True(t, writes[0].active)
	assert.Equal(t, workerId, writes[0].workerId)
	assert.Equal(t, writes[0].ts, writes[0].ts.Truncate(time.Millisecond))

	// session registered and scheduler notified
	session := h.dispatcher.session(workerId)
	require.NotNil(t, session)
	assert.Equal(t, workerId, session.WorkerId())
	assert.Equal(t, 1, h.dispatcher.notifyCount())

	// heartbeats run until Close
	assert.Eventually(t, func() bool { return h.workers.heartbeatCount() >= 2 }, time.Second, time.Millisecond)

	require.NoError(t, reg.Close())

	assert.Equal(t, []uuid.UUID{workerId}, h.dispatcher.released)
	assert.Nil(t, h.dispatcher.session(workerId))

	writes = h.workers.activeWrites()
	require.Len(t, writes, 2)
	assert.False(t, writes[1].active)
	assert.Equal(t, writes[0].ts, writes[1].ts, "deactivation is fenced on the same session timestamp")

	count := h.workers.heartbeatCount()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, count, h.workers.heartbeatCount(), "heartbeats stop on Close")

	// Close is idempotent
	require.NoError(t, reg.Close())
	require.Len(t, h.dispatcher.released, 1)
}

func TestOpenDefaultsAndFailures(t *testing.T) {
	t.Run("default slot config", func(t *testing.T) {
		h := newHarness(t)

		h.open(t, link.OpenOpts{})

		require.Len(t, h.operators.creates, 1)
		assert.Equal(t, map[string]int32{repository.SlotTypeDefault: defaultSlotCount}, h.operators.creates[0].opts.SlotConfig)
	})

	t.Run("unknown tenant", func(t *testing.T) {
		h := newHarness(t)

		_, err := h.link.Open(context.Background(), uuid.New(), 0, link.OpenOpts{})

		require.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Empty(t, h.operators.creates)
	})

	t.Run("rejected workflow", func(t *testing.T) {
		h := newHarness(t)
		h.admin.err = errors.New("bad workflow")

		_, err := h.link.Open(context.Background(), h.tenant.ID, 0, link.OpenOpts{
			Workflows: []*v1.CreateWorkflowVersionRequest{workflow("ns_a", "svc:run")},
		})

		require.ErrorContains(t, err, "bad workflow")
		assert.Empty(t, h.operators.upserts, "nothing is registered when a workflow is rejected")
	})

	t.Run("invalid action", func(t *testing.T) {
		h := newHarness(t)

		_, err := h.link.Open(context.Background(), h.tenant.ID, 0, link.OpenOpts{Actions: []string{"noverb"}})

		require.ErrorContains(t, err, "invalid registration")
		assert.Empty(t, h.admin.puts)
	})

	t.Run("deactivation error surfaces on Close", func(t *testing.T) {
		h := newHarness(t)
		h.workers.deactivate = errors.New("db down")

		reg, _ := h.open(t, link.OpenOpts{})

		require.ErrorContains(t, reg.Close(), "db down")
		require.Len(t, h.dispatcher.released, 1, "the session is released even when deactivation fails")
	})

	t.Run("missing worker row on Close is not an error", func(t *testing.T) {
		h := newHarness(t)
		h.workers.deactivate = pgx.ErrNoRows

		reg, _ := h.open(t, link.OpenOpts{})

		require.NoError(t, reg.Close())
	})
}

func TestHandleActionForwardsAndReportsFlowControl(t *testing.T) {
	h := newHarness(t)

	reg, r := h.open(t, link.OpenOpts{SlotConfig: map[string]int32{repository.SlotTypeDefault: 1}})

	session := h.dispatcher.session(r.workerId)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := reg.Actions(ctx)
	require.NoError(t, err)

	_, _, err = reg.Actions(ctx)
	require.Error(t, err, "Actions may only be called once")

	start := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN, TaskRunExternalId: uuid.NewString()}
	cancelAction := &contracts.AssignedAction{ActionType: contracts.ActionType_CANCEL_STEP_RUN, TaskRunExternalId: start.TaskRunExternalId}

	require.NoError(t, session.HandleAction(context.Background(), start))

	// the buffer holds one action (slot config), so the next is refused until the core reads
	require.ErrorIs(t, session.HandleAction(context.Background(), cancelAction), ErrFlowControl)

	assert.Same(t, start, <-actions)

	require.NoError(t, session.HandleAction(context.Background(), cancelAction))
	assert.Same(t, cancelAction, <-actions)

	// unsupported action types are dropped without error and without occupying the buffer
	require.NoError(t, session.HandleAction(context.Background(), &contracts.AssignedAction{ActionType: contracts.ActionType_START_GET_GROUP_KEY}))
	require.NoError(t, session.HandleAction(context.Background(), start))
	assert.Same(t, start, <-actions)

	// no errors are ever reported by the in-process link
	select {
	case err := <-errCh:
		t.Fatalf("unexpected error on the error channel: %v", err)
	default:
	}

	require.NoError(t, reg.Close())

	require.ErrorIs(t, session.HandleAction(context.Background(), start), ErrClosed)

	_, ok := <-actions
	assert.False(t, ok, "the action channel closes with the registration")

	_, ok = <-errCh
	assert.False(t, ok, "the error channel closes with the registration")
}

func TestActionsEndWithContext(t *testing.T) {
	h := newHarness(t)

	reg, r := h.open(t, link.OpenOpts{})

	ctx, cancel := context.WithCancel(context.Background())

	actions, _, err := reg.Actions(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-actions:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("the action channel did not close when ctx was cancelled")
	}

	require.ErrorIs(t, h.dispatcher.session(r.workerId).HandleAction(context.Background(), &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN}), ErrClosed)

	// Close still deactivates and releases after the stream ended on its own
	require.NoError(t, reg.Close())
	require.Len(t, h.dispatcher.released, 1)
	require.Len(t, h.workers.activeWrites(), 2)
}

func TestPutWorkflowAndUpdateActionsCallThrough(t *testing.T) {
	h := newHarness(t)

	reg, r := h.open(t, link.OpenOpts{Actions: []string{"ns_svc:run"}})

	require.Equal(t, 1, h.dispatcher.notifyCount())

	wf := workflow("ns_c", "ns_svc:Other")

	require.NoError(t, reg.PutWorkflow(context.Background(), wf, []string{"ns_svc:run", "ns_svc:other"}))

	require.Len(t, h.admin.puts, 1)
	assert.Same(t, wf, h.admin.puts[0].wf)
	assert.Equal(t, h.tenant.ID, tenantOf(t, h.admin.puts[0].ctx).ID)

	require.Len(t, h.operators.actions, 1)
	assert.Equal(t, []string{"ns_svc:other", "ns_svc:run"}, h.operators.actions[0], "the workflow's actions lead, then the full set without duplicates")
	assert.Equal(t, 2, h.dispatcher.notifyCount())

	require.NoError(t, reg.UpdateActions(context.Background(), []string{"ns_svc:run", "ns_svc:run"}))

	require.Len(t, h.operators.actions, 2)
	assert.Equal(t, []string{"ns_svc:run"}, h.operators.actions[1])
	assert.Equal(t, 3, h.dispatcher.notifyCount())

	require.Error(t, reg.UpdateActions(context.Background(), []string{"noverb"}))
	require.Error(t, reg.UpdateActions(context.Background(), []string{"ns_svc:run", ""}), "empty entries fail validation like they do on the gRPC path")
	require.Len(t, h.operators.actions, 2, "invalid actions are rejected before any write")

	require.Error(t, reg.PutWorkflow(context.Background(), workflow("ns_d", ""), nil), "a task without an action is rejected")
	require.Len(t, h.admin.puts, 1)

	_ = r
}

func TestSendStepActionEventFillsWorkerAndTenant(t *testing.T) {
	h := newHarness(t)

	reg, r := h.open(t, link.OpenOpts{})

	ev := &contracts.StepActionEvent{TaskRunExternalId: uuid.NewString(), EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED}

	require.NoError(t, reg.SendStepActionEvent(context.Background(), ev))

	require.Len(t, h.dispatcher.events, 1)
	assert.Equal(t, r.workerId.String(), h.dispatcher.events[0].WorkerId)
	assert.Equal(t, h.tenant.ID, tenantOf(t, h.dispatcher.eventCtx[0]).ID)

	other := uuid.NewString()
	require.NoError(t, reg.SendStepActionEvent(context.Background(), &contracts.StepActionEvent{WorkerId: other}))
	assert.Equal(t, other, h.dispatcher.events[1].WorkerId, "a worker id set by the caller is kept")
}

func TestOpenDurableHandshakeAndRelay(t *testing.T) {
	h := newHarness(t)

	reg, r := h.open(t, link.OpenOpts{})

	taskId := uuid.New()

	ch, err := reg.OpenDurable(context.Background(), taskId.String(), 3)
	require.NoError(t, err)

	require.Equal(t, []uuid.UUID{taskId}, h.dispatcher.durableRegistered)
	assert.Equal(t, h.tenant.ID, tenantOf(t, h.dispatcher.durableCtx[0]).ID, "the session context carries the tenant")

	// the register worker request went first and its ack was consumed by the handshake
	reqs := h.dispatcher.requests()
	require.Len(t, reqs, 1)
	assert.Equal(t, r.workerId.String(), reqs[0].GetRegisterWorker().GetWorkerId())

	// requests are stamped with the invocation's task id and count before reaching the engine.
	// Send stamps in place, so each send gets its own message as a decoded frame would.
	memo := func() *v1.DurableTaskRequest {
		return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{
			DurableTaskExternalId: "wrong",
			InvocationCount:       99,
			Key:                   []byte("k"),
		}}}
	}

	require.NoError(t, ch.Send(memo()))

	resp, err := ch.Recv()
	require.NoError(t, err)
	require.NotNil(t, resp.GetMemoAck())
	assert.Equal(t, taskId.String(), resp.GetMemoAck().GetRef().GetDurableTaskExternalId())
	assert.Equal(t, int32(3), resp.GetMemoAck().GetRef().GetInvocationCount())

	// one ack-bearing request at a time: a second one is refused until the ack is read
	require.NoError(t, ch.Send(memo()))
	require.ErrorIs(t, ch.Send(memo()), link.ErrRequestInFlight)

	_, err = ch.Recv()
	require.NoError(t, err)
	require.NoError(t, ch.Send(memo()), "the slot is released once the ack is read")

	_, err = ch.Recv()
	require.NoError(t, err)

	// complete_memo is stamped through its ref and is not ack-bearing
	complete := func() *v1.DurableTaskRequest {
		return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{CompleteMemo: &v1.DurableTaskCompleteMemoRequest{
			Ref: &v1.DurableEventLogEntryRef{DurableTaskExternalId: "wrong", InvocationCount: 99, NodeId: 5},
		}}}
	}

	require.NoError(t, ch.Send(complete()))
	require.NoError(t, ch.Send(complete()))

	reqs = h.dispatcher.requests()
	last := reqs[len(reqs)-1].GetCompleteMemo().GetRef()
	assert.Equal(t, taskId.String(), last.GetDurableTaskExternalId())
	assert.Equal(t, int32(3), last.GetInvocationCount())
	assert.Equal(t, int64(5), last.GetNodeId())

	require.Error(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{CompleteMemo: &v1.DurableTaskCompleteMemoRequest{}}}), "complete_memo needs a ref")
	require.Error(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_RegisterWorker{RegisterWorker: &v1.DurableTaskRequestRegisterWorker{}}}), "link-internal kinds are rejected")
	require.Error(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WorkerStatus{WorkerStatus: &v1.DurableTaskWorkerStatusRequest{}}}), "link-internal kinds are rejected")

	require.NoError(t, ch.Close())

	_, err = ch.Recv()
	require.ErrorIs(t, err, link.ErrChannelClosed)
	require.ErrorIs(t, ch.Send(memo()), link.ErrChannelClosed)
	require.NoError(t, ch.Close(), "Close is idempotent")

	// the session context ended with Close
	require.Error(t, h.dispatcher.durableCtx[0].Err())

	_, err = reg.OpenDurable(context.Background(), "not-a-uuid", 1)
	require.Error(t, err)
}

func TestOpenDurableRejectsBadAck(t *testing.T) {
	t.Run("engine error", func(t *testing.T) {
		h := newHarness(t)
		h.dispatcher.durableFirstResponse = &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
			Error: &v1.DurableTaskErrorResponse{ErrorMessage: "no such worker"},
		}}

		reg, _ := h.open(t, link.OpenOpts{})

		_, err := reg.OpenDurable(context.Background(), uuid.NewString(), 1)

		require.ErrorContains(t, err, "no such worker")
		require.Error(t, h.dispatcher.durableCtx[0].Err(), "the session is torn down after a failed handshake")
	})

	t.Run("unexpected message", func(t *testing.T) {
		h := newHarness(t)
		h.dispatcher.durableFirstResponse = &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{},
		}}

		reg, _ := h.open(t, link.OpenOpts{})

		_, err := reg.OpenDurable(context.Background(), uuid.NewString(), 1)

		require.ErrorContains(t, err, "register worker ack")
	})

	t.Run("handshake context cancelled", func(t *testing.T) {
		h := newHarness(t)

		reg, _ := h.open(t, link.OpenOpts{})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := reg.OpenDurable(ctx, uuid.NewString(), 1)

		require.ErrorIs(t, err, context.Canceled)
	})
}

// Every ack-bearing request kind is stamped with the invocation's ids before it reaches the
// engine, and a session the engine ends on its own is reported as a failure, not a close.
func TestDurableSendStampsEveryKind(t *testing.T) {
	h := newHarness(t)

	reg, _ := h.open(t, link.OpenOpts{})

	taskId := uuid.New()

	ch, err := reg.OpenDurable(context.Background(), taskId.String(), 7)
	require.NoError(t, err)

	cases := map[string]*v1.DurableTaskRequest{
		"memo":         {Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{}}},
		"trigger_runs": {Message: &v1.DurableTaskRequest_TriggerRuns{TriggerRuns: &v1.DurableTaskTriggerRunsRequest{}}},
		"wait_for":     {Message: &v1.DurableTaskRequest_WaitFor{WaitFor: &v1.DurableTaskWaitForRequest{}}},
		"evict":        {Message: &v1.DurableTaskRequest_EvictInvocation{EvictInvocation: &v1.DurableTaskEvictInvocationRequest{}}},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			fresh, err := reg.OpenDurable(context.Background(), taskId.String(), 7)
			require.NoError(t, err)

			defer func() { _ = fresh.Close() }()

			require.NoError(t, fresh.Send(req))

			var gotId string
			var gotCount int32

			switch m := req.GetMessage().(type) {
			case *v1.DurableTaskRequest_Memo:
				gotId, gotCount = m.Memo.DurableTaskExternalId, m.Memo.InvocationCount
			case *v1.DurableTaskRequest_TriggerRuns:
				gotId, gotCount = m.TriggerRuns.DurableTaskExternalId, m.TriggerRuns.InvocationCount
			case *v1.DurableTaskRequest_WaitFor:
				gotId, gotCount = m.WaitFor.DurableTaskExternalId, m.WaitFor.InvocationCount
			case *v1.DurableTaskRequest_EvictInvocation:
				gotId, gotCount = m.EvictInvocation.DurableTaskExternalId, m.EvictInvocation.InvocationCount
			}

			assert.Equal(t, taskId.String(), gotId)
			assert.Equal(t, int32(7), gotCount)
		})
	}

	// the engine ending the session (here: the fake's context, cancelled through a second
	// channel's Close is not shared, so cancel the session directly) surfaces as a failure
	ch.(*durableChannel).cancel()

	_, err = ch.Recv()
	require.ErrorIs(t, err, errSessionEnded)
	require.NoError(t, ch.Close())
}

func TestSlotBuffer(t *testing.T) {
	assert.Equal(t, 1, slotBuffer(nil))
	assert.Equal(t, 1, slotBuffer(map[string]int32{"a": -1}))
	assert.Equal(t, 30, slotBuffer(map[string]int32{"a": 10, "b": 20}))
}
