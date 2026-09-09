//go:build !e2e && !load && !rampup && !integration

package hostinproc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/operator/hostinproc"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/operatortest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const eventually = 5 * time.Second

// testHost is a Host over in-memory doubles, with the doubles kept for assertions.
type testHost struct {
	*hostinproc.Host

	tenant     *sqlcv1.Tenant
	operators  *operatorsvctest.OperatorStore
	workers    *operatorsvctest.WorkerStore
	dispatcher *operatorsvctest.Dispatcher
	workflows  *operatorsvctest.WorkflowStore
}

type hostOpts struct {
	svcOpts   []operatorsvc.Opt
	hostOpts  []hostinproc.Opt
	workflows bool
}

func newTestHost(t *testing.T, o hostOpts) *testHost {
	t.Helper()

	l := zerolog.Nop()
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	operators := operatorsvctest.NewOperatorStore()
	workers := operatorsvctest.NewWorkerStore()
	d := operatorsvctest.NewDispatcher()

	svc, err := operatorsvc.New(
		operatorsvc.Deps{Operators: operators, Workers: workers, Dispatcher: d, DispatcherId: uuid.New()},
		append([]operatorsvc.Opt{operatorsvc.WithLogger(&l)}, o.svcOpts...)...,
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = svc.Cleanup() })

	deps := hostinproc.Deps{
		Service:    svc,
		Tenants:    operatorsvctest.NewTenantStore(tenant),
		Heartbeats: workers,
		Logger:     &l,
	}

	th := &testHost{tenant: tenant, operators: operators, workers: workers, dispatcher: d}

	if o.workflows {
		th.workflows = operatorsvctest.NewWorkflowStore()
		deps.Workflows = th.workflows
	}

	host, err := hostinproc.New(deps, append([]hostinproc.Opt{hostinproc.WithHeartbeatInterval(time.Hour)}, o.hostOpts...)...)
	require.NoError(t, err)

	t.Cleanup(host.Close)

	th.Host = host

	return th
}

// claimedRow seeds an operator row the way the REST API creates a DAG operator.
func (h *testHost) claimedRow() *sqlcv1.V1Operator {
	return h.operators.Put(&sqlcv1.V1Operator{ID: uuid.New(), TenantID: h.tenant.ID, Name: "dag", Kind: sqlcv1.V1OperatorKindDAG, Config: []byte(`{}`)})
}

type nopHandler struct{}

func (nopHandler) HandleAction(context.Context, *contracts.AssignedAction) error { return nil }

func startAction(payload string) *contracts.AssignedAction {
	return &contracts.AssignedAction{
		ActionType:        contracts.ActionType_START_STEP_RUN,
		TaskId:            "task-1",
		TaskRunExternalId: uuid.NewString(),
		ActionId:          "echo:run",
		ActionPayload:     payload,
	}
}

func TestOpenClaimedRow(t *testing.T) {
	h := newTestHost(t, hostOpts{})
	row := h.claimedRow()
	handler := nopHandler{}

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, OperatorId: &row.ID}, operator.OpenOpts{
		Handler:    handler,
		Actions:    []string{"dag:a", "dag:b"},
		SlotConfig: map[string]int32{"durable": 4},
		Labels:     map[string]interface{}{"region": "eu", "rank": 3},
	})
	require.NoError(t, err)

	reg := s.Registration()
	assert.Equal(t, h.tenant.ID, reg.TenantId)
	assert.Equal(t, row.ID, reg.OperatorId)
	assert.False(t, reg.Resumed)
	assert.Equal(t, 1, h.operators.Count(), "a claimed row is not upserted")

	op, err := h.operators.GetOperatorById(t.Context(), row.ID)
	require.NoError(t, err)
	require.NotNil(t, op.WorkerID)
	assert.Equal(t, reg.WorkerId, *op.WorkerID, "the row points at the session's worker")

	created := h.workers.Created()
	require.Len(t, created, 1)
	assert.Equal(t, "dag", created[0].Name)
	assert.Equal(t, map[string]int32{"durable": 4}, created[0].SlotConfig)
	assert.Len(t, h.workers.Labels(reg.WorkerId), 2)

	// the fence id on the worker row is the dispatcher's session key
	sessionLog := h.workers.SessionLog()
	require.Len(t, sessionLog, 1)
	assert.Equal(t, sessionLog, h.dispatcher.SessionIdLog())
	assert.True(t, h.workers.IsActive(reg.WorkerId))

	require.Len(t, h.dispatcher.Handlers(), 1)
	assert.Equal(t, handler, h.dispatcher.Handlers()[0], "the dispatcher delivers to the handler directly")

	assert.ElementsMatch(t, []string{"dag:a", "dag:b"}, h.workers.ActionSet(reg.WorkerId), "the initial actions are linked before Open returns")
	assert.Equal(t, 1, h.SessionCount())

	require.NoError(t, s.Close(t.Context()))

	assert.Equal(t, []string{"activate", "pause", "deactivate"}, h.workers.Ops(), "Close is pause then deactivate")
	assert.False(t, h.workers.IsActive(reg.WorkerId))
	assert.Equal(t, 1, h.dispatcher.ReleasedCount())
	assert.Zero(t, h.SessionCount())

	require.NoError(t, s.Close(t.Context()), "Close is idempotent")
	assert.ErrorIs(t, s.AddActions(t.Context(), []string{"x"}), operator.ErrSessionClosed)
	assert.ErrorIs(t, s.Flush(t.Context()), operator.ErrSessionClosed)
	assert.ErrorIs(t, s.Pause(t.Context()), operator.ErrSessionClosed)
}

