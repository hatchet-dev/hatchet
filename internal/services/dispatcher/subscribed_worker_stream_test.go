package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stalledActionStream struct {
	entered     chan *contracts.AssignedAction
	release     chan struct{}
	unblockOnce sync.Once
}

func (s *stalledActionStream) Send(action *contracts.AssignedAction) error {
	s.entered <- action
	<-s.release
	return nil
}

func (s *stalledActionStream) unblock() {
	s.unblockOnce.Do(func() { close(s.release) })
}

func blockedWorker() (*subscribedWorker, *stalledActionStream) {
	stream := &stalledActionStream{entered: make(chan *contracts.AssignedAction, 100), release: make(chan struct{})}
	sender := rpcstream.NewSender[contracts.AssignedAction](context.Background(), stream)
	return newGRPCSubscribedWorker(sender, nil, uuid.New(), time.Second, nil), stream
}

func streamTestTask() *v1.V1TaskWithPayload {
	return &v1.V1TaskWithPayload{V1Task: &sqlcv1.V1Task{ID: 1, TenantID: uuid.New(), ExternalID: uuid.New(), StepID: uuid.New(), WorkflowID: uuid.New(), WorkflowVersionID: uuid.New(), WorkflowRunID: uuid.New()}}
}

func TestSubscribedWorker_DeadlineBeforeAdmissionIsRetryable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker, stream := blockedWorker()
		defer stream.unblock()
		if !worker.sendLock.Acquire() {
			t.Fatal("could not reserve lock")
		}
		defer worker.sendLock.Release()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := worker.StartTaskFromBulk(ctx, uuid.New(), streamTestTask(), nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unsent task must be retried, got %v", err)
		}
		if len(stream.entered) != 0 {
			t.Fatal("expired task reached transport")
		}
	})
}

func TestSubscribedWorker_CancellationRetainsSendOwnership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker, stream := blockedWorker()
		defer stream.unblock()
		ctx, cancel := context.WithCancel(context.Background())
		first := make(chan error, 1)
		go func() { first <- worker.sendToWorker(ctx, &contracts.AssignedAction{}) }()
		<-stream.entered
		cancel()
		if err := <-first; !errors.Is(err, errWorkerSendInterrupted) || !errors.Is(err, context.Canceled) {
			t.Fatalf("want uncertain cancellation, got %v", err)
		}
		if err := worker.sendToWorker(context.Background(), &contracts.AssignedAction{}); !errors.Is(err, errFlowControlActive) {
			t.Fatalf("active send must retain lock: %v", err)
		}
		if len(stream.entered) != 0 {
			t.Fatal("a second send reached the blocked transport")
		}
		stream.unblock()
		synctest.Wait()
		if err := worker.sendToWorker(context.Background(), &contracts.AssignedAction{}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSubscribedWorker_InFlightStartsAreNotRetried(t *testing.T) {
	for _, batch := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			worker, stream := blockedWorker()
			defer stream.unblock()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			result := make(chan error, 1)
			go func() {
				if batch {
					result <- worker.StartBatch(ctx, &contracts.AssignedAction{})
				} else {
					result <- worker.StartTaskFromBulk(ctx, uuid.New(), streamTestTask(), nil)
				}
			}()
			<-stream.entered
			if err := <-result; err != nil {
				t.Fatalf("in-flight start must not be retried: %v", err)
			}
			stream.unblock()
			synctest.Wait()
		})
	}
}

func TestSubscribedWorker_TransientStallAndPermanentStall(t *testing.T) {
	for _, transient := range []bool{true, false} {
		synctest.Test(t, func(t *testing.T) {
			worker, stream := blockedWorker()
			defer stream.unblock()
			results := make(chan error, 50)
			for range 50 {
				go func() { results <- worker.sendToWorker(context.Background(), &contracts.AssignedAction{}) }()
			}
			<-stream.entered
			synctest.Wait()
			if transient {
				time.Sleep(500 * time.Millisecond)
				stream.unblock()
				for range 50 {
					if err := <-results; err != nil {
						t.Fatalf("transient stall rejected action: %v", err)
					}
				}
			} else {
				for range 49 {
					if err := <-results; !errors.Is(err, errFlowControlActive) {
						t.Fatalf("stalled sibling must fail admission: %v", err)
					}
				}
				if len(stream.entered) != 0 {
					t.Fatal("stalled stream accepted another send")
				}
				stream.unblock()
				if err := <-results; err != nil {
					t.Fatalf("slow successful send failed: %v", err)
				}
			}
			synctest.Wait()
		})
	}
}

func TestSubscribedWorker_SlowSuccessIsNotAnErrorSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(old)
	defer provider.Shutdown(context.Background())
	synctest.Test(t, func(t *testing.T) {
		worker, stream := blockedWorker()
		defer stream.unblock()
		result := make(chan error, 1)
		go func() { result <- worker.sendToWorker(context.Background(), &contracts.AssignedAction{}) }()
		<-stream.entered
		time.Sleep(500 * time.Millisecond)
		stream.unblock()
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
	})
	foundSlow := false
	for _, span := range recorder.Ended() {
		if span.Status().Code == codes.Error {
			t.Fatalf("successful send produced error span %s", span.Name())
		}
		for _, event := range span.Events() {
			if event.Name == "slow worker send" {
				foundSlow = true
			}
		}
	}
	if !foundSlow {
		t.Fatal("slow-send event was not recorded")
	}
}

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
