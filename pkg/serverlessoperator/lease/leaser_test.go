//go:build !e2e && !load && !rampup && !integration

package lease

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

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
	tenantE = uuid.MustParse("00000000-0000-0000-0000-00000000000e")
)

// newLeaser builds a leaser with small claim batches and heartbeats it once, as Run does
// before its first tick: the database refuses claims from a process without a live row.
func newLeaser(t *testing.T, repo *memrepo.Repo, rec *fakeReconciler, processId uuid.UUID) *Leaser {
	t.Helper()

	return newLeaserWithConfig(t, repo, rec, Config{ProcessId: processId, ClaimBatch: 4})
}

func newLeaserWithConfig(t *testing.T, repo *memrepo.Repo, rec *fakeReconciler, cfg Config) *Leaser {
	t.Helper()

	l := zerolog.Nop()
	s := New(repo, rec, cfg, &l, Hooks{})

	require.NoError(t, s.Heartbeat(context.Background()))

	return s
}

func TestHeartbeatWritesOnlyTheProcessRow(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	l := zerolog.Nop()
	s := New(repo, &fakeReconciler{}, Config{ProcessId: pid}, &l, Hooks{})

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

func TestTickClaimsItsShareOfUnitsUpToFairWeight(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// Another live process already holds 4 endpoints' worth; 4 unowned units of weight 2
	// each. Two live processes share the 4 claimable units, so a tick claims at most 2, and
	// the weight fair share is 6 (total 12 over 2), so the weight budget is 6.
	repo.SetProcess(other, 2, 4, false)
	repo.SetLease(unit(tenantA, 0), nil, 2)
	repo.SetLease(unit(tenantB, 0), nil, 2)
	repo.SetLease(unit(tenantC, 0), nil, 2)
	repo.SetLease(unit(tenantD, 0), nil, 2)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0)}, rec.gained, "the unit budget is the claimable count divided by the live processes")

	// The walk starts at a random key and wraps around once, so at most two statements, each
	// asking for the whole budget.
	assert.LessOrEqual(t, len(repo.ClaimCalls()), 2)

	for _, n := range repo.ClaimCalls() {
		assert.Equal(t, int32(2), n, "each statement asks for the remaining budget")
	}

	assert.True(t, s.Ready())
	assert.Empty(t, rec.lost)

	// Next tick: 2 claimable units over 2 processes is a budget of 1; the weight fair share is
	// still 6 with 4 held, so C is claimed and D is left for the other process.
	require.NoError(t, s.Tick(context.Background()))
	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantB, 0), unit(tenantC, 0)}, rec.gained)

	// Now holding 6 of a fair share of 6: nothing more is claimed.
	require.NoError(t, s.Tick(context.Background()))
	assert.Len(t, rec.gained, 3)
	assert.Nil(t, repo.Lease(unit(tenantD, 0)).ProcessID, "the unit beyond the weight budget stays unowned")
}

func TestTickClaimsInBatchesBoundedByClaimBatch(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	// Alone with 10 claimable units: the budget is all 10, taken in statements of 4.
	units := make([]Unit, 0, 10)

	for i := 0; i < 10; i++ {
		u := unit(uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", i+1)), 0)
		units = append(units, u)
		repo.SetLease(u, nil, 1)
	}

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, units, s.Owned())
	assert.Len(t, rec.gained, 10)

	for _, n := range repo.ClaimCalls() {
		assert.LessOrEqual(t, n, int32(4), "no statement asks for more than ClaimBatch units")
	}

	// The walk starts at a random key and wraps around once, so at most one statement per
	// batch of 4 plus one for the wrap.
	assert.LessOrEqual(t, len(repo.ClaimCalls()), 4)
}

func TestTickCapsClaimsPerTick(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	for i := 0; i < 12; i++ {
		repo.SetLease(unit(uuid.New(), 0), nil, 1)
	}

	rec := &fakeReconciler{}
	s := newLeaserWithConfig(t, repo, rec, Config{ProcessId: pid, ClaimBatch: 4, MaxClaimPerTick: 5})

	require.NoError(t, s.Tick(context.Background()))
	assert.Len(t, s.Owned(), 5, "a tick never claims more than MaxClaimPerTick")

	require.NoError(t, s.Tick(context.Background()))
	assert.Len(t, s.Owned(), 10)
}