func TestOpenByName(t *testing.T) {
	h := newTestHost(t, hostOpts{})

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "contract-op"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close(t.Context()) })

	op, err := h.operators.GetOperatorById(t.Context(), s.Registration().OperatorId)
	require.NoError(t, err)
	assert.Equal(t, "contract-op", op.Name)
	assert.Equal(t, sqlcv1.V1OperatorKindGRPC, op.Kind, "the kind defaults to GRPC")
	assert.Nil(t, op.WorkerID, "an upserted row is not pointed at its worker: the claimer never claims it")

	// the same name opens another worker of the same operator
	again, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "contract-op"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = again.Close(t.Context()) })

	assert.Equal(t, s.Registration().OperatorId, again.Registration().OperatorId)
	assert.NotEqual(t, s.Registration().WorkerId, again.Registration().WorkerId)
	assert.Equal(t, 2, h.SessionCount())
}

func TestOpenRejects(t *testing.T) {
	h := newTestHost(t, hostOpts{})

	_, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "op"}, operator.OpenOpts{})
	require.Error(t, err, "a handler is required")

	_, err = h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID}, operator.OpenOpts{Handler: nopHandler{}})
	require.Error(t, err, "an operator id or a name is required")

	_, err = h.Open(t.Context(), operator.Identity{TenantId: uuid.New(), Name: "op"}, operator.OpenOpts{Handler: nopHandler{}})
	require.Error(t, err, "an unknown tenant is refused")

	other := h.operators.Put(&sqlcv1.V1Operator{ID: uuid.New(), TenantID: uuid.New(), Name: "dag", Kind: sqlcv1.V1OperatorKindDAG})
	_, err = h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, OperatorId: &other.ID}, operator.OpenOpts{Handler: nopHandler{}})
	assert.Equal(t, codes.NotFound, status.Code(err), "another tenant's row is refused")

	assert.Zero(t, h.SessionCount())
	assert.Zero(t, h.dispatcher.SessionCount())
}

// Deltas larger than one engine delta are chunked; a delta the budget refuses fails the open and
// leaves no half-open worker behind.
func TestOpenInitialActionsChunkedAndBudgeted(t *testing.T) {
	h := newTestHost(t, hostOpts{svcOpts: []operatorsvc.Opt{operatorsvc.WithMaxActionsPerOperator(1500)}})
	row := h.claimedRow()

	actions := make([]string, 0, 1500)

	for i := 0; i < 1500; i++ {
		actions = append(actions, fmt.Sprintf("svc:action%d", i))
	}

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, OperatorId: &row.ID}, operator.OpenOpts{Handler: nopHandler{}, Actions: actions})
	require.NoError(t, err)
	assert.Len(t, h.workers.ActionSet(s.Registration().WorkerId), 1500)

	require.NoError(t, s.Close(t.Context()))

	// the operator now holds 1500 links, so the next worker has no budget left
	_, err = h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, OperatorId: &row.ID}, operator.OpenOpts{Handler: nopHandler{}, Actions: []string{"svc:one-more"}})
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(errors.Unwrap(err)))

	assert.Zero(t, h.SessionCount())
	assert.Equal(t, 2, h.dispatcher.ReleasedCount(), "the session opened for the refused worker is released")
	assert.Equal(t, []string{"activate", "pause", "deactivate", "activate", "pause", "deactivate"}, h.workers.Ops(), "the refused worker is closed like any other")
}

