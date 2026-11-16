//go:build !e2e && !load && !rampup && !integration

package fairpool

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

func TestBudgetArithmetic(t *testing.T) {
	const rate = 2.0 // held-seconds per second, so the window holds 20

	start := time.Unix(1000, 0)
	var a account

	// an unused budget is full
	require.Zero(t, a.wait(rate, start))
	require.InDelta(t, 20, a.balance, 1e-9)

	// debit
	a.debit(25, rate, start)
	require.InDelta(t, -5, a.balance, 1e-9)
	require.Equal(t, 2500*time.Millisecond, a.wait(rate, start))

	// refill at the rate
	require.Equal(t, 1500*time.Millisecond, a.wait(rate, start.Add(time.Second)))
	require.Zero(t, a.wait(rate, start.Add(3*time.Second)))
	require.InDelta(t, 1, a.balance, 1e-9)

	// refill stops at the window
	a.refill(rate, start.Add(time.Hour))
	require.InDelta(t, 20, a.balance, 1e-9)

	// debt is floored at the window, so a wait is at most the window
	a.debit(1000, rate, start.Add(time.Hour))
	require.InDelta(t, -20, a.balance, 1e-9)
	require.Equal(t, budgetWindow, a.wait(rate, start.Add(time.Hour)))
}

func TestBudgetRateCutKeepsAtMostWindowOfDebt(t *testing.T) {
	// a pool of 200: 99 percent is 198 held-seconds per second, 1 percent is 2
	const high, low = 198.0, 2.0

	start := time.Unix(1000, 0)
	var a account

	a.debit(1e6, high, start)
	require.InDelta(t, -1980, a.balance, 1e-9, "at the floor of the high rate")

	require.Equal(t, budgetWindow, a.wait(low, start), "the debt is clamped to the window of the new rate")
	require.InDelta(t, -20, a.balance, 1e-9)

	require.Equal(t, 5*time.Second, a.wait(low, start.Add(5*time.Second)))
}

func TestBudgetNoBudgetForgetsDebt(t *testing.T) {
	const rate = 2.0

	start := time.Unix(1000, 0)
	var a account

	a.debit(1000, rate, start)
	require.Equal(t, budgetWindow, a.wait(rate, start))

	// a release while there is no budget: nothing to wait for, and the old debt is gone
	a.debit(1000, 0, start.Add(time.Second))
	require.Zero(t, a.wait(0, start.Add(time.Second)))

	// the budget comes back full
	require.Zero(t, a.wait(rate, start.Add(2*time.Second)))
	require.InDelta(t, 20, a.balance, 1e-9)
}

func newBudgetGate(rate float64) *gate {
	return newGate(0, 10, rate, time.Second, "budget-test-"+uuid.NewString(), nil)
}

// spend checks out and releases one connection for key that was held for held.
func spend(t *testing.T, g *gate, tenantID uuid.UUID, held time.Duration) {
	t.Helper()

	hold, err := g.pin(context.Background(), tenantID.String(), tenantID, noopOpen)
	require.NoError(t, err)

	hold.start = hold.start.Add(-held)
	hold.release()
}

// timedAwait runs awaitBudget for tenantID and returns how long it took.
func timedAwait(ctx context.Context, g *gate, tenantID uuid.UUID) (time.Duration, error) {
	start := time.Now()
	err := g.awaitBudget(ctx, tenantID.String(), tenantID)

	return time.Since(start), err
}

func TestAwaitBudget(t *testing.T) {
	// 1 held-second per second: the window holds 10, so 10.5 held seconds leave 0.5 s of debt
	g := newBudgetGate(1)
	noisy := uuid.New()
	other := uuid.New()

	spend(t, g, noisy, 10500*time.Millisecond)

	start := time.Now()
	hold, err := g.pin(context.Background(), noisy.String(), noisy, noopOpen)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 50*time.Millisecond, "a checkout is metered, never delayed")
	hold.release()

	elapsed, err := timedAwait(context.Background(), g, other)
	require.NoError(t, err)
	require.Less(t, elapsed, 50*time.Millisecond, "another tenant is unaffected")

	elapsed, err = timedAwait(context.Background(), g, noisy)
	require.NoError(t, err)
	require.GreaterOrEqual(t, elapsed, 400*time.Millisecond)
	require.Less(t, elapsed, 1500*time.Millisecond)

	// the wait paid the debt off
	elapsed, err = timedAwait(context.Background(), g, noisy)
	require.NoError(t, err)
	require.Less(t, elapsed, 50*time.Millisecond)
}

func TestAwaitBudgetEndsWithContext(t *testing.T) {
	g := newBudgetGate(1)
	noisy := uuid.New()

	spend(t, g, noisy, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	elapsed, err := timedAwait(ctx, g, noisy)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, elapsed, 500*time.Millisecond)
}

func TestAwaitBudgetWaitersRecheckTheBalance(t *testing.T) {
	g := newBudgetGate(1)
	noisy := uuid.New()

	// 0.5 s of debt: one reading would release every waiter after 0.5 s
	spend(t, g, noisy, 10500*time.Millisecond)

	const waiters = 20

	elapsed := make(chan time.Duration, waiters)

	for range waiters {
		go func() {
			d, err := timedAwait(context.Background(), g, noisy)
			assert.NoError(t, err)
			elapsed <- d
		}()
	}

	// while they wait, work that is not deferred releases another second of held time
	time.Sleep(250 * time.Millisecond)
	spend(t, g, noisy, time.Second)

	for range waiters {
		d := <-elapsed
		require.GreaterOrEqual(t, d, 1300*time.Millisecond, "each waiter re-checks the balance after its sleep")
		require.Less(t, d, 3*time.Second)
	}
}