func TestExpiredClaimerCannotClaimUntilItHeartbeats(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), nil, 1)

	rec := &fakeReconciler{}
	l := zerolog.Nop()
	s := New(repo, rec, Config{ProcessId: pid, ClaimBatch: 4}, &l, Hooks{})

	// The process row expired (or was swept): the claim statement refuses the claimer.
	repo.SetProcess(pid, 0, 0, true)

	require.NoError(t, s.Tick(context.Background()))
	assert.Empty(t, rec.gained, "a process without a live row cannot take units")
	assert.Nil(t, repo.Lease(unit(tenantA, 0)).ProcessID)

	require.NoError(t, s.Heartbeat(context.Background()))
	require.NoError(t, s.Tick(context.Background()))
	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.gained)
}

func TestRevivedOwnerKeepsItsUnits(t *testing.T) {
	repo := memrepo.New()
	a := uuid.New()
	b := uuid.New()

	repo.SetLease(unit(tenantA, 0), nil, 1)

	recA := &fakeReconciler{}
	leaserA := newLeaser(t, repo, recA, a)
	require.NoError(t, leaserA.Tick(context.Background()))
	require.Equal(t, []Unit{unit(tenantA, 0)}, recA.gained)

	// A's row expires, B is about to take over, then A heartbeats first: the claim decides
	// liveness in its own snapshot and leaves A's unit alone.
	repo.SetProcess(a, 1, 1, true)

	recB := &fakeReconciler{}
	leaserB := newLeaser(t, repo, recB, b)

	require.NoError(t, leaserA.Heartbeat(context.Background()))
	require.NoError(t, leaserB.Tick(context.Background()))

	assert.Empty(t, recB.gained, "a live owner's unit is not transferred")
	assert.Equal(t, a, *repo.Lease(unit(tenantA, 0)).ProcessID)

	// Once A is really dead, B takes the unit and A learns the loss on its next tick.
	repo.SetProcess(a, 1, 1, true)
	require.NoError(t, leaserB.Tick(context.Background()))
	assert.Equal(t, []Unit{unit(tenantA, 0)}, recB.gained)

	require.NoError(t, leaserA.Heartbeat(context.Background()))
	require.NoError(t, leaserA.Tick(context.Background()))
	assert.Equal(t, []Unit{unit(tenantA, 0)}, recA.lost)
	assert.Empty(t, leaserA.Owned())
}

func TestSweptOwnerUnitsAreClaimable(t *testing.T) {
	repo := memrepo.New()
	dead := uuid.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), nil, 1)

	leaserDead := newLeaser(t, repo, &fakeReconciler{}, dead)
	require.NoError(t, leaserDead.Tick(context.Background()))
	require.Equal(t, dead, *repo.Lease(unit(tenantA, 0)).ProcessID)

	// The dead process's row expires and is swept before anyone took its unit over.
	repo.SetProcess(dead, 1, 1, true)

	rec := &fakeReconciler{}
	s := newLeaser(t, repo, rec, pid)
	require.NoError(t, s.sweep(context.Background()))

	assert.Nil(t, repo.Lease(unit(tenantA, 0)).ProcessID, "the sweep released the unit")

	require.NoError(t, s.Tick(context.Background()))
	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.gained)
}

func TestHeartbeatLapseKicksAReconcile(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	l := zerolog.Nop()
	s := New(repo, &fakeReconciler{}, Config{ProcessId: pid, TTL: 20 * time.Millisecond}, &l, Hooks{})

	require.NoError(t, s.Heartbeat(context.Background()))

	select {
	case <-s.kick:
		t.Fatal("a first heartbeat is not a lapse")
	default:
	}

	time.Sleep(40 * time.Millisecond)
	require.NoError(t, s.Heartbeat(context.Background()))

	select {
	case <-s.kick:
	default:
		t.Fatal("a heartbeat later than the TTL after the previous one must kick a reconcile")
	}
}

