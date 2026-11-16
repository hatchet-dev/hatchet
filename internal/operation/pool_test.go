package operation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// fakeControl pauses the tenants in paused and runs await as the budget wait.
type fakeControl struct {
	v1.TenantControlRepository

	paused map[uuid.UUID]bool
	await  func(ctx context.Context) error
}

func (f fakeControl) IsPaused(id uuid.UUID) bool { return f.paused[id] }

func (f fakeControl) AwaitBudget(ctx context.Context, _ uuid.UUID) error {
	if f.await == nil {
		return nil
	}

	return f.await(ctx)
}

func TestTenantOperationPoolSkipsPausedTenants(t *testing.T) {
	active := uuid.New()
	paused := uuid.New()
	isPaused := map[uuid.UUID]bool{paused: true}

	pool := &TenantOperationPool{
		control: fakeControl{paused: isPaused},
		method:  func(ctx context.Context, id string) (bool, error) { return false, nil },
		ql:      &l,
	}
	t.Cleanup(func() {
		pool.ops.Range(func(_ uuid.UUID, op *SerialOperation) bool {
			op.Stop()
			return true
		})
	})

	hasOp := func(id uuid.UUID) bool {
		_, ok := pool.ops.Load(id)
		return ok
	}

	pool.setTenants([]uuid.UUID{active, paused})
	require.True(t, hasOp(active))
	require.False(t, hasOp(paused))

	isPaused[paused] = false
	isPaused[active] = true

	pool.setTenants([]uuid.UUID{active, paused})
	require.False(t, hasOp(active), "pausing a tenant stops its operation")
	require.True(t, hasOp(paused), "resuming a tenant starts its operation")
}

func TestTenantOperationPoolYieldsBeforeEachRun(t *testing.T) {
	const timeout = 200 * time.Millisecond

	tenantId := uuid.New()
	ran := make(chan time.Duration, 1)

	pool := &TenantOperationPool{
		control: fakeControl{await: func(ctx context.Context) error {
			time.Sleep(2 * timeout)
			return nil
		}},
		method: func(ctx context.Context, id string) (bool, error) {
			deadline, _ := ctx.Deadline()
			ran <- time.Until(deadline)
			return false, nil
		},
		timeout: timeout,
		ql:      &l,
	}

	op := pool.getOperation(tenantId)
	t.Cleanup(op.Stop)

	op.RunOrContinue(&l)

	select {
	case left := <-ran:
		require.Greater(t, left, timeout/2, "the budget wait does not count against the run's timeout")
	case <-time.After(5 * time.Second):
		t.Fatal("the method did not run")
	}
}

func TestTenantOperationPoolStoppedDuringYieldSkipsRun(t *testing.T) {
	var ran atomic.Bool
	yielding := make(chan struct{})

	pool := &TenantOperationPool{
		control: fakeControl{await: func(ctx context.Context) error {
			close(yielding)
			<-ctx.Done()
			return ctx.Err()
		}},
		method: func(ctx context.Context, id string) (bool, error) {
			ran.Store(true)
			return false, nil
		},
		timeout: time.Second,
		ql:      &l,
	}

	op := pool.getOperation(uuid.New())
	op.RunOrContinue(&l)

	<-yielding
	op.Stop()

	require.Eventually(t, func() bool {
		op.mu.RLock()
		defer op.mu.RUnlock()
		return !op.isRunning
	}, time.Second, 10*time.Millisecond)
	require.False(t, ran.Load(), "a run whose yield ended with the operation does not call the method")
}
