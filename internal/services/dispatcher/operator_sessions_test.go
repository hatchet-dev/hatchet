package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
)

// operatorStreamHarness runs a Listen stream fake whose handler registers it with the
// dispatcher through AddOperatorStreamSession, the way the operator service does.
type operatorStreamHarness struct {
	d      *DispatcherImpl
	sender *rpcstream.Sender[v1contracts.OperatorListenResponse]
	// sent receives every message written on the stream
	sent chan *v1contracts.OperatorListenResponse
	// ready is closed once the handler has registered the session
	ready chan struct{}
	// done receives the handler's return value
	done chan error
}

// Send implements the stream: messages are queued for the test to receive.
func (h *operatorStreamHarness) Send(msg *v1contracts.OperatorListenResponse) error {
	h.sent <- msg
	return nil
}

func newOperatorStreamHarness(t *testing.T, workerId uuid.UUID) *operatorStreamHarness {
	t.Helper()

	l := zerolog.Nop()

	h := &operatorStreamHarness{
		d: &DispatcherImpl{
			workers:                             &workers{},
			l:                                   &l,
			defaultMaxWorkerLockAcquisitionTime: time.Second,
		},
		sent:  make(chan *v1contracts.OperatorListenResponse, 16),
		ready: make(chan struct{}),
		done:  make(chan error, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.sender = rpcstream.NewSender[v1contracts.OperatorListenResponse](ctx, h)

	go func() {
		defer h.sender.Close()

		session := h.d.AddOperatorStreamSession(ctx, workerId, uuid.New(), h.sender)
		defer session.Release()

		close(h.ready)

		select {
		case <-session.Fin():
			h.done <- nil
		case <-ctx.Done():
			h.done <- ctx.Err()
		}
	}()

	t.Cleanup(cancel)

	select {
	case <-h.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not register the session")
	}

	return h
}

// recv returns the next assigned action written on the stream.
func (h *operatorStreamHarness) recv(t *testing.T) *contracts.AssignedAction {
	t.Helper()

	select {
	case msg := <-h.sent:
		action := msg.GetAction()

		if action == nil {
			t.Fatalf("stream carried %T, want an action", msg.GetMessage())
		}

		return action
	case <-time.After(5 * time.Second):
		t.Fatal("no message on the stream")
		return nil
	}
}

func (h *operatorStreamHarness) worker(t *testing.T, workerId uuid.UUID) *subscribedWorker {
	t.Helper()

	ws, err := h.d.workers.Get(workerId)

	if err != nil {
		t.Fatalf("expected session to be registered: %v", err)
	}

	if len(ws) != 1 {
		t.Fatalf("expected exactly one session, got %d", len(ws))
	}

	return ws[0]
}

func TestAddOperatorStreamSessionSendsAction(t *testing.T) {
	workerId := uuid.New()
	h := newOperatorStreamHarness(t, workerId)

	action := &contracts.AssignedAction{
		TenantId:          uuid.NewString(),
		TaskRunExternalId: uuid.NewString(),
		ActionType:        contracts.ActionType_CANCEL_STEP_RUN,
	}

	if err := h.worker(t, workerId).StartBatch(context.Background(), action); err != nil {
		t.Fatalf("could not send action: %v", err)
	}

	got := h.recv(t)

	if !proto.Equal(got, action) {
		t.Fatalf("received action %v, want %v", got, action)
	}
}

// The shutdown drain signals fin and the handler must hang up; release then removes the
// session so later sends do not target a closed stream.
func TestAddOperatorStreamSessionFinAndRelease(t *testing.T) {
	workerId := uuid.New()
	h := newOperatorStreamHarness(t, workerId)

	worker := h.worker(t, workerId)

	select {
	case worker.finished <- true:
	case <-time.After(5 * time.Second):
		t.Fatal("handler is not selecting on fin")
	}

	select {
	case err := <-h.done:
		if err != nil {
			t.Fatalf("handler returned error on fin: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after fin")
	}

	if err := h.sender.Send(&v1contracts.OperatorListenResponse{}); !errors.Is(err, rpcstream.ErrClosed) {
		t.Fatalf("expected the stream to be closed once the handler returned, got %v", err)
	}

	if _, err := h.d.workers.Get(workerId); err != nil {
		t.Fatalf("release should leave the worker map entry: %v", err)
	}

	ws, _ := h.d.workers.Get(workerId)

	if len(ws) != 0 {
		t.Fatalf("release did not remove the session, %d remain", len(ws))
	}
}

// stubActionHandler records the actions an in-process session was handed.
type stubActionHandler struct {
	mu       sync.Mutex
	received []*contracts.AssignedAction
}

func (s *stubActionHandler) HandleAction(_ context.Context, action *contracts.AssignedAction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.received = append(s.received, action)

	return nil
}

// An in-process session is keyed on the session id its host chose, the same id the host records
// on the worker row as the listener fence, and delivers by calling the handler directly.
func TestAddOperatorSessionRoutesToTheHandler(t *testing.T) {
	l := zerolog.Nop()
	d := &DispatcherImpl{workers: &workers{}, l: &l}

	workerId := uuid.New()
	sessionId := uuid.New()
	handler := &stubActionHandler{}

	session := d.AddOperatorSession(workerId, sessionId, handler)

	w, err := d.workers.Get(workerId)

	if err != nil {
		t.Fatalf("expected the session to be registered: %v", err)
	}

	if len(w) != 1 {
		t.Fatalf("expected exactly one session, got %d", len(w))
	}

	action := &contracts.AssignedAction{ActionId: "svc:a"}

	if err := w[0].sendToWorker(context.Background(), action); err != nil {
		t.Fatalf("expected the action to reach the handler: %v", err)
	}

	handler.mu.Lock()
	got := len(handler.received)
	handler.mu.Unlock()

	if got != 1 {
		t.Fatalf("expected the handler to receive one action, got %d", got)
	}

	session.Release()

	// Release is idempotent: a host that releases twice must not disturb a newer session
	session.Release()

	after, err := d.workers.Get(workerId)

	if err != nil {
		t.Fatalf("unexpected error reading the worker's sessions: %v", err)
	}

	if len(after) != 0 {
		t.Fatalf("expected the session to be gone after Release, got %d", len(after))
	}
}

// A paused session returns the starts it is asked to deliver instead of sending them, the way a
// failed send does, so the dispatcher requeues them; cancels still go through, and lifting the
// pause delivers again.
func TestOperatorStreamSessionPausedReturnsStarts(t *testing.T) {
	workerId := uuid.New()
	h := newOperatorStreamHarness(t, workerId)

	start := &contracts.AssignedAction{
		TenantId:          uuid.NewString(),
		TaskRunExternalId: uuid.NewString(),
		ActionType:        contracts.ActionType_START_STEP_RUN,
	}

	cancelAction := &contracts.AssignedAction{
		TenantId:          uuid.NewString(),
		TaskRunExternalId: uuid.NewString(),
		ActionType:        contracts.ActionType_CANCEL_STEP_RUN,
	}

	worker := h.worker(t, workerId)
	worker.setPaused(true)

	err := worker.StartBatch(context.Background(), start)

	if !errors.Is(err, errWorkerPaused) {
		t.Fatalf("expected errWorkerPaused, got %v", err)
	}

	if err := worker.StartBatch(context.Background(), cancelAction); err != nil {
		t.Fatalf("could not send cancel while paused: %v", err)
	}

	got := h.recv(t)

	if !proto.Equal(got, cancelAction) {
		t.Fatalf("received %v while paused, want the cancel %v", got, cancelAction)
	}

	worker.setPaused(false)

	if err := worker.StartBatch(context.Background(), start); err != nil {
		t.Fatalf("could not send start after the pause was lifted: %v", err)
	}

	got = h.recv(t)

	if !proto.Equal(got, start) {
		t.Fatalf("received %v, want %v", got, start)
	}
}

// The handler-backed session pauses the same way: a paused session never calls its handler.
func TestOperatorHandlerSessionPausedReturnsStarts(t *testing.T) {
	l := zerolog.Nop()
	d := &DispatcherImpl{workers: &workers{}, l: &l}
	workerId := uuid.New()
	handler := &stubActionHandler{}

	session := d.AddOperatorSession(workerId, uuid.New(), handler)
	defer session.Release()

	calls := func() int {
		handler.mu.Lock()
		defer handler.mu.Unlock()

		return len(handler.received)
	}

	session.SetPaused(true)

	ws, err := d.workers.Get(workerId)

	if err != nil || len(ws) != 1 {
		t.Fatalf("expected one session, got %d (%v)", len(ws), err)
	}

	start := &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN}

	if err := ws[0].StartBatch(context.Background(), start); !errors.Is(err, errWorkerPaused) {
		t.Fatalf("expected errWorkerPaused, got %v", err)
	}

	if calls() != 0 {
		t.Fatalf("the handler was called %d times while paused", calls())
	}

	session.SetPaused(false)

	if err := ws[0].StartBatch(context.Background(), start); err != nil {
		t.Fatalf("could not deliver after the pause was lifted: %v", err)
	}

	if calls() != 1 {
		t.Fatalf("the handler was called %d times, want 1", calls())
	}
}

// The fan-out writes assigned actions on the operator stream wrapped in its message type.
func TestOperatorListenStreamWrapsActions(t *testing.T) {
	h := newOperatorStreamHarness(t, uuid.New())

	action := &contracts.AssignedAction{TaskRunExternalId: uuid.NewString(), ActionType: contracts.ActionType_START_STEP_RUN}

	if err := (operatorListenStream{sender: h.sender}).Send(action); err != nil {
		t.Fatalf("could not send: %v", err)
	}

	msg := <-h.sent

	if msg.GetAck() != nil || !proto.Equal(msg.GetAction(), action) {
		t.Fatalf("stream carried %v, want the action wrapped", msg)
	}
}
