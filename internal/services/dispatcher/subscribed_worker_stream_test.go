package dispatcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
)

type fakeActionStream struct {
	sent []*contracts.AssignedAction
}

func (f *fakeActionStream) Send(action *contracts.AssignedAction) error {
	f.sent = append(f.sent, action)
	return nil
}

// Once the Listen handler has returned, its stream must never be written to again: the HTTP/2
// server panics on a write after the handler is done. A send that loses the race with the
// handler returning has to come back as an error instead.
func TestSubscribedWorker_SendAfterStreamCloseReturnsError(t *testing.T) {
	stream := &fakeActionStream{}
	sender := rpcstream.NewSender[contracts.AssignedAction](context.Background(), stream)
	worker := newGRPCSubscribedWorker(sender, nil, uuid.New(), time.Second, nil)

	if err := worker.sendToWorker(context.Background(), &contracts.AssignedAction{}); err != nil {
		t.Fatalf("expected send on an open stream to succeed, got %v", err)
	}

	sender.Close()

	err := worker.sendToWorker(context.Background(), &contracts.AssignedAction{})

	if !errors.Is(err, rpcstream.ErrClosed) {
		t.Fatalf("expected ErrClosed after the stream is closed, got %v", err)
	}

	if len(stream.sent) != 1 {
		t.Fatalf("expected exactly one action to reach the stream, got %d", len(stream.sent))
	}
}

// cancelOnSendStream completes every write and cancels the caller's context on
// the way out, so the send result and the context cancellation are both ready
// when sendToWorker picks one.
type cancelOnSendStream struct {
	cancel context.CancelFunc
	sent   int
}

func (f *cancelOnSendStream) Send(_ *contracts.AssignedAction) error {
	f.sent++
	f.cancel()
	return nil
}

// A write that returned before the context ended is a completed send: the worker
// has the action, so it must not be reported as a failure (which would requeue a
// task that is already running).
func TestSubscribedWorker_SendCompletedBeforeCancelIsNotAFailure(t *testing.T) {
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		stream := &cancelOnSendStream{cancel: cancel}
		sender := rpcstream.NewSender[contracts.AssignedAction](context.Background(), stream)
		worker := newGRPCSubscribedWorker(sender, nil, uuid.New(), time.Second, nil)

		err := worker.sendToWorker(ctx, &contracts.AssignedAction{})
		cancel()

		if err != nil {
			t.Fatalf("iteration %d: completed write reported as failed send: %v", i, err)
		}

		if stream.sent != 1 {
			t.Fatalf("iteration %d: expected exactly one write, got %d", i, stream.sent)
		}
	}
}

type blockingStream struct {
	entered  chan struct{}
	release  chan struct{}
	returned atomic.Bool
}

func (f *blockingStream) Send(_ *contracts.AssignedAction) error {
	close(f.entered)
	<-f.release
	f.returned.Store(true)
	return nil
}

// A context that ends while the write is still in flight is still reported as
// a failed send, since nothing says whether the worker will get the action.
func TestSubscribedWorker_CancelDuringInFlightSendIsAFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &blockingStream{entered: make(chan struct{}), release: make(chan struct{})}
	defer close(stream.release)
	sender := rpcstream.NewSender[contracts.AssignedAction](context.Background(), stream)
	worker := newGRPCSubscribedWorker(sender, nil, uuid.New(), time.Second, nil)

	errCh := make(chan error, 1)
	go func() { errCh <- worker.sendToWorker(ctx, &contracts.AssignedAction{}) }()

	select {
	case <-stream.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("stream write was never started")
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled while the write is in flight, got %v", err)
		}
		if stream.returned.Load() {
			t.Fatal("write should still be in flight when the error is reported")
		}
	case <-time.After(sendResultGrace + 2*time.Second):
		t.Fatal("sendToWorker did not return after the context was cancelled")
	}
}
