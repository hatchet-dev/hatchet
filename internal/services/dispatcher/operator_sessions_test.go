package dispatcher

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

const operatorStreamMethod = "/test.OperatorStream/Listen"

// operatorStreamHarness runs a bidi stream over an in-process gRPC server whose handler
// registers the stream with the dispatcher through AddOperatorStreamSession. Real gRPC is
// required because PreparedMsg.Encode reads codec info that only a live server stream carries.
type operatorStreamHarness struct {
	d      *DispatcherImpl
	stream grpc.ClientStream
	// ready is closed once the handler has registered the session
	ready chan struct{}
	// done receives the handler's return value
	done chan error
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
		ready: make(chan struct{}),
		done:  make(chan error, 1),
	}

	handler := func(srv any, stream grpc.ServerStream) error {
		session := h.d.AddOperatorStreamSession(workerId, uuid.New(), stream, nil)
		defer session.Release()

		close(h.ready)

		select {
		case <-session.Fin():
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	srv.RegisterService(&grpc.ServiceDesc{
		ServiceName: "test.OperatorStream",
		HandlerType: (*any)(nil),
		Streams: []grpc.StreamDesc{{
			StreamName: "Listen",
			Handler: func(srv any, stream grpc.ServerStream) error {
				err := handler(srv, stream)
				h.done <- err
				return err
			},
			ServerStreams: true,
			ClientStreams: true,
		}},
	}, nil)

	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)

	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    "Listen",
		ServerStreams: true,
		ClientStreams: true,
	}, operatorStreamMethod)

	if err != nil {
		t.Fatalf("could not open stream: %v", err)
	}

	h.stream = stream

	t.Cleanup(func() {
		cancel()
		conn.Close()
		srv.Stop()
		lis.Close()
	})

	select {
	case <-h.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not register the session")
	}

	return h
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

	got := &contracts.AssignedAction{}

	if err := h.stream.RecvMsg(got); err != nil {
		t.Fatalf("could not receive action: %v", err)
	}

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

	if err := h.stream.RecvMsg(&contracts.AssignedAction{}); !errors.Is(err, io.EOF) {
		t.Fatalf("expected stream to end with EOF, got %v", err)
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