func TestDeltasAreSynchronous(t *testing.T) {
	h := newTestHost(t, hostOpts{})
	row := h.claimedRow()

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, OperatorId: &row.ID}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close(t.Context()) })

	workerId := s.Registration().WorkerId

	require.NoError(t, s.AddActions(t.Context(), []string{"dag:a", "dag:b"}))
	assert.ElementsMatch(t, []string{"dag:a", "dag:b"}, h.workers.ActionSet(workerId))

	require.NoError(t, s.RemoveActions(t.Context(), []string{"dag:a", "dag:missing"}))
	assert.Equal(t, []string{"dag:b"}, h.workers.ActionSet(workerId))

	require.NoError(t, s.Flush(t.Context()))

	err = s.AddActions(t.Context(), []string{"not an action id"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestHeartbeatCoversOpenAndDrainingSessions(t *testing.T) {
	h := newTestHost(t, hostOpts{hostOpts: []hostinproc.Opt{hostinproc.WithHeartbeatInterval(5 * time.Millisecond)}})

	first, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "a"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	second, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "b"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	both := []uuid.UUID{first.Registration().WorkerId, second.Registration().WorkerId}

	require.Eventually(t, func() bool {
		beats := h.workers.BulkHeartbeats()

		if len(beats) == 0 {
			return false
		}

		return assert.ObjectsAreEqual(len(both), len(beats[len(beats)-1])) && containsAll(beats[len(beats)-1], both)
	}, eventually, time.Millisecond, "one bulk heartbeat covers every open session")

	// a paused session is still heartbeated: it is draining, not gone
	require.NoError(t, first.Pause(t.Context()))
	assert.True(t, h.workers.IsPaused(first.Registration().WorkerId))

	require.NoError(t, second.Close(t.Context()))

	require.Eventually(t, func() bool {
		beats := h.workers.BulkHeartbeats()
		last := beats[len(beats)-1]

		return len(last) == 1 && last[0] == first.Registration().WorkerId
	}, eventually, time.Millisecond, "a closed session leaves the heartbeat set")

	require.NoError(t, first.Close(t.Context()))
	assert.Equal(t, []string{"activate", "activate", "pause", "pause", "deactivate", "deactivate"}, h.workers.Ops(), "a session paused before Close is not paused again")

	before := len(h.workers.BulkHeartbeats())
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, before, len(h.workers.BulkHeartbeats()), "nothing is written with no session open")
}

func containsAll(have []uuid.UUID, want []uuid.UUID) bool {
	set := make(map[uuid.UUID]struct{}, len(have))

	for _, id := range have {
		set[id] = struct{}{}
	}

	for _, id := range want {
		if _, ok := set[id]; !ok {
			return false
		}
	}

	return true
}

func TestSendStepActionEvent(t *testing.T) {
	h := newTestHost(t, hostOpts{})

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "a"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close(t.Context()) })

	require.NoError(t, s.SendStepActionEvent(t.Context(), &contracts.StepActionEvent{TaskRunExternalId: "run-1", EventType: contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED}))

	calls := h.dispatcher.StepCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, s.Registration().WorkerId.String(), calls[0].WorkerId, "an empty worker id is the session's")

	err = s.SendStepActionEvent(t.Context(), &contracts.StepActionEvent{WorkerId: uuid.NewString(), TaskRunExternalId: "run-2"})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "another worker's id is refused")
}

func TestPutWorkflow(t *testing.T) {
	h := newTestHost(t, hostOpts{workflows: true})

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "a"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close(t.Context()) })

	wf := &v1contracts.CreateWorkflowVersionRequest{
		Name:          "wf",
		Tasks:         []*v1contracts.CreateTaskOpts{{ReadableId: "a", Action: "svc:a"}, {ReadableId: "b", Action: "svc:b"}, {ReadableId: "c", Action: "svc:a"}},
		OnFailureTask: &v1contracts.CreateTaskOpts{ReadableId: "fail", Action: "svc:fail"},
	}

	actions, err := s.PutWorkflow(t.Context(), wf)
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:a", "svc:b", "svc:fail"}, actions, "the ids are the stored steps', deduplicated, including the on-failure task")
	assert.Equal(t, []*v1contracts.CreateWorkflowVersionRequest{wf}, h.workflows.Puts())
	assert.Empty(t, h.workers.ActionSet(s.Registration().WorkerId), "a put does not touch the action set")

	h.workflows.FailPut(errors.New("invalid workflow"))
	_, err = s.PutWorkflow(t.Context(), wf)
	assert.ErrorContains(t, err, "invalid workflow")

	// a host without an admin service cannot put workflows
	bare := newTestHost(t, hostOpts{})
	bs, err := bare.Open(t.Context(), operator.Identity{TenantId: bare.tenant.ID, Name: "a"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = bs.Close(t.Context()) })

	_, err = bs.PutWorkflow(t.Context(), wf)
	assert.ErrorIs(t, err, operator.ErrNotSupported)
}