func TestAwaitBudgetWaitsAtMostWindow(t *testing.T) {
	t.Parallel()

	g := newBudgetGate(1)
	noisy := uuid.New()

	// work that is not deferred keeps the tenant at its floor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spend(t, g, noisy, time.Minute)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				spend(t, g, noisy, time.Second)
			}
		}
	}()

	elapsed, err := timedAwait(context.Background(), g, noisy)
	require.NoError(t, err)
	require.GreaterOrEqual(t, elapsed, budgetWindow)
	require.Less(t, elapsed, budgetWindow+time.Second)
}

func TestBudgetOverrideReplacesDefault(t *testing.T) {
	limited := uuid.New()
	exempt := uuid.New()

	// no default budget; an override gives one tenant a budget
	g := newBudgetGate(0)
	g.budgets.Store(&map[uuid.UUID]float64{limited: 1})

	spend(t, g, limited, 10500*time.Millisecond)
	spend(t, g, exempt, time.Hour)

	elapsed, err := timedAwait(context.Background(), g, limited)
	require.NoError(t, err)
	require.GreaterOrEqual(t, elapsed, 400*time.Millisecond)

	elapsed, err = timedAwait(context.Background(), g, exempt)
	require.NoError(t, err)
	require.Less(t, elapsed, 50*time.Millisecond)

	// a default budget; an override of no budget exempts one tenant
	g = newBudgetGate(1)
	g.budgets.Store(&map[uuid.UUID]float64{exempt: 0})

	spend(t, g, exempt, time.Hour)

	elapsed, err = timedAwait(context.Background(), g, exempt)
	require.NoError(t, err)
	require.Less(t, elapsed, 50*time.Millisecond)
}

func TestBudgetOverrideChanges(t *testing.T) {
	tenantID := uuid.New()
	key := tenantID.String()

	// a pool of 10 at 99 percent, at its floor
	g := newBudgetGate(0)
	g.budgets.Store(&map[uuid.UUID]float64{tenantID: 9.9})
	spend(t, g, tenantID, time.Hour)

	// cut to 1 percent: the next wait is at most the window
	g.budgets.Store(&map[uuid.UUID]float64{tenantID: 0.1})
	require.Equal(t, budgetWindow, g.budgetWait(key, tenantID, time.Now()))

	// no budget, with work released meanwhile, then a budget again: the old debt is gone
	g.budgets.Store(&map[uuid.UUID]float64{tenantID: 0})
	spend(t, g, tenantID, time.Second)

	g.budgets.Store(&map[uuid.UUID]float64{tenantID: 0.1})
	require.Zero(t, g.budgetWait(key, tenantID, time.Now()))
}

func heldSeconds(t *testing.T, poolName, key string) (float64, bool) {
	t.Helper()

	ch := make(chan promclient.Metric)
	go func() {
		prometheus.FairpoolHeldSeconds.Collect(ch)
		close(ch)
	}()

	value, found := 0.0, false

	for metric := range ch {
		var m dto.Metric
		require.NoError(t, metric.Write(&m))

		labels := map[string]string{}
		for _, label := range m.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		if labels["pool"] == poolName && labels["tenant_id"] == key {
			value, found = m.GetCounter().GetValue(), true
		}
	}

	return value, found
}

func TestHeldSecondsPerBucket(t *testing.T) {
	g := newBudgetGate(0)
	tenantA := uuid.New()
	tenantB := uuid.New()

	spend(t, g, tenantA, 2*time.Second)
	spend(t, g, tenantA, 3*time.Second)
	spend(t, g, tenantB, time.Second)

	shared, err := g.pin(context.Background(), sharedKey, uuid.Nil, noopOpen)
	require.NoError(t, err)
	shared.start = shared.start.Add(-4 * time.Second)
	shared.release()

	for key, want := range map[string]float64{tenantA.String(): 5, tenantB.String(): 1, sharedKey: 4} {
		got, ok := heldSeconds(t, g.poolName, key)
		require.True(t, ok, key)
		require.InDelta(t, want, got, 0.1, key)
	}

	// a key idle for accountIdleTimeout loses its series at the next sweep; an active one keeps it
	g.mu.Lock()
	g.accounts[tenantB.String()].lastUsed = time.Now().Add(-accountIdleTimeout)
	g.lastSweep = time.Time{}
	g.mu.Unlock()

	spend(t, g, tenantA, time.Second)

	_, ok := heldSeconds(t, g.poolName, tenantB.String())
	require.False(t, ok)

	got, ok := heldSeconds(t, g.poolName, tenantA.String())
	require.True(t, ok)
	require.InDelta(t, 6, got, 0.1)

	// a key that comes back after the sweep starts a new series
	spend(t, g, tenantB, time.Second)

	got, ok = heldSeconds(t, g.poolName, tenantB.String())
	require.True(t, ok)
	require.InDelta(t, 1, got, 0.1)
}

// BenchmarkGateCheckout is the gate's own cost per checkout, with no connection: what the always-on
// gate adds to every ForTenant call.
func BenchmarkGateCheckout(b *testing.B) {
	tenantID := uuid.New()
	key := tenantID.String()

	for _, bc := range []struct {
		name string
		rate float64
	}{
		{"uncapped", 0},
		{"budget", 1},
	} {
		b.Run(bc.name, func(b *testing.B) {
			g := newBudgetGate(bc.rate)
			b.ReportAllocs()

			for b.Loop() {
				hold, err := g.pin(context.Background(), key, tenantID, noopOpen)
				if err != nil {
					b.Fatal(err)
				}

				hold.release()
			}
		})
	}
}
