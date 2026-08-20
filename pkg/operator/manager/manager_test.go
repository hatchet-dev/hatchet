//go:build !e2e && !load && !rampup && !integration

package manager

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeOperator struct {
	workerId  uuid.UUID
	cleanedUp chan struct{}
	release   chan struct{} // if non-nil, Cleanup blocks until it is closed
}

func (f *fakeOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	return nil
}

func (f *fakeOperator) WorkerId() uuid.UUID { return f.workerId }

func (f *fakeOperator) Cleanup() {
	if f.release != nil {
		<-f.release
	}

	close(f.cleanedUp)
}

func (f *fakeOperator) Drain() {
	f.Cleanup()
}

func newTestManager() *OperatorManager {
	l := zerolog.Nop()
	return NewOperatorManager(uuid.New(), &l, nil, nil, nil)
}

// reconcileAndReport runs a single reconcile pass and returns the full active set it reports
// on the operators channel.
func reconcileAndReport(t *testing.T, om *OperatorManager, claimed []*sqlcv1.V1Operator) []operator.Operator {
	t.Helper()

	done := make(chan struct{})

	go func() {
		om.reconcileOperators(context.Background(), claimed)
		close(done)
	}()

	select {
	case ops := <-om.operatorsCh:
		<-done
		return ops
	case <-time.After(time.Second):
		t.Fatal("reconcile did not report an operator set")
		return nil
	}
}

// TestReconcileTearsDownUnclaimedOperators verifies an operator that disappears from the
// claim result is removed from the active set and cleaned up.
func TestReconcileTearsDownUnclaimedOperators(t *testing.T) {
	om := newTestManager()

	opId := uuid.New()
	fake := &fakeOperator{workerId: uuid.New(), cleanedUp: make(chan struct{})}
	om.operators.Store(opId, fake)

	claimed := []*sqlcv1.V1Operator{{ID: opId}}

	if ops := reconcileAndReport(t, om, claimed); len(ops) != 1 {
		t.Fatalf("expected 1 active operator while claimed, got %d", len(ops))
	}

	if ops := reconcileAndReport(t, om, nil); len(ops) != 0 {
		t.Fatalf("expected operator to be torn down once no longer claimed, got %d active", len(ops))
	}

	select {
	case <-fake.cleanedUp:
	case <-time.After(time.Second):
		t.Fatal("operator Cleanup was not invoked after teardown")
	}
}

// TestDrainingWorkerKeepsHeartbeating verifies a torn-down operator's worker stays in the
// heartbeat set until its Cleanup (drain) completes, then drops out.
func TestDrainingWorkerKeepsHeartbeating(t *testing.T) {
	om := newTestManager()

	opId := uuid.New()
	release := make(chan struct{})
	fake := &fakeOperator{workerId: uuid.New(), cleanedUp: make(chan struct{}), release: release}
	om.operators.Store(opId, fake)

	reconcileAndReport(t, om, nil)

	// the operator is now draining (Cleanup is blocked on release): its worker must still
	// be heartbeated
	if ids := om.heartbeatWorkerIds(); len(ids) != 1 || ids[0] != fake.workerId {
		t.Fatalf("expected draining worker %s in heartbeat set, got %v", fake.workerId, ids)
	}

	close(release)

	select {
	case <-fake.cleanedUp:
	case <-time.After(time.Second):
		t.Fatal("operator Cleanup did not complete after release")
	}

	// once the drain finishes, the worker drops out of the heartbeat set
	deadline := time.After(time.Second)

	for len(om.heartbeatWorkerIds()) != 0 {
		select {
		case <-deadline:
			t.Fatal("drained worker was never removed from the heartbeat set")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
