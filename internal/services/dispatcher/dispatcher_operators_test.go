package dispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

type stubOperator struct {
	workerId uuid.UUID
}

func (s *stubOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	return nil
}

func (s *stubOperator) WorkerId() uuid.UUID { return s.workerId }

func (s *stubOperator) Cleanup() {}

func (s *stubOperator) Drain() {}

// TestListenForOperatorsReconcilesWorkerEntries verifies the dispatcher mirrors the
// manager's full operator set: repeated reports are idempotent, operators that leave the
// set are removed, and regular (gRPC) worker entries are never touched.
func TestListenForOperatorsReconcilesWorkerEntries(t *testing.T) {
	d := &DispatcherImpl{workers: &workers{}}

	op1 := &stubOperator{workerId: uuid.New()}
	op2 := &stubOperator{workerId: uuid.New()}

	grpcWorkerId := uuid.New()
	d.workers.Add(grpcWorkerId, "session", newGRPCSubscribedWorker(nil, nil, grpcWorkerId, time.Second, nil))

	ch := make(chan []operator.Operator)
	done := make(chan struct{})

	go func() {
		d.listenForOperators(ch)
		close(done)
	}()

	ch <- []operator.Operator{op1, op2}
	// resending the same set must be idempotent (no duplicate sessions)
	ch <- []operator.Operator{op1, op2}
	// op2 leaves the reported set
	ch <- []operator.Operator{op1}
	close(ch)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listenForOperators did not exit when the channel closed")
	}

	ws, err := d.workers.Get(op1.WorkerId())

	if err != nil {
		t.Fatalf("expected op1 worker to be registered: %v", err)
	}

	if len(ws) != 1 {
		t.Fatalf("expected exactly one session for op1's worker, got %d", len(ws))
	}

	if _, err := d.workers.Get(op2.WorkerId()); err == nil {
		t.Fatal("expected op2 worker to be removed after leaving the reported set")
	}

	if _, err := d.workers.Get(grpcWorkerId); err != nil {
		t.Fatalf("expected gRPC worker entry to be untouched: %v", err)
	}
}