// OpenDurable is the engine session's: the handshake names the session's worker and Recv only
// sees invocation traffic.
func TestOpenDurable(t *testing.T) {
	h := newTestHost(t, hostOpts{})

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "a"}, operator.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close(t.Context()) })

	workerId := s.Registration().WorkerId
	taskId := uuid.New()

	go func() {
		require.Eventually(t, func() bool { return len(h.dispatcher.Durables()) > 0 }, eventually, time.Millisecond)

		inv := h.dispatcher.Durables()[0]
		req := <-inv.Requests
		assert.Equal(t, workerId.String(), req.GetRegisterWorker().GetWorkerId())

		inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskResponseRegisterWorker{WorkerId: workerId.String()},
		}}
	}()

	ch, err := s.OpenDurable(t.Context(), taskId, 2)
	require.NoError(t, err)

	t.Cleanup(func() { _ = ch.Close() })

	inv := h.dispatcher.Durables()[0]
	assert.Equal(t, taskId, inv.ExternalId)

	require.NoError(t, ch.Send(t.Context(), &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_Memo{
		Memo: &v1contracts.DurableTaskMemoRequest{Key: []byte("k")},
	}}))

	sent := <-inv.Requests
	assert.Equal(t, taskId.String(), sent.GetMemo().GetDurableTaskExternalId())
	assert.Equal(t, int32(2), sent.GetMemo().GetInvocationCount())

	inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_MemoAck{
		MemoAck: &v1contracts.DurableTaskEventMemoAckResponse{Ref: &v1contracts.DurableEventLogEntryRef{NodeId: 1}},
	}}

	resp, err := ch.Recv(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, resp.GetMemoAck())
}

// The same handler the gRPC end-to-end suite hosts over OperatorService runs here unchanged:
// the dispatcher hands it actions directly and its reports land on the dispatcher.
func TestEchoThroughHost(t *testing.T) {
	h := newTestHost(t, hostOpts{})
	echo := &operatortest.Echo{}

	s, err := h.Open(t.Context(), operator.Identity{TenantId: h.tenant.ID, Name: "echo"}, operator.OpenOpts{
		Handler: echo,
		Actions: []string{"echo:run"},
	})
	require.NoError(t, err)
	require.NoError(t, echo.Start(t.Context(), s))

	require.Len(t, h.dispatcher.Handlers(), 1)
	handler := h.dispatcher.Handlers()[0]

	action := startAction(`{"input":{"message":"hello"}}`)
	require.NoError(t, handler.HandleAction(t.Context(), action))
	require.NoError(t, handler.HandleAction(t.Context(), &contracts.AssignedAction{ActionType: contracts.ActionType_CANCEL_STEP_RUN, TaskRunExternalId: action.TaskRunExternalId}))

	require.Eventually(t, func() bool { return len(h.dispatcher.StepCalls()) == 2 }, eventually, time.Millisecond)

	calls := h.dispatcher.StepCalls()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, calls[0].EventType)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, calls[1].EventType)
	assert.Equal(t, `{"message":"hello"}`, calls[1].EventPayload)
	assert.Equal(t, s.Registration().WorkerId.String(), calls[1].WorkerId)
	assert.Equal(t, action.TaskRunExternalId, calls[1].TaskRunExternalId)
	assert.Equal(t, 1, echo.Handled())

	// the host's teardown order: pause, the operator's drain, close
	require.NoError(t, s.Pause(t.Context()))
	echo.Drain(t.Context())
	require.NoError(t, s.Close(t.Context()))

	assert.Equal(t, []string{"activate", "pause", "deactivate"}, h.workers.Ops())
	assert.False(t, h.workers.IsActive(s.Registration().WorkerId))
}
