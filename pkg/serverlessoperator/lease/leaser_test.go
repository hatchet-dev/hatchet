//go:build !e2e && !load && !rampup && !integration

package lease

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

type fakeReconciler struct {
	inflight map[Unit]int
	gained   []Unit
	lost     []Unit
	mu       sync.Mutex
}

func (f *fakeReconciler) UnitsGained(_ context.Context, units []Unit) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gained = append(f.gained, units...)
}

func (f *fakeReconciler) UnitsLost(_ context.Context, units []Unit) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lost = append(f.lost, units...)
}

func (f *fakeReconciler) InFlight(unit Unit) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inflight[unit]
}

func unit(tenant uuid.UUID, shard int32) Unit {
	return Unit{TenantId: tenant, Shard: shard}
}

// tenantIds are fixed so ordering (which the memrepo makes deterministic) is predictable.
var (
	tenantA = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	tenantB = uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	tenantC = uuid.MustParse("00000000-0000-0000-0000-00000000000c")
	tenantD = uuid.MustParse("00000000-0000-0000-0000-00000000000d")
)

func newLeaser(t *testing.T, repo *memrepo.Repo, rec *fakeReconciler, processId uuid.UUID) *Leaser {
	t.Helper()

	l := zerolog.Nop()

	return New(repo, rec, Config{ProcessId: processId, ClaimBatch: 4}, &l, Hooks{})
}

func TestHeartbeatWritesOnlyTheProcessRow(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	s := newLeaser(t, repo, &fakeReconciler{}, pid)

	require.NoError(t, s.Heartbeat(context.Background()))

	require.Len(t, repo.Heartbeats(), 1)
	assert.Equal(t, pid, repo.Heartbeats()[0].ProcessId)
	assert.Equal(t, defaultTTL, repo.Heartbeats()[0].TTL)
	assert.Equal(t, int32(0), repo.Heartbeats()[0].UnitCount)
	assert.Empty(t, repo.StatusWrites())
	assert.Empty(t, repo.ActionWrites())
	assert.Empty(t, repo.ClaimCalls())
	assert.Empty(t, repo.ShedCalls())
}

func TestTickClaimsUpToFairShare(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// Another live process already holds 4 endpoints' worth; 4 unowned units of weight 2
	// each. Total 12 over 2 processes: fair share 6, budget 6. With a claim batch of 4 the
	// first batch is capped at 4 units (budget 6 units at most) and claims all four, then
	// budget is spent; so the batch cap is exercised by a batch of 2 below instead.
	repo.SetProcess(other, 2, 4, false)
	repo.SetLease(unit(tenantA, 0), nil, 2)
	repo.SetLease(unit(tenantB, 0), nil, 2)
	repo.SetLease(unit(tenantC, 0), nil, 2)
	repo.SetLease(unit(tenantD, 0), nil, 2)

	rec := &fakeReconciler{}
	l := zerolog.Nop()
	s := New(repo, rec, Config{ProcessId: pid, ClaimBatch: 2}, &l, Hooks{})

	require.NoError(t, s.Tick(context.Background()))

	// Batches of 2: A and B (budget 2 left), then a batch capped at 2 claims C and D. The
	// overshoot is at most one batch; the next tick sheds back to fair share if it crosses
	// the hysteresis threshold.
	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0), unit(tenantC, 0), unit(tenantD, 0)}, rec.gained)
	assert.Equal(t, []int32{2, 2}, repo.ClaimCalls(), "claims run in small batches until the budget is spent")
	assert.True(t, s.Ready())
	assert.Empty(t, rec.lost)
}

func TestTickStopsClaimingWhenBudgetIsSpent(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// Fair share 6 with batches of 1: A, B, C are claimed (budget 6, 4, 2, 0) and D stays.
	repo.SetProcess(other, 2, 4, false)
	repo.SetLease(unit(tenantA, 0), nil, 2)
	repo.SetLease(unit(tenantB, 0), nil, 2)
	repo.SetLease(unit(tenantC, 0), nil, 2)
	repo.SetLease(unit(tenantD, 0), nil, 2)

	rec := &fakeReconciler{}
	l := zerolog.Nop()
	s := New(repo, rec, Config{ProcessId: pid, ClaimBatch: 1}, &l, Hooks{})

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0), unit(tenantC, 0)}, rec.gained)
	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0), unit(tenantC, 0)}, s.Owned())
	assert.Nil(t, repo.Lease(unit(tenantD, 0)).ProcessID, "unit beyond the budget stays unowned")
	assert.Equal(t, []int32{1, 1, 1}, repo.ClaimCalls())
}

