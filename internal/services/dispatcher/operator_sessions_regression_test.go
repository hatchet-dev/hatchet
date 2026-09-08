package dispatcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/syncx"
)

// The shutdown drain ranges the session map and signals fin on every stream-backed session it
// finds. A Listen handler that exits concurrently releases its session and stops selecting on
// fin, so the drain must not block on a pointer it captured before the release.
func TestOperatorStreamSessionDrainDoesNotBlockAfterRelease(t *testing.T) {
	d := &DispatcherImpl{workers: &workers{}}

	session := d.AddOperatorStreamSession(uuid.New(), uuid.New(), nil, nil)

	var captured *subscribedWorker

	d.workers.Range(func(_ uuid.UUID, sessions *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		sessions.Range(func(_ uuid.UUID, w *subscribedWorker) bool {
			captured = w
			return false
		})

		return false
	})

	session.Release()

	done := make(chan struct{})

	go func() {
		captured.requestFin()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("drain blocked on a released session")
	}
}

// A drain that reaches a live session still hangs it up: the handler observes fin.
func TestOperatorStreamSessionDrainSignalsLiveSession(t *testing.T) {
	d := &DispatcherImpl{workers: &workers{}}

	session := d.AddOperatorStreamSession(uuid.New(), uuid.New(), nil, nil)
	defer session.Release()

	var captured *subscribedWorker

	d.workers.Range(func(_ uuid.UUID, sessions *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		sessions.Range(func(_ uuid.UUID, w *subscribedWorker) bool {
			captured = w
			return false
		})

		return false
	})

	go captured.requestFin()

	select {
	case <-session.Fin():
	case <-time.After(time.Second):
		t.Fatal("live session did not observe fin")
	}
}

// blockedSendStream parks every SendMsg on gate and records the order calls entered.
type blockedSendStream struct {
	grpc.ServerStream

	entered chan int32
	gate    chan struct{}
	calls   atomic.Int32
}

func (s *blockedSendStream) SendMsg(any) error {
	s.entered <- s.calls.Add(1)
	<-s.gate

	return nil
}

// gRPC forbids concurrent SendMsg calls on one stream. When a dispatch operation is cancelled
// while its SendMsg is blocked by flow control, the caller returns but the send serialisation
// must stay held until that SendMsg exits: the next caller fails fast with errFlowControlActive
// instead of starting an overlapping SendMsg.
func TestOperatorStreamSessionCancelledSendDoesNotOverlap(t *testing.T) {
	workerId := uuid.New()
	h := newOperatorStreamHarness(t, workerId)
	w := h.worker(t, workerId)

	blocked := &blockedSendStream{ServerStream: w.stream, entered: make(chan int32, 2), gate: make(chan struct{})}
	w.stream = blocked
	w.sendLock.Timeout = 50 * time.Millisecond

	defer close(blocked.gate)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)

	go func() {
		first <- w.StartBatch(ctx, &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN})
	}()

	<-blocked.entered
	cancel()

	select {
	case err := <-first:
		if err == nil {
			t.Fatal("cancelled send returned nil")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled caller did not return")
	}

	second := make(chan error, 1)

	go func() {
		second <- w.StartBatch(context.Background(), &contracts.AssignedAction{ActionType: contracts.ActionType_START_STEP_RUN})
	}()

	select {
	case n := <-blocked.entered:
		t.Fatalf("SendMsg call %d entered while the first SendMsg was still blocked", n)
	case err := <-second:
		if !errors.Is(err, errFlowControlActive) {
			t.Fatalf("second caller returned %v, want errFlowControlActive", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second caller neither failed fast nor overlapped")
	}
}
