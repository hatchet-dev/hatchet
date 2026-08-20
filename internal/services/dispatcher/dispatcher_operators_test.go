package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type stubOperator struct {
	workerId uuid.UUID

	mu             sync.Mutex
	receivedTypes  []contracts.ActionType
	handleActionFn func(ctx context.Context, action *contracts.AssignedAction) error
}

func (s *stubOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	s.mu.Lock()
	s.receivedTypes = append(s.receivedTypes, action.ActionType)
	s.mu.Unlock()

	if s.handleActionFn != nil {
		return s.handleActionFn(ctx, action)
	}

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

// TestSubscribedWorker_CancelTask_RoutesToOperator verifies that cancelling a task assigned
// to an operator-backed worker actually reaches the operator's HandleAction with
// CANCEL_STEP_RUN (rather than being silently swallowed by the stream-worker flow-control
// path, which operator-backed workers don't go through at all).
func TestSubscribedWorker_CancelTask_RoutesToOperator(t *testing.T) {
	op := &stubOperator{workerId: uuid.New()}
	worker := newOperatorSubscribedWorker(op.workerId, nil, op)

	task := &sqlcv1.V1Task{
		ID:                1,
		ExternalID:        uuid.New(),
		StepID:            uuid.New(),
		StepReadableID:    "step-1",
		ActionID:          "action-1",
		WorkflowID:        uuid.New(),
		WorkflowVersionID: uuid.New(),
		WorkflowRunID:     uuid.New(),
	}

	err := worker.CancelTask(context.Background(), uuid.New(), task, 0, nil)
	if err != nil {
		t.Fatalf("expected CancelTask to succeed for an operator-backed worker, got: %v", err)
	}

	op.mu.Lock()
	defer op.mu.Unlock()

	if len(op.receivedTypes) != 1 || op.receivedTypes[0] != contracts.ActionType_CANCEL_STEP_RUN {
		t.Fatalf("expected operator to receive exactly one CANCEL_STEP_RUN action, got: %v", op.receivedTypes)
	}
}

// TestSubscribedWorker_CancelTask_OperatorErrorPropagates verifies that a real failure from
// the operator (as opposed to a flow-control/timeout condition) is surfaced to the caller
// instead of being swallowed.
func TestSubscribedWorker_CancelTask_OperatorErrorPropagates(t *testing.T) {
	wantErr := context.Canceled // any non-flow-control, non-deadline error stand-in

	op := &stubOperator{
		workerId: uuid.New(),
		handleActionFn: func(ctx context.Context, action *contracts.AssignedAction) error {
			return wantErr
		},
	}
	worker := newOperatorSubscribedWorker(op.workerId, nil, op)

	task := &sqlcv1.V1Task{
		ID:                1,
		ExternalID:        uuid.New(),
		StepID:            uuid.New(),
		StepReadableID:    "step-1",
		ActionID:          "action-1",
		WorkflowID:        uuid.New(),
		WorkflowVersionID: uuid.New(),
		WorkflowRunID:     uuid.New(),
	}

	err := worker.CancelTask(context.Background(), uuid.New(), task, 0, nil)
	if err == nil {
		t.Fatal("expected CancelTask to propagate the operator's error, got nil")
	}
}
