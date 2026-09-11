//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/durable"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

// fakeDurableChannel answers every memo with a memo_ack and records what was sent.
type fakeDurableChannel struct {
	sent   []*v1.DurableTaskRequest
	recv   chan *v1.DurableTaskResponse
	closed chan struct{}
	once   sync.Once
	mu     sync.Mutex
}

func newFakeDurableChannel() *fakeDurableChannel {
	return &fakeDurableChannel{recv: make(chan *v1.DurableTaskResponse, 8), closed: make(chan struct{})}
}

func (f *fakeDurableChannel) Send(_ context.Context, req *v1.DurableTaskRequest) error {
	f.mu.Lock()
	f.sent = append(f.sent, req)
	f.mu.Unlock()

	if memo := req.GetMemo(); memo != nil {
		f.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: &v1.DurableEventLogEntryRef{
				DurableTaskExternalId: memo.DurableTaskExternalId,
				InvocationCount:       memo.InvocationCount,
				NodeId:                1,
			}},
		}}
	}

	return nil
}

func (f *fakeDurableChannel) Recv(ctx context.Context) (*v1.DurableTaskResponse, error) {
	select {
	case resp := <-f.recv:
		return resp, nil
	case <-f.closed:
		return nil, operator.ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExpectEntry is a no-op: the relay registers no entries of its own.
func (f *fakeDurableChannel) ExpectEntry(_, _ int64) error { return nil }

func (f *fakeDurableChannel) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func (f *fakeDurableChannel) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

// TestDurableDeliveryEndToEnd drives a durable action through the registration: durable
// slot, STARTED, signed upgrade verified by the endpoint, first frame, a memo round trip,
// done, COMPLETED.
func TestDurableDeliveryEndToEnd(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	row := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	secret := "secret-a"

	type seen struct {
		first *v1.ServerlessFirstFrame
		memo  map[string]json.RawMessage
	}

	got := make(chan seen, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := contract.UpgradeSigningPayload(
			r.Header.Get(contract.TimestampHeader),
			r.Header.Get(contract.NonceHeader),
			r.Header.Get(contract.TaskIdHeader),
			r.Header.Get(contract.InvocationHeader),
		)

		if r.Header.Get(contract.EndpointIdHeader) != row.ID.String() || !signature.Verify(payload, secret, r.Header.Get(contract.SignatureHeader)) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)

		if err != nil {
			return
		}

		defer conn.Close()

		var s seen

		_, data, err := conn.ReadMessage()

		if err != nil {
			return
		}

		frame, err := contract.UnmarshalFrame(data)

		if err != nil || frame.GetFirst() == nil {
			return
		}

		s.first = frame.GetFirst()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":1,"request":{"memo":{"key":"aw=="}}}`))

		_, data, err = conn.ReadMessage()

		if err != nil || json.Unmarshal(data, &s.memo) != nil {
			return
		}

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"done":{"output":"{\"memo\":\"done\"}"}}`))

		got <- s
	}))
	defer srv.Close()

	row.TriggerUrl = srv.URL + "/trigger"
	row.InlineWaitBudgetMs = 5000
	env.addEndpoint(row)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(row)})
	reg := env.host.session(0)
	require.NotNil(t, reg)

	ch := newFakeDurableChannel()
	var opened []string

	reg.setOpenDurable(func(taskId uuid.UUID, invocation int32) (operator.DurableChannel, error) {
		opened = append(opened, taskId.String())
		assert.Equal(t, int32(2), invocation)

		return ch, nil
	})

	invocation := int32(2)
	action := startAction(row.Namespace, "svc:run")
	action.DurableTaskInvocationCount = &invocation
	reg.deliver(t, action)

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2 }, eventually, 10*time.Millisecond)

	types := reg.eventTypes()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, types[0])
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, types[1])
	assert.JSONEq(t, `{"memo":"done"}`, reg.lastEvent().EventPayload)

	s := <-got
	assert.Equal(t, row.Namespace.String(), s.first.Namespace)
	assert.Equal(t, int32(2), s.first.InvocationCount)
	assert.Equal(t, int32(5000), s.first.InlineWaitBudgetMs)
	assert.Contains(t, s.memo, "response")

	assert.Equal(t, []string{action.TaskRunExternalId}, opened)
	assert.True(t, ch.isClosed(), "the relay closes the channel")

	ch.mu.Lock()
	require.Len(t, ch.sent, 1)
	assert.Equal(t, action.TaskRunExternalId, ch.sent[0].GetMemo().DurableTaskExternalId)
	ch.mu.Unlock()

	assert.Equal(t, 0, reg.inflightCount(env))
}

func (f *fakeSession) inflightCount(env *testEnv) int {
	ts := env.tenant(f.reg.TenantId)

	if ts == nil {
		return 0
	}

	reg := ts.registration()

	if reg == nil {
		return 0
	}

	return reg.inFlight()
}

// TestDurableDeliveryCloseWithoutDoneIsRetryable covers the crash rule through the core.
func TestDurableDeliveryCloseWithoutDoneIsRetryable(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	row := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)

		if err != nil {
			return
		}

		_, _, _ = conn.ReadMessage()
		_ = conn.NetConn().Close()
	}))
	defer srv.Close()

	row.TriggerUrl = srv.URL
	env.addEndpoint(row)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(row)})
	reg := env.host.session(0)
	reg.setOpenDurable(func(uuid.UUID, int32) (operator.DurableChannel, error) { return newFakeDurableChannel(), nil })

	invocation := int32(0)
	action := startAction(row.Namespace, "svc:run")
	action.DurableTaskInvocationCount = &invocation
	reg.deliver(t, action)

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2 }, eventually, 10*time.Millisecond)

	ev := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev.EventType)
	assert.Contains(t, ev.EventPayload, "without a done frame")
	assert.False(t, *ev.ShouldNotRetry)
}

// TestDurableDeliveryCancelSendsNoSecondEvent: an engine cancel closes the socket with 4003
// and the CANCELLED event sent by the cancel path is the only terminal event.
func TestDurableDeliveryCancelSendsNoSecondEvent(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	row := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})

	closeCode := make(chan int, 1)
	upgraded := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)

		if err != nil {
			return
		}

		defer conn.Close()

		upgraded <- struct{}{}

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				if ce, ok := err.(*websocket.CloseError); ok {
					closeCode <- ce.Code
				}

				return
			}
		}
	}))
	defer srv.Close()

	row.TriggerUrl = srv.URL
	env.addEndpoint(row)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(row)})
	reg := env.host.session(0)
	reg.setOpenDurable(func(uuid.UUID, int32) (operator.DurableChannel, error) { return newFakeDurableChannel(), nil })

	invocation := int32(1)
	action := startAction(row.Namespace, "svc:run")
	action.DurableTaskInvocationCount = &invocation
	reg.deliver(t, action)

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 1 }, eventually, 10*time.Millisecond)

	// STARTED is reported before the socket is dialed: a cancel that lands before the
	// upgrade aborts the dial and no socket is ever closed, which is not what this test is
	// about. Wait for the endpoint to hold the socket.
	select {
	case <-upgraded:
	case <-time.After(eventually):
		t.Fatal("endpoint never saw the upgrade")
	}

	cancel := &contracts.AssignedAction{ActionType: contracts.ActionType_CANCEL_STEP_RUN, TaskRunExternalId: action.TaskRunExternalId}
	reg.deliver(t, cancel)

	select {
	case code := <-closeCode:
		assert.Equal(t, durable.CloseCancelled, code)
	case <-time.After(eventually):
		t.Fatal("endpoint never saw the close")
	}

	require.Eventually(t, func() bool { return reg.inflightCount(env) == 0 }, eventually, 10*time.Millisecond)

	types := reg.eventTypes()
	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED,
	}, types)
}