func TestRunRetriesTheInitialHeartbeat(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	repo.SetFailWrites(errors.New("database unreachable"))

	l := zerolog.Nop()
	s := New(repo, &fakeReconciler{}, Config{ProcessId: pid, HeartbeatInterval: time.Hour, RebalanceInterval: time.Hour, SweepInterval: time.Hour}, &l, Hooks{})
	s.initialBackoff = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)

	go func() { done <- s.Run(ctx) }()

	time.Sleep(30 * time.Millisecond)

	select {
	case err := <-done:
		t.Fatalf("Run returned while the database was unreachable: %v", err)
	default:
	}

	repo.SetFailWrites(nil)

	require.Eventually(t, s.Ready, 3*time.Second, 5*time.Millisecond, "the process becomes ready once the database answers")

	cancel()
	require.NoError(t, <-done)
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

// A window of empty units ahead of the populated ones must not hide them: empty units are
// not claimable, so the count sample skips them and the populated unit is claimed on the
// first tick even though the process already holds its fair share.
func TestTickClaimsPopulatedUnitsBehindEmptyOnes(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()

	repo.SetLease(unit(tenantA, 0), &pid, 1)
	repo.SetLease(unit(tenantA, 1), nil, 0)
	repo.SetLease(unit(tenantB, 0), nil, 0)
	repo.SetLease(unit(tenantC, 0), nil, 0)
	repo.SetLease(unit(tenantD, 0), nil, 0)
	repo.SetLease(unit(tenantE, 0), nil, 3)

	rec := &fakeReconciler{}
	s := newLeaserWithConfig(t, repo, rec, Config{ProcessId: pid, ClaimBatch: 4, MaxClaimPerTick: 4})

	for i := 0; i < 3; i++ {
		require.NoError(t, s.Tick(context.Background()))
		require.NoError(t, s.Heartbeat(context.Background()))
	}

	assert.Equal(t, []Unit{unit(tenantA, 0), unit(tenantE, 0)}, s.Owned(), "the populated unit behind the empty window is claimed")

	for _, u := range []Unit{unit(tenantA, 1), unit(tenantB, 0), unit(tenantC, 0), unit(tenantD, 0)} {
		assert.Nil(t, repo.Lease(u).ProcessID, "an empty unit is never claimed")
	}
}

// A full count sample means the backlog is at least a tick's worth: the process takes a full
// MaxClaimPerTick rather than the sample divided by the fleet, so a cold start does not crawl
// on a large fleet, and the count never has to cover the whole backlog.
func TestSaturatedSampleClaimsAFullTick(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	repo.SetProcess(other, 0, 0, false)

	for i := 0; i < 6; i++ {
		repo.SetLease(unit(uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", i+10)), 0), nil, 1)
	}

	rec := &fakeReconciler{}
	s := newLeaserWithConfig(t, repo, rec, Config{ProcessId: pid, ClaimBatch: 2, MaxClaimPerTick: 4})

	require.NoError(t, s.Tick(context.Background()))

	assert.Len(t, rec.gained, 4, "a saturated sample claims MaxClaimPerTick, not its share of the sample")

	// The sample is exact once the backlog fits in it: 4 held here, 0 there, 2 claimable is
	// a fair share of 3, so the rest is the other process's to claim.
	require.NoError(t, s.Tick(context.Background()))
	assert.Len(t, rec.gained, 4)
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

// A unit with deliveries in flight is shed after the idle ones, not never: when the idle
// units cannot cover the excess, the busy one goes.
func TestTickShedsBusyUnitWhenIdleOnesDoNotCover(t *testing.T) {
	repo := memrepo.New()
	pid := uuid.New()
	other := uuid.New()

	// 5 held (busy 1, idle 4) against a process with nothing: fair share 3, threshold 3.6,
	// excess 2. The idle unit does not fit; the busy one does.
	repo.SetLease(unit(tenantA, 0), &pid, 1)
	repo.SetLease(unit(tenantB, 0), &pid, 4)

	rec := &fakeReconciler{inflight: map[Unit]int{unit(tenantA, 0): 1}}
	s := newLeaser(t, repo, rec, pid)

	require.NoError(t, s.Tick(context.Background()))
	require.Len(t, rec.gained, 2)

	repo.SetProcess(other, 0, 0, false)

	require.NoError(t, s.Tick(context.Background()))

	assert.Equal(t, []Unit{unit(tenantA, 0)}, rec.lost)
	assert.Equal(t, []Unit{unit(tenantB, 0)}, s.Owned())
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