func TestTickClaimsOneUnitWhenHoldingNothing(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// The other process holds everything but one unit and its weight dwarfs the unowned
	// unit, so the budget is zero; the floor of one unit still claims it.
	repo.SetProcess(other, 10, 100, false)
	repo.SetLease(unit(tenantA, 0), nil, 1)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.gained)
}

func TestTickTakesOverDeadProcessUnits(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	dead := uuid.New()

	repo.SetProcess(dead, 1, 3, true)
	repo.SetLease(unit(tenantA, 0), &dead, 3)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.gained)
	assert.Equal(t, pid, *repo.Lease(unit(tenantA, 0)).ProcessID)
}

func TestTickShedsAboveFairShareSmallestIdleFirst(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// This process holds 10 (weights 1, 2, 3, 4) and is alone, so the first tick only
	// learns the units. Then a second process appears with nothing: fair share 5, the
	// hysteresis threshold is 6, so shed excess 5: the idle weight-1 unit, then weight 2;
	// the weight-3 unit has a delivery in flight and is skipped; weight 4 does not fit.
	repo.SetLease(unit(tenantA, 0), &pid, 1)
	repo.SetLease(unit(tenantB, 0), &pid, 2)
	repo.SetLease(unit(tenantC, 0), &pid, 3)
	repo.SetLease(unit(tenantD, 0), &pid, 4)

	rec := &fakeReconciler{inflight: map[Unit]int{unit(tenantC, 0): 1}}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))
	require.Len(t, rec.gained, 4)
	require.Empty(t, repo.ShedCalls())

	repo.SetProcess(other, 0, 0, false)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0)}, rec.lost)
	assert.Equal(t, []Unit{unit(tenantC, 0), unit(tenantD, 0)}, s.Owned())
	assert.Nil(t, repo.Lease(unit(tenantA, 0)).ProcessID)
	assert.Nil(t, repo.Lease(unit(tenantB, 0)).ProcessID)
	assert.Equal(t, pid, *repo.Lease(unit(tenantC, 0)).ProcessID)
	assert.Empty(t, repo.StatusWrites())
	assert.Empty(t, repo.ActionWrites())
}

func TestTickNeverShedsWhenAlone(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), &pid, 1)
	repo.SetLease(unit(tenantB, 0), &pid, 50)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Empty(t, rec.lost)
	assert.Empty(t, repo.ShedCalls())
}

func TestTickWithinHysteresisDoesNotShed(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// 11 here, 9 there: fair share 10, threshold 12, no shed.
	repo.SetProcess(other, 1, 9, false)
	repo.SetLease(unit(tenantA, 0), &pid, 5)
	repo.SetLease(unit(tenantB, 0), &pid, 6)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Empty(t, rec.lost)
	assert.Empty(t, repo.ShedCalls())
}

func TestTickReportsUnitsTakenAway(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), &pid, 1)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))
	require.Equal(t, []Unit{unit(tenantA, 0)}, rec.gained)

	// Another process declared this one dead and took the unit.
	thief := uuid.New()
	repo.SetLease(unit(tenantA, 0), &thief, 1)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.lost)
	assert.Empty(t, s.Owned())
}

func TestReleaseAndDelete(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), &pid, 1)

	s := newLeaser(t, repo, &fakeReconciler{}, pid)
	require.NoError(t, s.Heartbeat(context.Background()))
	require.NoError(t, s.Tick(context.Background()))

	released, err := s.Release(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []Unit{unit(tenantA, 0)}, released)
	assert.Equal(t, 1, repo.ReleaseAlls())
	assert.Nil(t, repo.Lease(unit(tenantA, 0)).ProcessID)
	assert.Empty(t, s.Owned())

	require.NoError(t, s.DeleteProcess(context.Background()))
	assert.Equal(t, []uuid.UUID{pid}, repo.DeletedProcs())
}
